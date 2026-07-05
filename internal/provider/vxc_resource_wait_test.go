package provider

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

func TestWaitForVXCProvision_PendingExternalApproval(t *testing.T) {
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{
			ProvisioningStatus: "DEPLOYABLE",
			VXCApproval: &megaport.VXCApproval{
				Status:  vxcApprovalPendingExternal,
				Message: "Partner Org",
				Type:    "NEW",
			},
		},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.Error(t, err)
	var pending *vxcPendingApprovalError
	require.ErrorAs(t, err, &pending)
	assert.Equal(t, vxcApprovalPendingExternal, pending.approval.Status)
	assert.Equal(t, "Partner Org", pending.approval.Message)
}

func TestWaitForVXCProvision_PendingInternalApproval(t *testing.T) {
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{
			ProvisioningStatus: "DEPLOYABLE",
			VXCApproval:        &megaport.VXCApproval{Status: vxcApprovalPendingInternal},
		},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	var pending *vxcPendingApprovalError
	require.ErrorAs(t, err, &pending)
	assert.Equal(t, vxcApprovalPendingInternal, pending.approval.Status)
}

func TestWaitForVXCProvision_ReadyWinsOverStaleApproval(t *testing.T) {
	// The API keeps returning the vxcApproval object after approval; a ready
	// provisioning status must take precedence over it.
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{
			ProvisioningStatus: megaport.SERVICE_LIVE,
			VXCApproval:        &megaport.VXCApproval{Status: vxcApprovalPendingExternal},
		},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.NoError(t, err)
}

func TestWaitForVXCProvision_EmptyApprovalDoesNotTriggerPending(t *testing.T) {
	// The API returns a vxcApproval object with empty fields when no approval
	// is in play; it must not be mistaken for a pending order.
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{
			ProvisioningStatus: "DEPLOYABLE",
			VXCApproval:        &megaport.VXCApproval{},
		},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", 20*time.Millisecond, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "time expired")
}

