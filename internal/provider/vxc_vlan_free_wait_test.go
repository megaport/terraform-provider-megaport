package provider

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sequencePortService answers each availability check from check, which
// receives the 1-based call number.
type sequencePortService struct {
	megaport.PortService
	calls atomic.Int32
	check func(call int32) (bool, error)
}

func (s *sequencePortService) CheckPortVLANAvailability(_ context.Context, _ string, _ int) (bool, error) {
	return s.check(s.calls.Add(1))
}

// setFreeWait sets the VLAN free wait budget and a 1ms poll for one test.
// Tests that call it must not run in parallel, because both are package state.
func setFreeWait(t *testing.T, timeout time.Duration) {
	t.Helper()
	prevTimeout, prevInterval := waitForTime, vlanFreePollInterval
	waitForTime, vlanFreePollInterval = timeout, time.Millisecond
	t.Cleanup(func() { waitForTime, vlanFreePollInterval = prevTimeout, prevInterval })
}

func recordDestroyed(t *testing.T, portUID string, vlan int) {
	t.Helper()
	key := destroyedVLAN{portUID: portUID, vlan: vlan}
	destroyedVLANs.Store(key, struct{}{})
	t.Cleanup(func() { destroyedVLANs.Delete(key) })
}

func TestWaitForVLANFree_FreesAfterPolls(t *testing.T) {
	svc := &sequencePortService{check: func(call int32) (bool, error) { return call >= 3, nil }}

	err := waitForVLANFree(context.Background(), svc, "port-uid", 920, time.Second, time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int32(3), svc.calls.Load())
}

func TestWaitForVLANFree_RetriesTransientErrors(t *testing.T) {
	svc := &sequencePortService{check: func(call int32) (bool, error) {
		if call <= 2 {
			return false, errors.New("502 bad gateway")
		}
		return true, nil
	}}

	err := waitForVLANFree(context.Background(), svc, "port-uid", 920, time.Second, time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int32(3), svc.calls.Load())
}

func TestWaitForVLANFree_TimesOut(t *testing.T) {
	svc := &sequencePortService{check: func(int32) (bool, error) { return false, nil }}

	err := waitForVLANFree(context.Background(), svc, "port-uid", 920, 20*time.Millisecond, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "still in use")
}

func TestWaitForVLANFree_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := &sequencePortService{check: func(int32) (bool, error) { return false, nil }}

	err := waitForVLANFree(ctx, svc, "port-uid", 920, time.Second, time.Millisecond)
	assert.ErrorIs(t, err, context.Canceled)
}

func destroyedPreflightInput(svc megaport.PortService, portUID string) vlanPreflightInput {
	return vlanPreflightInput{
		svc:         svc,
		end:         "A-End",
		productUID:  portUID,
		productType: megaport.PRODUCT_MEGAPORT,
		orderedVLAN: types.Int64Value(920),
		currentVLAN: types.Int64Null(),
	}
}

func TestVLANAvailabilityPreflight_WaitsForDestroyedVLAN(t *testing.T) {
	setFreeWait(t, time.Second)
	recordDestroyed(t, "port-destroyed-frees", 920)
	svc := &sequencePortService{check: func(call int32) (bool, error) { return call >= 3, nil }}

	diags := vlanAvailabilityPreflight(context.Background(), destroyedPreflightInput(svc, "port-destroyed-frees"))
	assert.False(t, diags.HasError(), "unexpected error: %v", diags)
	assert.Equal(t, int32(3), svc.calls.Load())
}

func TestVLANAvailabilityPreflight_DestroyedVLANNeverFrees(t *testing.T) {
	setFreeWait(t, 20*time.Millisecond)
	recordDestroyed(t, "port-destroyed-held", 920)
	svc := &sequencePortService{check: func(int32) (bool, error) { return false, nil }}

	diags := vlanAvailabilityPreflight(context.Background(), destroyedPreflightInput(svc, "port-destroyed-held"))
	require.True(t, diags.HasError())
	assert.Equal(t, "VLAN 920 is not available on the A-End port", diags.Errors()[0].Summary())
	detail := diags.Errors()[0].Detail()
	assert.Contains(t, detail, "port-destroyed-held")
	assert.Contains(t, detail, "still in use")
	assert.Contains(t, detail, "ordered_vlan")
}

