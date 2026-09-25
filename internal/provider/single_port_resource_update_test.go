// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

// singlePortUpdateService answers the port calls Update makes. GetPort returns the port as
// the API holds it after the modify.
type singlePortUpdateService struct {
	megaport.PortService

	port      megaport.Port
	modifyErr error
	modified  []megaport.ModifyPortRequest
}

func (s *singlePortUpdateService) ModifyPort(_ context.Context, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
	s.modified = append(s.modified, *req)
	if s.modifyErr != nil {
		return nil, s.modifyErr
	}
	return &megaport.ModifyPortResponse{IsUpdated: true}, nil
}

func (s *singlePortUpdateService) GetPort(_ context.Context, _ string) (*megaport.Port, error) {
	port := s.port
	return &port, nil
}

func (s *singlePortUpdateService) ListPortResourceTags(_ context.Context, _ string) (map[string]string, error) {
	return nil, nil
}

func TestSinglePortUpdate_PendingApproval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modifyErr   error
		apiPort     megaport.Port
		wantWarning bool
	}{
		{
			name:        "a term increase pending approval completes with a warning and keeps the plan",
			modifyErr:   megaport.ErrModifyPendingApproval,
			apiPort:     megaport.Port{UID: "port-1", Name: "port-old", ContractTermMonths: 12, CostCentre: "cc-1"},
			wantWarning: true,
		},
		{
			name:    "a term increase with no approval completes with no warning",
			apiPort: megaport.Port{UID: "port-1", Name: "port-new", ContractTermMonths: 24, CostCentre: "cc-2", MarketplaceVisibility: true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			svc := &singlePortUpdateService{port: tc.apiPort, modifyErr: tc.modifyErr}
			r := &portResource{client: &megaport.Client{PortService: svc}}

			schemaResp := fwresource.SchemaResponse{}
			r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

			stateModel := singlePortUpdateTestModel("port-old", 12, "cc-1", false)
			planModel := singlePortUpdateTestModel("port-new", 24, "cc-2", true)
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
			if diags := state.Set(ctx, &stateModel); diags.HasError() {
				t.Fatalf("building state: %v", diags.Errors())
			}
			if diags := plan.Set(ctx, &planModel); diags.HasError() {
				t.Fatalf("building plan: %v", diags.Errors())
			}

			resp := fwresource.UpdateResponse{State: state}
			r.Update(ctx, fwresource.UpdateRequest{State: state, Plan: plan}, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
			}
			if len(svc.modified) != 1 {
				t.Fatalf("got %d modify calls, want 1", len(svc.modified))
			}
			if term := svc.modified[0].ContractTermMonths; term == nil || *term != 24 {
				t.Errorf("modify sent term %v, want 24", term)
			}

			warnings := resp.Diagnostics.Warnings()
			switch {
			case !tc.wantWarning && len(warnings) > 0:
				t.Errorf("unexpected warnings: %v", warnings)
			case tc.wantWarning && len(warnings) != 1:
				t.Fatalf("got %d warnings (%v), want 1", len(warnings), warnings)
			case tc.wantWarning && !strings.Contains(warnings[0].Detail(), "term increase on port port-1 needs order approval"):
				t.Errorf("warning %q does not name the port", warnings[0].Detail())
			}

			var got singlePortResourceModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("reading state: %v", diags.Errors())
			}
			if !got.Name.Equal(planModel.Name) || !got.ContractTermMonths.Equal(planModel.ContractTermMonths) ||
				!got.CostCentre.Equal(planModel.CostCentre) || !got.MarketplaceVisibility.Equal(planModel.MarketplaceVisibility) {
				t.Errorf("state holds name %s, term %s, cost centre %s, visibility %s; want the planned values",
					got.Name, got.ContractTermMonths, got.CostCentre, got.MarketplaceVisibility)
			}
		})
	}
}

func singlePortUpdateTestModel(name string, term int64, costCentre string, visibility bool) singlePortResourceModel {
	return singlePortResourceModel{
		UID:                   types.StringValue("port-1"),
		Name:                  types.StringValue(name),
		ContractTermMonths:    types.Int64Value(term),
		CostCentre:            types.StringValue(costCentre),
		MarketplaceVisibility: types.BoolValue(visibility),
		PortSpeed:             types.Int64Value(10000),
		LocationID:            types.Int64Value(5),
		Resources:             types.ObjectNull(portResourcesAttrs),
		ResourceTags:          types.MapNull(types.StringType),
	}
}
