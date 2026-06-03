package provider

import (
	"context"
	"errors"
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