// A VLAN that no VXC destroyed in this process held belongs to a live service, so the
// preflight fails on the first answer.
func TestVLANAvailabilityPreflight_UndestroyedVLANFailsWithoutWaiting(t *testing.T) {
	setFreeWait(t, 200*time.Millisecond)
	recordDestroyed(t, "port-unrelated", 921)
	svc := &sequencePortService{check: func(int32) (bool, error) { return false, nil }}

	diags := vlanAvailabilityPreflight(context.Background(), destroyedPreflightInput(svc, "port-unrelated"))
	require.True(t, diags.HasError())
	assert.Equal(t, int32(1), svc.calls.Load())
	assert.Contains(t, diags.Errors()[0].Detail(), "already in use")
}

// Two creates that pin the same destroyed VLAN cannot both hold it, so only the
// first waits.
func TestVLANAvailabilityPreflight_SecondPinOnDestroyedVLANFailsWithoutWaiting(t *testing.T) {
	setFreeWait(t, 200*time.Millisecond)
	recordDestroyed(t, "port-destroyed-twice", 920)
	first := &sequencePortService{check: func(call int32) (bool, error) { return call >= 2, nil }}
	require.False(t, vlanAvailabilityPreflight(context.Background(), destroyedPreflightInput(first, "port-destroyed-twice")).HasError())

	second := &sequencePortService{check: func(int32) (bool, error) { return false, nil }}
	diags := vlanAvailabilityPreflight(context.Background(), destroyedPreflightInput(second, "port-destroyed-twice"))
	require.True(t, diags.HasError())
	assert.Equal(t, int32(1), second.calls.Load())
	assert.Contains(t, diags.Errors()[0].Detail(), "already in use")
}

func TestWaitForVLANFree_TimeoutReportsLastError(t *testing.T) {
	svc := &sequencePortService{check: func(int32) (bool, error) { return false, errors.New("502 bad gateway") }}

	err := waitForVLANFree(context.Background(), svc, "port-uid", 920, 20*time.Millisecond, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502 bad gateway")
}

func TestWaitForVLANFree_TakenAnswerClearsEarlierError(t *testing.T) {
	svc := &sequencePortService{check: func(call int32) (bool, error) {
		if call == 1 {
			return false, errors.New("502 bad gateway")
		}
		return false, nil
	}}

	err := waitForVLANFree(context.Background(), svc, "port-uid", 920, 20*time.Millisecond, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "still in use")
}

// A destroyed VLAN that already reads free needs no wait, and a later pin on it
// must not wait either.
func TestVLANAvailabilityPreflight_FreeDestroyedVLANDropsRecord(t *testing.T) {
	setFreeWait(t, time.Second)
	recordDestroyed(t, "port-destroyed-free", 920)
	svc := &sequencePortService{check: func(int32) (bool, error) { return true, nil }}

	require.False(t, vlanAvailabilityPreflight(context.Background(), destroyedPreflightInput(svc, "port-destroyed-free")).HasError())
	_, recorded := destroyedVLANs.Load(destroyedVLAN{portUID: "port-destroyed-free", vlan: 920})
	assert.False(t, recorded)
}

// deleteVXC runs Delete against a state built from the two ends.
func deleteVXC(t *testing.T, b *vxcValueBuilder, mock *MockVXCService, aEnd, bEnd vxcEndSpec) fwresource.DeleteResponse {
	t.Helper()
	attrs := map[string]tftypes.Value{}
	require.NoError(t, b.vxc(b.end(aEnd), b.end(bEnd), nil).As(&attrs))
	attrs["product_uid"] = tftypes.NewValue(tftypes.String, "vxc-old")
	state := tftypes.NewValue(b.objType, attrs)

	resp := fwresource.DeleteResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
	waitTestResource(mock).Delete(context.Background(), fwresource.DeleteRequest{
		State: tfsdk.State{Schema: b.schema, Raw: state},
	}, &resp)
	return resp
}

func TestVXCDelete_RecordsDestroyedVLANs(t *testing.T) {
	b := newVXCValueBuilder(t)
	t.Cleanup(func() {
		destroyedVLANs.Delete(destroyedVLAN{portUID: "port-del-a", vlan: 920})
		destroyedVLANs.Delete(destroyedVLAN{portUID: "port-del-b", vlan: 0})
	})

	resp := deleteVXC(t, b, &MockVXCService{GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.STATUS_DECOMMISSIONED}},
		vxcEndSpec{productUID: "port-del-a", currentUID: "port-del-a", vlan: int64p(920)},
		vxcEndSpec{productUID: "port-del-b", currentUID: "port-del-b", vlan: int64p(0)},
	)
	require.False(t, resp.Diagnostics.HasError(), "unexpected error: %v", resp.Diagnostics)

	_, aRecorded := destroyedVLANs.Load(destroyedVLAN{portUID: "port-del-a", vlan: 920})
	assert.True(t, aRecorded, "the A-End VLAN should be recorded")
	_, bRecorded := destroyedVLANs.Load(destroyedVLAN{portUID: "port-del-b", vlan: 0})
	assert.False(t, bRecorded, "an auto-assigned VLAN of 0 should not be recorded")
}