func TestWaitForVXCProvision_FailedIsTerminal(t *testing.T) {
	var calls atomic.Int32
	r := waitTestResource(&MockVXCService{
		GetVXCFunc: func(ctx context.Context, id string) (*megaport.VXC, error) {
			calls.Add(1)
			return &megaport.VXC{ProvisioningStatus: vxcStatusFailed}, nil
		},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to provision")
	assert.Equal(t, int32(1), calls.Load(), "FAILED should end the wait on the first poll")
}

func TestWaitForVXCProvision_CancelledParentIsTerminal(t *testing.T) {
	var calls atomic.Int32
	r := waitTestResource(&MockVXCService{
		GetVXCFunc: func(ctx context.Context, id string) (*megaport.VXC, error) {
			calls.Add(1)
			return &megaport.VXC{ProvisioningStatus: vxcStatusCancelledParent}, nil
		},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "terminal state")
	assert.Equal(t, int32(1), calls.Load(), "CANCELLED_PARENT should end the wait on the first poll")
}

func TestWaitForVXCProvision_FailedWinsOverStalePendingApproval(t *testing.T) {
	// A rejected or expired order must error even if a stale PENDING_*
	// approval object accompanies the FAILED status.
	r := waitTestResource(&MockVXCService{
		GetVXCResult: &megaport.VXC{
			ProvisioningStatus: vxcStatusFailed,
			VXCApproval:        &megaport.VXCApproval{Status: vxcApprovalPendingExternal},
		},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	require.Error(t, err)
	var pending *vxcPendingApprovalError
	assert.False(t, errors.As(err, &pending), "FAILED must not be reported as pending approval")
	assert.Contains(t, err.Error(), "failed to provision")
}

func TestWaitForVXCProvision_PendingAppearsOnLaterPoll(t *testing.T) {
	var calls atomic.Int32
	r := waitTestResource(&MockVXCService{
		GetVXCFunc: func(ctx context.Context, id string) (*megaport.VXC, error) {
			switch calls.Add(1) {
			case 1:
				return nil, errors.New("transient read error")
			case 2:
				return &megaport.VXC{ProvisioningStatus: "DEPLOYABLE", VXCApproval: &megaport.VXCApproval{}}, nil
			default:
				return &megaport.VXC{
					ProvisioningStatus: "DEPLOYABLE",
					VXCApproval:        &megaport.VXCApproval{Status: vxcApprovalPendingExternal},
				}, nil
			}
		},
	})

	err := r.waitForVXCProvision(context.Background(), "test-uid", time.Second, time.Millisecond)
	var pending *vxcPendingApprovalError
	require.ErrorAs(t, err, &pending)
	assert.GreaterOrEqual(t, calls.Load(), int32(3))
}

// stubProductService answers GetProductType for Create's product-type probe.
// Other ProductService methods panic if reached.
type stubProductService struct {
	megaport.ProductService
}

func (s *stubProductService) GetProductType(ctx context.Context, productUID string) (string, error) {
	return megaport.PRODUCT_MEGAPORT, nil
}

// TestVXCCreate_PendingApprovalCompletesWithWarning drives Create end to end
// through the pending-approval branch: the apply must succeed with a warning
// and the half-provisioned VXC written to state.
func TestVXCCreate_PendingApprovalCompletesWithWarning(t *testing.T) {
	ctx := context.Background()

	mock := &MockVXCService{
		BuyVXCResult: &megaport.BuyVXCResponse{TechnicalServiceUID: "vxc-uid-123"},
		GetVXCResult: &megaport.VXC{
			UID:                "vxc-uid-123",
			Name:               "pending-vxc",
			RateLimit:          1000,
			ContractTermMonths: 12,
			ProvisioningStatus: "DEPLOYABLE",
			VXCApproval: &megaport.VXCApproval{
				Status:  vxcApprovalPendingExternal,
				Message: "Partner Org",
				Type:    "NEW",
			},
		},
		ListVXCResourceTagsResult: map[string]string{},
	}
	r := &vxcResource{client: &megaport.Client{VXCService: mock, ProductService: &stubProductService{}}}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	schemaObjType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok, "schema type is not tftypes.Object")

	endVal := func(attr string, uid string) tftypes.Value {
		endObjType, ok := schemaObjType.AttributeTypes[attr].(tftypes.Object)
		require.True(t, ok, "%s type is not tftypes.Object", attr)
		attrs := nullValueMap(endObjType)
		attrs["requested_product_uid"] = tftypes.NewValue(tftypes.String, uid)
		return tftypes.NewValue(endObjType, attrs)
	}

	planAttrs := nullValueMap(schemaObjType)
	planAttrs["product_name"] = tftypes.NewValue(tftypes.String, "pending-vxc")
	planAttrs["rate_limit"] = tftypes.NewValue(tftypes.Number, 1000)
	planAttrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
	planAttrs["a_end"] = endVal("a_end", "port-a-uid")
	planAttrs["b_end"] = endVal("b_end", "port-b-uid")

	req := fwresource.CreateRequest{
		Plan: tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(schemaObjType, planAttrs)},
	}
	resp := fwresource.CreateResponse{
		State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(schemaObjType, nil)},
	}

	r.Create(ctx, req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "expected no errors, got: %v", resp.Diagnostics.Errors())
	var pendingWarnings []string
	for _, w := range resp.Diagnostics.Warnings() {
		if w.Summary() == "VXC pending approval" {
			pendingWarnings = append(pendingWarnings, w.Detail())
		}
	}
	require.Len(t, pendingWarnings, 1)
	assert.Contains(t, pendingWarnings[0], "approved by Partner Org")

	var uid, status string
	require.False(t, resp.State.GetAttribute(ctx, path.Root("product_uid"), &uid).HasError())
	assert.Equal(t, "vxc-uid-123", uid)
	require.False(t, resp.State.GetAttribute(ctx, path.Root("provisioning_status"), &status).HasError())
	assert.Equal(t, "DEPLOYABLE", status)
}

func TestVXCPendingApprovalWarning_ExternalVsInternal(t *testing.T) {
	external := vxcPendingApprovalWarning("vxc-name", "vxc-uid", megaport.VXCApproval{
		Status:  vxcApprovalPendingExternal,
		Message: "Partner Org",
	})
	assert.Contains(t, external, "waiting for the connection to be approved by Partner Org")

	// PENDING_INTERNAL must tell the user their own org approves, not imply a
	// third party will, otherwise they wait on something only they can do.
	internal := vxcPendingApprovalWarning("vxc-name", "vxc-uid", megaport.VXCApproval{
		Status:  vxcApprovalPendingInternal,
		Message: "My Own Org",
	})
	assert.Contains(t, internal, "requires approval from your own organization")
	assert.Contains(t, internal, "My Own Org")
	assert.NotContains(t, internal, "waiting for the connection to be approved by")
}

// TestVXCRead_FailedStatusWarns checks that a Read of a VXC reporting FAILED
// keeps the resource in state and emits a warning rather than erroring or
// removing it.
func TestVXCRead_FailedStatusWarns(t *testing.T) {
	ctx := context.Background()

	mock := &MockVXCService{
		GetVXCResult: &megaport.VXC{
			UID:                "vxc-uid-123",
			Name:               "failed-vxc",
			RateLimit:          1000,
			ContractTermMonths: 12,
			ProvisioningStatus: vxcStatusFailed,
		},
		ListVXCResourceTagsResult: map[string]string{},
	}
	r := &vxcResource{client: &megaport.Client{VXCService: mock, ProductService: &stubProductService{}}}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	schemaObjType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok, "schema type is not tftypes.Object")

	stateAttrs := nullValueMap(schemaObjType)
	stateAttrs["product_uid"] = tftypes.NewValue(tftypes.String, "vxc-uid-123")
	stateAttrs["product_name"] = tftypes.NewValue(tftypes.String, "failed-vxc")
	stateAttrs["rate_limit"] = tftypes.NewValue(tftypes.Number, 1000)
	stateAttrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
	stateVal := tftypes.NewValue(schemaObjType, stateAttrs)

	req := fwresource.ReadRequest{
		State: tfsdk.State{Schema: s, Raw: stateVal},
	}
	resp := fwresource.ReadResponse{
		State: tfsdk.State{Schema: s, Raw: stateVal},
	}

	r.Read(ctx, req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "expected no errors, got: %v", resp.Diagnostics.Errors())
	var failedWarnings []string
	for _, w := range resp.Diagnostics.Warnings() {
		if w.Summary() == "VXC failed to provision" {
			failedWarnings = append(failedWarnings, w.Detail())
		}
	}
	require.Len(t, failedWarnings, 1)
	assert.Contains(t, failedWarnings[0], "status FAILED")

	// The resource must remain in state, not be removed.
	var uid string
	require.False(t, resp.State.GetAttribute(ctx, path.Root("product_uid"), &uid).HasError())
	assert.Equal(t, "vxc-uid-123", uid)
}

// TestVXCUpdate_PendingApprovalErrorIsEnriched checks that when the API
// rejects an update because the order still awaits approval, the error
// diagnostic points at the approval workflow.
func TestVXCUpdate_PendingApprovalErrorIsEnriched(t *testing.T) {
	ctx := context.Background()

	mock := &MockVXCService{
		UpdateVXCErr: errors.New("400: VXC is pending processing and requires approval to modify network attributes"),
		GetVXCResult: &megaport.VXC{
			UID:                "vxc-uid-123",
			ProvisioningStatus: "DEPLOYABLE",
			VXCApproval:        &megaport.VXCApproval{Status: vxcApprovalPendingExternal, Message: "Partner Org", Type: vxcApprovalTypeNew},
		},
	}
	r := &vxcResource{client: &megaport.Client{VXCService: mock, ProductService: &stubProductService{}}}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	schemaObjType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok, "schema type is not tftypes.Object")

	makeVal := func(rateLimit int64) tftypes.Value {
		attrs := nullValueMap(schemaObjType)
		attrs["product_uid"] = tftypes.NewValue(tftypes.String, "vxc-uid-123")
		attrs["product_name"] = tftypes.NewValue(tftypes.String, "pending-vxc")
		attrs["rate_limit"] = tftypes.NewValue(tftypes.Number, rateLimit)
		attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
		for _, end := range []string{"a_end", "b_end"} {
			endObjType, ok := schemaObjType.AttributeTypes[end].(tftypes.Object)
			require.True(t, ok, "%s type is not tftypes.Object", end)
			endAttrs := nullValueMap(endObjType)
			endAttrs["requested_product_uid"] = tftypes.NewValue(tftypes.String, "port-"+end)
			attrs[end] = tftypes.NewValue(endObjType, endAttrs)
		}
		return tftypes.NewValue(schemaObjType, attrs)
	}

	req := fwresource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: s, Raw: makeVal(2000)},
		State: tfsdk.State{Schema: s, Raw: makeVal(1000)},
	}
	resp := fwresource.UpdateResponse{
		State: tfsdk.State{Schema: s, Raw: makeVal(1000)},
	}

	r.Update(ctx, req, &resp)

	require.True(t, resp.Diagnostics.HasError())
	var detail string
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Error Updating VXC" {
			detail = d.Detail()
		}
	}
	assert.Contains(t, detail, "requires approval to modify network attributes")
	assert.Contains(t, detail, "still pending approval ("+vxcApprovalPendingExternal+")")
}

