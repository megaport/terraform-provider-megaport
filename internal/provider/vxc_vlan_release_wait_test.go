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

// setReleaseWait sets the release wait budget and a 1ms poll for one test.
// Tests that call it must not run in parallel, because both are package state.
func setReleaseWait(t *testing.T, timeout time.Duration) {
	t.Helper()
	prevTimeout, prevInterval := waitForTime, vlanReleasePollInterval
	waitForTime, vlanReleasePollInterval = timeout, time.Millisecond
	t.Cleanup(func() { waitForTime, vlanReleasePollInterval = prevTimeout, prevInterval })
}

func recordReleased(t *testing.T, portUID string, vlan int) {
	t.Helper()
	key := releasedVLAN{portUID: portUID, vlan: vlan}
	releasedVLANs.Store(key, struct{}{})
	t.Cleanup(func() { releasedVLANs.Delete(key) })
}

func TestWaitForVLANRelease_FreesAfterPolls(t *testing.T) {
	svc := &sequencePortService{check: func(call int32) (bool, error) { return call >= 3, nil }}

	err := waitForVLANRelease(context.Background(), svc, "port-uid", 920, time.Second, time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int32(3), svc.calls.Load())
}

func TestWaitForVLANRelease_RetriesTransientErrors(t *testing.T) {
	svc := &sequencePortService{check: func(call int32) (bool, error) {
		if call <= 2 {
			return false, errors.New("502 bad gateway")
		}
		return true, nil
	}}

	err := waitForVLANRelease(context.Background(), svc, "port-uid", 920, time.Second, time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int32(3), svc.calls.Load())
}

func TestWaitForVLANRelease_TimesOut(t *testing.T) {
	svc := &sequencePortService{check: func(int32) (bool, error) { return false, nil }}

	err := waitForVLANRelease(context.Background(), svc, "port-uid", 920, 20*time.Millisecond, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "still in use")
}

func TestWaitForVLANRelease_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := &sequencePortService{check: func(int32) (bool, error) { return false, nil }}

	err := waitForVLANRelease(ctx, svc, "port-uid", 920, time.Second, time.Millisecond)
	assert.ErrorIs(t, err, context.Canceled)
}

func releasedPreflightInput(svc megaport.PortService, portUID string) vlanPreflightInput {
	return vlanPreflightInput{
		svc:         svc,
		end:         "A-End",
		productUID:  portUID,
		productType: megaport.PRODUCT_MEGAPORT,
		orderedVLAN: types.Int64Value(920),
		currentVLAN: types.Int64Null(),
	}
}

func TestVLANAvailabilityPreflight_WaitsForReleasedVLAN(t *testing.T) {
	setReleaseWait(t, time.Second)
	recordReleased(t, "port-released-frees", 920)
	svc := &sequencePortService{check: func(call int32) (bool, error) { return call >= 3, nil }}

	diags := vlanAvailabilityPreflight(context.Background(), releasedPreflightInput(svc, "port-released-frees"))
	assert.False(t, diags.HasError(), "unexpected error: %v", diags)
	assert.Equal(t, int32(3), svc.calls.Load())
}

func TestVLANAvailabilityPreflight_ReleasedVLANNeverFrees(t *testing.T) {
	setReleaseWait(t, 20*time.Millisecond)
	recordReleased(t, "port-released-held", 920)
	svc := &sequencePortService{check: func(int32) (bool, error) { return false, nil }}

	diags := vlanAvailabilityPreflight(context.Background(), releasedPreflightInput(svc, "port-released-held"))
	require.True(t, diags.HasError())
	assert.Equal(t, "VLAN 920 is not available on the A-End port", diags.Errors()[0].Summary())
	detail := diags.Errors()[0].Detail()
	assert.Contains(t, detail, "port-released-held")
	assert.Contains(t, detail, "still in use")
	assert.Contains(t, detail, "ordered_vlan")
}

// A VLAN no destroy in this process freed belongs to a live service, so the
// preflight fails on the first answer, even with a long wait budget.
func TestVLANAvailabilityPreflight_UnreleasedVLANFailsWithoutWaiting(t *testing.T) {
	setReleaseWait(t, time.Hour)
	recordReleased(t, "port-unrelated", 921)
	svc := &sequencePortService{check: func(int32) (bool, error) { return false, nil }}

	diags := vlanAvailabilityPreflight(context.Background(), releasedPreflightInput(svc, "port-unrelated"))
	require.True(t, diags.HasError())
	assert.Equal(t, int32(1), svc.calls.Load())
	assert.Contains(t, diags.Errors()[0].Detail(), "already in use")
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

func TestVXCDelete_RecordsReleasedVLANs(t *testing.T) {
	b := newVXCValueBuilder(t)
	t.Cleanup(func() {
		releasedVLANs.Delete(releasedVLAN{portUID: "port-del-a", vlan: 920})
		releasedVLANs.Delete(releasedVLAN{portUID: "port-del-b", vlan: 0})
	})

	resp := deleteVXC(t, b, &MockVXCService{GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.STATUS_DECOMMISSIONED}},
		vxcEndSpec{productUID: "port-del-a", currentUID: "port-del-a", vlan: int64p(920)},
		vxcEndSpec{productUID: "port-del-b", currentUID: "port-del-b", vlan: int64p(0)},
	)
	require.False(t, resp.Diagnostics.HasError(), "unexpected error: %v", resp.Diagnostics)

	_, aRecorded := releasedVLANs.Load(releasedVLAN{portUID: "port-del-a", vlan: 920})
	assert.True(t, aRecorded, "the A-End VLAN should be recorded")
	_, bRecorded := releasedVLANs.Load(releasedVLAN{portUID: "port-del-b", vlan: 0})
	assert.False(t, bRecorded, "an auto-assigned VLAN of 0 should not be recorded")
}

// A VXC that never reached DECOMMISSIONED may still hold its VLAN, so a later
// create must not wait on it.
func TestVXCDelete_DoesNotRecordWhenNotDecommissioned(t *testing.T) {
	setReleaseWait(t, 0)
	b := newVXCValueBuilder(t)
	t.Cleanup(func() { releasedVLANs.Delete(releasedVLAN{portUID: "port-del-live", vlan: 920}) })

	resp := deleteVXC(t, b, &MockVXCService{GetVXCResult: &megaport.VXC{ProvisioningStatus: megaport.STATUS_CANCELLED}},
		vxcEndSpec{productUID: "port-del-live", currentUID: "port-del-live", vlan: int64p(920)},
		vxcEndSpec{productUID: "port-del-other", currentUID: "port-del-other"},
	)
	require.True(t, resp.Diagnostics.HasError())

	_, recorded := releasedVLANs.Load(releasedVLAN{portUID: "port-del-live", vlan: 920})
	assert.False(t, recorded)
}

// A replacement destroys the old VXC and then creates the new one with the same
// pinned VLAN while the port still reports it taken.
func TestVXCReplace_CreateWaitsForVLANFreedByDelete(t *testing.T) {
	setReleaseWait(t, time.Second)
	b := newVXCValueBuilder(t)
	t.Cleanup(func() { releasedVLANs.Delete(releasedVLAN{portUID: "port-replace", vlan: 920}) })

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