// A VXC that never reached DECOMMISSIONED may still hold its VLAN, so a later
// create must not wait on it.
func TestVXCDelete_DoesNotRecordWhenNotDecommissioned(t *testing.T) {
	setFreeWait(t, 0)
	b := newVXCValueBuilder(t)
	t.Cleanup(func() { destroyedVLANs.Delete(destroyedVLAN{portUID: "port-del-live", vlan: 920}) })

	resp := deleteVXC(t, b, &MockVXCService{GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.STATUS_CANCELLED}},
		vxcEndSpec{productUID: "port-del-live", currentUID: "port-del-live", vlan: int64p(920)},
		vxcEndSpec{productUID: "port-del-other", currentUID: "port-del-other"},
	)
	require.True(t, resp.Diagnostics.HasError())

	_, recorded := destroyedVLANs.Load(destroyedVLAN{portUID: "port-del-live", vlan: 920})
	assert.False(t, recorded)
}

// A replacement destroys the old VXC and then creates the new one with the same
// pinned VLAN while the port still reports it taken.
func TestVXCReplace_CreateWaitsForVLANFreedByDelete(t *testing.T) {
	setFreeWait(t, time.Second)
	b := newVXCValueBuilder(t)
	t.Cleanup(func() {
		destroyedVLANs.Delete(destroyedVLAN{portUID: "port-replace", vlan: 920})
		destroyedVLANs.Delete(destroyedVLAN{portUID: "port-replace-b", vlan: 100})
	})

	delResp := deleteVXC(t, b, &MockVXCService{GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.STATUS_DECOMMISSIONED}},
		vxcEndSpec{productUID: "port-replace", currentUID: "port-replace", orderedVLAN: int64p(920), vlan: int64p(920)},
		vxcEndSpec{productUID: "port-replace-b", currentUID: "port-replace-b", vlan: int64p(100)},
	)
	require.False(t, delResp.Diagnostics.HasError(), "unexpected error: %v", delResp.Diagnostics)

	ps := newPreflightServer(t, map[string]int{"port-replace": 920})
	ps.mu.Lock()
	ps.freeFromQuery = 3
	ps.mu.Unlock()

	plan := b.vxc(
		b.end(vxcEndSpec{productUID: "port-replace", orderedVLAN: int64p(920)}),
		b.end(vxcEndSpec{productUID: "port-replace-b", orderedVLAN: int64p(100)}),
		nil,
	)
	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: b.schema}}
	ps.resource(t).Create(context.Background(), fwresource.CreateRequest{Plan: tfsdk.Plan{Schema: b.schema, Raw: plan}}, &resp)

	for _, d := range resp.Diagnostics.Errors() {
		assert.False(t, strings.Contains(d.Summary(), "is not available"), "unexpected VLAN error: %s", d.Summary())
	}
	var aEndQueries int
	for _, q := range ps.vlanQueries {
		if q == (vlanQuery{"port-replace", "920"}) {
			aEndQueries++
		}
	}
	assert.Equal(t, 3, aEndQueries, "the A-End VLAN should be polled until it frees")
}
