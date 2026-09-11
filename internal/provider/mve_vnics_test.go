package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

func TestFromAPIMVE_NoVnicsIsAnError(t *testing.T) {
	var model mveResourceModel
	diags := model.fromAPIMVE(context.Background(), &megaport.MVE{UID: "mve-1"}, nil)

	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "mve-1")
	assert.Equal(t, "mve-1", model.UID.ValueString(), "fields before the vnics check still land in state")
}

func TestFromAPIMVE_WithVnics(t *testing.T) {
	var model mveResourceModel
	mve := &megaport.MVE{
		UID:               "mve-1",
		NetworkInterfaces: []*megaport.MVENetworkInterface{{Description: "Data Plane", VLAN: 100}},
	}
	diags := model.fromAPIMVE(context.Background(), mve, nil)

	require.False(t, diags.HasError(), diags)
	assert.Len(t, model.NetworkInterfaces.Elements(), 1)
}

func shortenVnicRetryInterval(t *testing.T) {
	t.Helper()
	previous := mveVnicRetryInterval
	mveVnicRetryInterval = time.Millisecond
	t.Cleanup(func() { mveVnicRetryInterval = previous })
}

func TestGetMVEWithVnics_RereadsWhileVnicsAreMissing(t *testing.T) {
	shortenVnicRetryInterval(t)
	calls := 0
	svc := &MockMVEService{GetMVEFunc: func(ctx context.Context, mveID string) (*megaport.MVE, error) {
		calls++
		if calls == 1 {
			return &megaport.MVE{UID: mveID}, nil
		}
		return &megaport.MVE{UID: mveID, NetworkInterfaces: []*megaport.MVENetworkInterface{{Description: "Data Plane"}}}, nil
	}}

	mve, err := getMVEWithVnics(context.Background(), svc, "mve-1")

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	assert.Len(t, mve.NetworkInterfaces, 1)
}

func TestGetMVEWithVnics_ReturnsTheLastReadAfterRetries(t *testing.T) {
	shortenVnicRetryInterval(t)
	calls := 0
	svc := &MockMVEService{GetMVEFunc: func(ctx context.Context, mveID string) (*megaport.MVE, error) {
		calls++
		return &megaport.MVE{UID: mveID}, nil
	}}

	mve, err := getMVEWithVnics(context.Background(), svc, "mve-1")

	require.NoError(t, err)
	assert.Equal(t, 1+mveVnicRetries, calls)
	assert.Empty(t, mve.NetworkInterfaces, "the caller's fromAPIMVE turns this into the error")
}

func TestGetMVEWithVnics_DoesNotRetryAReadError(t *testing.T) {
	shortenVnicRetryInterval(t)
	calls := 0
	svc := &MockMVEService{GetMVEFunc: func(ctx context.Context, mveID string) (*megaport.MVE, error) {
		calls++
		return nil, errors.New("boom")
	}}

	_, err := getMVEWithVnics(context.Background(), svc, "mve-1")

	require.EqualError(t, err, "boom")
	assert.Equal(t, 1, calls)
}
