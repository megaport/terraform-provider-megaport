package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

// MockVXCService is a mock of the VXC service for testing
type MockVXCService struct {
	ListVXCsResult            []*megaport.VXC
	ListVXCsErr               error
	GetVXCResult              *megaport.VXC
	GetVXCErr                 error
	ListVXCResourceTagsFunc   func(ctx context.Context, vxcID string) (map[string]string, error)
	ListVXCResourceTagsErr    error
	ListVXCResourceTagsResult map[string]string
	CapturedResourceTagVXCUID string
}

func (m *MockVXCService) ListVXCs(ctx context.Context, req *megaport.ListVXCsRequest) ([]*megaport.VXC, error) {
	if m.ListVXCsErr != nil {
		return nil, m.ListVXCsErr
	}
	if m.ListVXCsResult != nil {
		return m.ListVXCsResult, nil
	}
	return []*megaport.VXC{}, nil
}

func (m *MockVXCService) GetVXC(ctx context.Context, id string) (*megaport.VXC, error) {
	if m.GetVXCErr != nil {
		return nil, m.GetVXCErr
	}
	return m.GetVXCResult, nil
}

func (m *MockVXCService) ListVXCResourceTags(ctx context.Context, vxcID string) (map[string]string, error) {
	m.CapturedResourceTagVXCUID = vxcID
	if m.ListVXCResourceTagsFunc != nil {
		return m.ListVXCResourceTagsFunc(ctx, vxcID)
	}
	if m.ListVXCResourceTagsErr != nil {
		return nil, m.ListVXCResourceTagsErr
	}
	if m.ListVXCResourceTagsResult != nil {
		return m.ListVXCResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the VXCService interface with minimal stubs
func (m *MockVXCService) BuyVXC(ctx context.Context, req *megaport.BuyVXCRequest) (*megaport.BuyVXCResponse, error) {
	return nil, nil
}

func (m *MockVXCService) ValidateVXCOrder(ctx context.Context, req *megaport.BuyVXCRequest) error {
	return nil
}

func (m *MockVXCService) DeleteVXC(ctx context.Context, id string, req *megaport.DeleteVXCRequest) error {
	return nil
}

func (m *MockVXCService) UpdateVXC(ctx context.Context, id string, req *megaport.UpdateVXCRequest) (*megaport.VXC, error) {
	return nil, nil
}

func (m *MockVXCService) LookupPartnerPorts(ctx context.Context, req *megaport.LookupPartnerPortsRequest) (*megaport.LookupPartnerPortsResponse, error) {
	return nil, nil
}

func (m *MockVXCService) ListPartnerPorts(ctx context.Context, req *megaport.ListPartnerPortsRequest) (*megaport.ListPartnerPortsResponse, error) {
	return nil, nil
}

func (m *MockVXCService) UpdateVXCResourceTags(ctx context.Context, vxcID string, tags map[string]string) error {
	return nil
}

// vxcsReadRequest builds a datasource.ReadRequest and ReadResponse for the vxcs
// data source schema. Non-nil productUID sets product_uid; non-nil
// includeResourceTags sets include_resource_tags; otherwise each is null.
func vxcsReadRequest(t *testing.T, ds *vxcsDataSource, productUID *string, includeResourceTags *bool) (datasource.ReadRequest, *datasource.ReadResponse) {
	t.Helper()
	ctx := context.Background()

	schemaResp := datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	tfType := schemaResp.Schema.Type().TerraformType(ctx)

	var uidVal tftypes.Value
	if productUID != nil {
		uidVal = tftypes.NewValue(tftypes.String, *productUID)
	} else {
		uidVal = tftypes.NewValue(tftypes.String, nil)
	}

	var tagsVal tftypes.Value
	if includeResourceTags != nil {
		tagsVal = tftypes.NewValue(tftypes.Bool, *includeResourceTags)
	} else {
		tagsVal = tftypes.NewValue(tftypes.Bool, nil)
	}

	vxcsAttrType := schemaResp.Schema.Attributes["vxcs"].GetType().TerraformType(ctx)
	configRaw := tftypes.NewValue(tfType, map[string]tftypes.Value{
		"product_uid":           uidVal,
		"include_resource_tags": tagsVal,
		"vxcs":                  tftypes.NewValue(vxcsAttrType, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	return req, resp
}

func TestReadVXCs_ListAll(t *testing.T) {
	ctx := context.Background()
	mockVXCService := &MockVXCService{
		ListVXCsResult: []*megaport.VXC{
			{UID: "vxc-1", Name: "VXC One"},
			{UID: "vxc-2", Name: "VXC Two"},
		},
	}
	ds := &vxcsDataSource{client: &megaport.Client{VXCService: mockVXCService}}

	req, resp := vxcsReadRequest(t, ds, nil, nil)
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state vxcsModel
	diags := resp.State.Get(ctx, &state)
	require.False(t, diags.HasError())

	assert.True(t, state.ProductUID.IsNull())

	var details []vxcDetailModel
	diags = state.VXCs.ElementsAs(ctx, &details, false)
	require.False(t, diags.HasError())

	require.Len(t, details, 2)
	assert.Equal(t, "vxc-1", details[0].UID.ValueString())
	assert.Equal(t, "VXC One", details[0].Name.ValueString())
	assert.Equal(t, "vxc-2", details[1].UID.ValueString())
	assert.Equal(t, "VXC Two", details[1].Name.ValueString())
}

func TestReadVXCs_GetByUID(t *testing.T) {
	ctx := context.Background()
	mockVXCService := &MockVXCService{
		GetVXCResult: &megaport.VXC{UID: "vxc-1", Name: "VXC One"},
	}
	ds := &vxcsDataSource{client: &megaport.Client{VXCService: mockVXCService}}

	uid := "vxc-1"
	req, resp := vxcsReadRequest(t, ds, &uid, nil)
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state vxcsModel
	diags := resp.State.Get(ctx, &state)
	require.False(t, diags.HasError())

	assert.Equal(t, "vxc-1", state.ProductUID.ValueString())

	var details []vxcDetailModel
	diags = state.VXCs.ElementsAs(ctx, &details, false)
	require.False(t, diags.HasError())

	require.Len(t, details, 1)
	assert.Equal(t, "vxc-1", details[0].UID.ValueString())
	assert.Equal(t, "VXC One", details[0].Name.ValueString())
}

func TestReadVXCs_ListError(t *testing.T) {
	ctx := context.Background()
	mockVXCService := &MockVXCService{ListVXCsErr: errors.New("API error")}
	ds := &vxcsDataSource{client: &megaport.Client{VXCService: mockVXCService}}

	req, resp := vxcsReadRequest(t, ds, nil, nil)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "API error")
}

func TestReadVXCs_GetByUIDError(t *testing.T) {
	ctx := context.Background()
	mockVXCService := &MockVXCService{GetVXCErr: errors.New("VXC not found")}
	ds := &vxcsDataSource{client: &megaport.Client{VXCService: mockVXCService}}

	uid := "vxc-nonexistent"
	req, resp := vxcsReadRequest(t, ds, &uid, nil)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "VXC not found")
}

func TestReadVXCs_TagsNotFetchedByDefault(t *testing.T) {
	ctx := context.Background()
	tagsCalled := false
	mockVXCService := &MockVXCService{
		ListVXCsResult: []*megaport.VXC{{UID: "vxc-1", Name: "VXC One"}},
		ListVXCResourceTagsFunc: func(_ context.Context, _ string) (map[string]string, error) {
			tagsCalled = true
			return map[string]string{"env": "test"}, nil
		},
	}
	ds := &vxcsDataSource{client: &megaport.Client{VXCService: mockVXCService}}

	req, resp := vxcsReadRequest(t, ds, nil, nil)
	ds.Read(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())
	assert.False(t, tagsCalled, "ListVXCResourceTags should not be called when include_resource_tags is null")

	var state vxcsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())
	var details []vxcDetailModel
	require.False(t, state.VXCs.ElementsAs(ctx, &details, false).HasError())
	require.Len(t, details, 1)
	assert.True(t, details[0].ResourceTags.IsNull())
}

func TestReadVXCs_TagsFetchedWhenOptedIn(t *testing.T) {
	ctx := context.Background()
	tagsCalled := false
	mockVXCService := &MockVXCService{
		ListVXCsResult: []*megaport.VXC{{UID: "vxc-1", Name: "VXC One"}},
		ListVXCResourceTagsFunc: func(_ context.Context, _ string) (map[string]string, error) {
			tagsCalled = true
			return map[string]string{"env": "test"}, nil
		},
	}
	ds := &vxcsDataSource{client: &megaport.Client{VXCService: mockVXCService}}

	yes := true
	req, resp := vxcsReadRequest(t, ds, nil, &yes)
	ds.Read(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())
	assert.True(t, tagsCalled, "ListVXCResourceTags should be called when include_resource_tags=true")

	var state vxcsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())
	var details []vxcDetailModel
	require.False(t, state.VXCs.ElementsAs(ctx, &details, false).HasError())
	require.Len(t, details, 1)
	assert.False(t, details[0].ResourceTags.IsNull())
	assert.Equal(t, "vxc-1", mockVXCService.CapturedResourceTagVXCUID)
}

func TestFromAPIVXCDetail(t *testing.T) {
	t.Run("Maps all fields correctly", func(t *testing.T) {
		vxc := &megaport.VXC{
			UID:                "vxc-abc-123",
			Name:               "My Test VXC",
			RateLimit:          1000,
			ProvisioningStatus: "LIVE",
			CreatedBy:          "user@example.com",
			CostCentre:         "CC-001",
			ContractTermMonths: 12,
			CompanyUID:         "company-abc",
			CompanyName:        "Acme Corp",
			DistanceBand:       "METRO",
			SecondaryName:      "Secondary VXC",
			Shutdown:           false,
			Locked:             false,
			AdminLocked:        true,
			Cancelable:         true,
			AEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "port-aaa",
				Name:       "Port A",
				LocationID: 123,
				VLAN:       100,
			},
			BEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "port-bbb",
				Name:       "Port B",
				LocationID: 456,
				VLAN:       200,
			},
			AttributeTags: map[string]string{
				"tag1": "val1",
				"tag2": "val2",
			},
		}

		tags := map[string]string{
			"env":   "production",
			"owner": "team-a",
		}

		detail := fromAPIVXCDetail(vxc, tags)

		assert.Equal(t, "vxc-abc-123", detail.UID.ValueString())
		assert.Equal(t, "My Test VXC", detail.Name.ValueString())
		assert.Equal(t, int64(1000), detail.RateLimit.ValueInt64())
		assert.Equal(t, "LIVE", detail.ProvisioningStatus.ValueString())
		assert.Equal(t, "user@example.com", detail.CreatedBy.ValueString())
		assert.Equal(t, "CC-001", detail.CostCentre.ValueString())
		assert.Equal(t, int64(12), detail.ContractTermMonths.ValueInt64())
		assert.Equal(t, "company-abc", detail.CompanyUID.ValueString())
		assert.Equal(t, "Acme Corp", detail.CompanyName.ValueString())
		assert.Equal(t, "METRO", detail.DistanceBand.ValueString())
		assert.Equal(t, "Secondary VXC", detail.SecondaryName.ValueString())
		assert.Equal(t, false, detail.Shutdown.ValueBool())
		assert.Equal(t, false, detail.Locked.ValueBool())
		assert.Equal(t, true, detail.AdminLocked.ValueBool())
		assert.Equal(t, true, detail.Cancelable.ValueBool())
		assert.Equal(t, "port-aaa", detail.AEndUID.ValueString())
		assert.Equal(t, "Port A", detail.AEndName.ValueString())
		assert.Equal(t, int64(123), detail.AEndLocationID.ValueInt64())
		assert.False(t, detail.AEndVLAN.IsNull())
		assert.Equal(t, int64(100), detail.AEndVLAN.ValueInt64())
		assert.Equal(t, "port-bbb", detail.BEndUID.ValueString())
		assert.Equal(t, "Port B", detail.BEndName.ValueString())
		assert.Equal(t, int64(456), detail.BEndLocationID.ValueInt64())
		assert.False(t, detail.BEndVLAN.IsNull())
		assert.Equal(t, int64(200), detail.BEndVLAN.ValueInt64())
		assert.False(t, detail.AttributeTags.IsNull())
		assert.False(t, detail.ResourceTags.IsNull())
	})

	t.Run("VLAN 0 maps to null", func(t *testing.T) {
		vxc := &megaport.VXC{
			UID: "vxc-zero-vlan",
			AEndConfiguration: megaport.VXCEndConfiguration{
				VLAN: 0,
			},
			BEndConfiguration: megaport.VXCEndConfiguration{
				VLAN: 0,
			},
		}
		detail := fromAPIVXCDetail(vxc, nil)
		assert.True(t, detail.AEndVLAN.IsNull(), "A-End VLAN 0 should map to null")
		assert.True(t, detail.BEndVLAN.IsNull(), "B-End VLAN 0 should map to null")
	})

	t.Run("Nil time fields produce null strings", func(t *testing.T) {
		vxc := &megaport.VXC{
			UID:               "vxc-nil-times",
			CreateDate:        nil,
			LiveDate:          nil,
			ContractStartDate: nil,
			ContractEndDate:   nil,
		}

		detail := fromAPIVXCDetail(vxc, nil)

		assert.True(t, detail.CreateDate.IsNull(), "nil CreateDate should map to null")
		assert.True(t, detail.LiveDate.IsNull(), "nil LiveDate should map to null")
		assert.True(t, detail.ContractStartDate.IsNull(), "nil ContractStartDate should map to null")
		assert.True(t, detail.ContractEndDate.IsNull(), "nil ContractEndDate should map to null")
	})

	t.Run("Nil tags produce null maps", func(t *testing.T) {
		vxc := &megaport.VXC{
			UID:           "vxc-nil-tags",
			AttributeTags: nil,
		}

		detail := fromAPIVXCDetail(vxc, nil)

		assert.True(t, detail.AttributeTags.IsNull())
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Empty resource tags produce null map", func(t *testing.T) {
		vxc := &megaport.VXC{
			UID: "vxc-empty-tags",
		}

		detail := fromAPIVXCDetail(vxc, map[string]string{})

		assert.True(t, detail.ResourceTags.IsNull())
	})
}

func TestFromAPIVXCDetail_ResourceTagsOptIn(t *testing.T) {
	t.Run("Tags nil when not fetched", func(t *testing.T) {
		vxc := &megaport.VXC{UID: "vxc-1"}
		detail := fromAPIVXCDetail(vxc, nil)
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Tags populated when fetched", func(t *testing.T) {
		vxc := &megaport.VXC{UID: "vxc-1"}
		tags := map[string]string{"env": "prod"}
		detail := fromAPIVXCDetail(vxc, tags)
		assert.False(t, detail.ResourceTags.IsNull())
	})
}

// Ensure vxcsModel compiles with the schema.
func TestVXCsModel_Structure(t *testing.T) {
	model := vxcsModel{
		ProductUID: types.StringValue("vxc-123"),
		VXCs:       types.ListNull(types.ObjectType{AttrTypes: vxcDetailAttrs}),
	}
	assert.Equal(t, "vxc-123", model.ProductUID.ValueString())
	assert.True(t, model.VXCs.IsNull())
}
