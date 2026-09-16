package provider

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

func waitTestResource(mock *MockVXCService) *vxcResource {
	return &vxcResource{client: &megaport.Client{VXCService: mock}}
}

func TestWaitForVXCProvision_ReadyOnFirstPoll(t *testing.T) {
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.SERVICE_LIVE},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.NoError(t, err)
}

func TestWaitForVXCProvision_RetriesTransientErrors(t *testing.T) {
	var calls atomic.Int32
	r := waitTestResource(&MockVXCService{
		GetVXCFunc: func(ctx context.Context, id string) (*megaport.VXC, error) {
			if calls.Add(1) <= 2 {
				return nil, errors.New("invalid character '[' after object key:value pair")
			}
			return &megaport.VXC{ProvisioningStatus: megaport.SERVICE_CONFIGURED}, nil
		},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls.Load(), int32(3))
}

func TestWaitForVXCProvision_TerminalState(t *testing.T) {
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.STATUS_CANCELLED},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "terminal state")
}

func TestWaitForVXCProvision_TimesOut(t *testing.T) {
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{ProvisioningStatus: "DEPLOYABLE"},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", 20*time.Millisecond, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "time expired")
}

func TestWaitForVXCProvision_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{ProvisioningStatus: "DEPLOYABLE"},
	})

	err := r.waitForVXCProvision(ctx, "test-uid", time.Second, time.Millisecond)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWaitForVXCDecommission_DecommissionedOnFirstPoll(t *testing.T) {
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.STATUS_DECOMMISSIONED},
	})

	err := r.waitForVXCDecommission(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.NoError(t, err)
}

func TestWaitForVXCDecommission_RetriesTransientErrors(t *testing.T) {
	var calls atomic.Int32
	r := waitTestResource(&MockVXCService{
		GetVXCFunc: func(ctx context.Context, id string) (*megaport.VXC, error) {
			if calls.Add(1) <= 2 {
				return nil, errors.New("invalid character '[' after object key:value pair")
			}
			return &megaport.VXC{ProvisioningStatus: megaport.STATUS_DECOMMISSIONED}, nil
		},
	})

	err := r.waitForVXCDecommission(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls.Load(), int32(3))
}

func TestWaitForVXCDecommission_NotFoundIsGone(t *testing.T) {
	r := waitTestResource(&MockVXCService{
		GetVXCErr: &megaport.ErrorResponse{
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
				Request:    &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/v2/product/test-uid"}},
			},
			Message: "Not Found",
		},
	})

	err := r.waitForVXCDecommission(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.NoError(t, err)
}

// A VXC stuck in CANCELLED means the network termination failed, so the service
// is still live. The destroy has to fail and name the VXC and that status.
func TestWaitForVXCDecommission_TimesOutOnCancelled(t *testing.T) {
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.STATUS_CANCELLED},
	})

	err := r.waitForVXCDecommission(context.Background(), "test-uid", 20*time.Millisecond, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "time expired")
	assert.Contains(t, err.Error(), "test-uid")
	assert.Contains(t, err.Error(), megaport.STATUS_CANCELLED)
}

func TestWaitForVXCDecommission_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.SERVICE_LIVE},
	})

	err := r.waitForVXCDecommission(ctx, "test-uid", time.Second, time.Millisecond)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