// TestVXCUpdate_SpeedChangeApprovalDoesNotEnrich guards against attributing an
// update failure to a pending new-order approval when the pending approval is
// actually a speed change on an already-live VXC.
func TestVXCUpdate_SpeedChangeApprovalDoesNotEnrich(t *testing.T) {
	ctx := context.Background()

	mock := &MockVXCService{
		UpdateVXCErr: errors.New("400: some unrelated update failure"),
		GetVXCResult: &megaport.VXC{
			UID:                "vxc-uid-123",
			ProvisioningStatus: megaport.SERVICE_LIVE,
			VXCApproval:        &megaport.VXCApproval{Status: vxcApprovalPendingExternal, Type: "SPEED_CHANGE"},
		},
	}
	r := &vxcResource{client: &megaport.Client{VXCService: mock, ProductService: &stubProductService{}}}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	schemaObjType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok, "schema type is not tftypes.Object")

	makeVal := func(rateLimit int64) tftypes.Value {
		attrs := nullValueMap(schemaObjType)
		attrs["product_uid"] = tftypes.NewValue(tftypes.String, "vxc-uid-123")
		attrs["product_name"] = tftypes.NewValue(tftypes.String, "live-vxc")
		attrs["rate_limit"] = tftypes.NewValue(tftypes.Number, rateLimit)
		attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
		for _, end := range []string{"a_end", "b_end"} {
			endObjType, ok := schemaObjType.AttributeTypes[end].(tftypes.Object)
			require.True(t, ok, "%s type is not tftypes.Object", end)
			endAttrs := nullValueMap(endObjType)
			endAttrs["requested_product_uid"] = tftypes.NewValue(tftypes.String, "port-"+end)
			attrs[end] = tftypes.NewValue(endObjType, endAttrs)
		}
		return tftypes.NewValue(schemaObjType, attrs)
	}

	req := fwresource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: s, Raw: makeVal(2000)},
		State: tfsdk.State{Schema: s, Raw: makeVal(1000)},
	}
	resp := fwresource.UpdateResponse{
		State: tfsdk.State{Schema: s, Raw: makeVal(1000)},
	}

	r.Update(ctx, req, &resp)

	require.True(t, resp.Diagnostics.HasError())
	var detail string
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Error Updating VXC" {
			detail = d.Detail()
		}
	}
	assert.Contains(t, detail, "some unrelated update failure")
	assert.NotContains(t, detail, "still pending approval")
}
