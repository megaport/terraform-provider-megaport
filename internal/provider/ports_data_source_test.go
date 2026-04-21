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

// MockPortService is a mock of the Port service for testing
type MockPortService struct {
	ListPortsResult            []*megaport.Port
	ListPortsErr               error
	GetPortResult              *megaport.Port
	GetPortErr                 error
	ListPortResourceTagsFunc   func(ctx context.Context, portID string) (map[string]string, error)
	ListPortResourceTagsErr    error
	ListPortResourceTagsResult map[string]string
	CapturedResourceTagPortUID string
}

func (m *MockPortService) ListPorts(ctx context.Context) ([]*megaport.Port, error) {
	if m.ListPortsErr != nil {
		return nil, m.ListPortsErr
	}
	if m.ListPortsResult != nil {
		return m.ListPortsResult, nil
	}
	return []*megaport.Port{}, nil
}

func (m *MockPortService) GetPort(ctx context.Context, portId string) (*megaport.Port, error) {
	if m.GetPortErr != nil {
		return nil, m.GetPortErr
	}
	return m.GetPortResult, nil
}

func (m *MockPortService) ListPortResourceTags(ctx context.Context, portID string) (map[string]string, error) {
	m.CapturedResourceTagPortUID = portID
	if m.ListPortResourceTagsFunc != nil {
		return m.ListPortResourceTagsFunc(ctx, portID)
	}
	if m.ListPortResourceTagsErr != nil {
		return nil, m.ListPortResourceTagsErr
	}
	if m.ListPortResourceTagsResult != nil {
		return m.ListPortResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the PortService interface with minimal stubs
func (m *MockPortService) BuyPort(ctx context.Context, req *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
	return nil, nil
}

func (m *MockPortService) ValidatePortOrder(ctx context.Context, req *megaport.BuyPortRequest) error {
	return nil
}

func (m *MockPortService) ModifyPort(ctx context.Context, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
	return nil, nil
}

func (m *MockPortService) DeletePort(ctx context.Context, req *megaport.DeletePortRequest) (*megaport.DeletePortResponse, error) {
	return nil, nil
}

func (m *MockPortService) RestorePort(ctx context.Context, portId string) (*megaport.RestorePortResponse, error) {
	return nil, nil
}

func (m *MockPortService) LockPort(ctx context.Context, portId string) (*megaport.LockPortResponse, error) {
	return nil, nil
}

func (m *MockPortService) UnlockPort(ctx context.Context, portId string) (*megaport.UnlockPortResponse, error) {
	return nil, nil
}

func (m *MockPortService) CheckPortVLANAvailability(ctx context.Context, portId string, vlan int) (bool, error) {
	return false, nil
}

func (m *MockPortService) UpdatePortResourceTags(ctx context.Context, portID string, tags map[string]string) error {
	return nil
}

// portsReadRequest builds a datasource.ReadRequest and ReadResponse for the ports
// data source schema. Non-nil productUID sets product_uid; non-nil
// includeResourceTags sets include_resource_tags; otherwise each is null.
func portsReadRequest(t *testing.T, ds *portsDataSource, productUID *string, includeResourceTags *bool) (datasource.ReadRequest, *datasource.ReadResponse) {
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

	portsAttrType := schemaResp.Schema.Attributes["ports"].GetType().TerraformType(ctx)
	configRaw := tftypes.NewValue(tfType, map[string]tftypes.Value{
		"product_uid":           uidVal,
		"include_resource_tags": tagsVal,
		"ports":                 tftypes.NewValue(portsAttrType, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	return req, resp
}

func TestReadPorts_ListAll(t *testing.T) {
	ctx := context.Background()
	mockPortService := &MockPortService{
		ListPortsResult: []*megaport.Port{
			{UID: "port-1", Name: "Port One"},
			{UID: "port-2", Name: "Port Two"},
		},
	}
	ds := &portsDataSource{client: &megaport.Client{PortService: mockPortService}}

	req, resp := portsReadRequest(t, ds, nil, nil)
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state portsModel
	diags := resp.State.Get(ctx, &state)
	require.False(t, diags.HasError())

	assert.True(t, state.ProductUID.IsNull())

	var details []portDetailModel
	diags = state.Ports.ElementsAs(ctx, &details, false)
	require.False(t, diags.HasError())

	require.Len(t, details, 2)
	assert.Equal(t, "port-1", details[0].UID.ValueString())
	assert.Equal(t, "Port One", details[0].Name.ValueString())
	assert.Equal(t, "port-2", details[1].UID.ValueString())
	assert.Equal(t, "Port Two", details[1].Name.ValueString())
}

func TestReadPorts_GetByUID(t *testing.T) {
	ctx := context.Background()
	mockPortService := &MockPortService{
		GetPortResult: &megaport.Port{UID: "port-1", Name: "Port One"},
	}
	ds := &portsDataSource{client: &megaport.Client{PortService: mockPortService}}

	uid := "port-1"
	req, resp := portsReadRequest(t, ds, &uid, nil)
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state portsModel
	diags := resp.State.Get(ctx, &state)
	require.False(t, diags.HasError())

	assert.Equal(t, "port-1", state.ProductUID.ValueString())

	var details []portDetailModel
	diags = state.Ports.ElementsAs(ctx, &details, false)
	require.False(t, diags.HasError())

	require.Len(t, details, 1)
	assert.Equal(t, "port-1", details[0].UID.ValueString())
	assert.Equal(t, "Port One", details[0].Name.ValueString())
}

func TestReadPorts_ListError(t *testing.T) {
	ctx := context.Background()
	mockPortService := &MockPortService{ListPortsErr: errors.New("API error")}
	ds := &portsDataSource{client: &megaport.Client{PortService: mockPortService}}

	req, resp := portsReadRequest(t, ds, nil, nil)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "API error")
}

func TestReadPorts_GetByUIDError(t *testing.T) {
	ctx := context.Background()
	mockPortService := &MockPortService{GetPortErr: errors.New("port not found")}
	ds := &portsDataSource{client: &megaport.Client{PortService: mockPortService}}

	uid := "port-nonexistent"
	req, resp := portsReadRequest(t, ds, &uid, nil)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "port not found")
}

func TestReadPorts_GetByUIDReturnsNil(t *testing.T) {
	ctx := context.Background()
	mockPortService := &MockPortService{GetPortResult: nil, GetPortErr: nil}
	ds := &portsDataSource{client: &megaport.Client{PortService: mockPortService}}

	uid := "port-nonexistent"
	req, resp := portsReadRequest(t, ds, &uid, nil)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "not found")
}

func TestReadPorts_TagsNotFetchedByDefault(t *testing.T) {
	ctx := context.Background()
	tagsCalled := false
	mockPortService := &MockPortService{
		ListPortsResult: []*megaport.Port{{UID: "port-1", Name: "Port One"}},
		ListPortResourceTagsFunc: func(_ context.Context, _ string) (map[string]string, error) {
			tagsCalled = true
			return map[string]string{"env": "test"}, nil
		},
	}
	ds := &portsDataSource{client: &megaport.Client{PortService: mockPortService}}

	req, resp := portsReadRequest(t, ds, nil, nil)
	ds.Read(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())
	assert.False(t, tagsCalled, "ListPortResourceTags should not be called when include_resource_tags is null")

	var state portsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())
	var details []portDetailModel
	require.False(t, state.Ports.ElementsAs(ctx, &details, false).HasError())
	require.Len(t, details, 1)
	assert.True(t, details[0].ResourceTags.IsNull())
}

func TestReadPorts_TagsFetchedWhenOptedIn(t *testing.T) {
	ctx := context.Background()
	tagsCalled := false
	mockPortService := &MockPortService{
		ListPortsResult: []*megaport.Port{{UID: "port-1", Name: "Port One"}},
		ListPortResourceTagsFunc: func(_ context.Context, _ string) (map[string]string, error) {
			tagsCalled = true
			return map[string]string{"env": "test"}, nil
		},
	}
	ds := &portsDataSource{client: &megaport.Client{PortService: mockPortService}}

	yes := true
	req, resp := portsReadRequest(t, ds, nil, &yes)
	ds.Read(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())
	assert.True(t, tagsCalled, "ListPortResourceTags should be called when include_resource_tags=true")

	var state portsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())
	var details []portDetailModel
	require.False(t, state.Ports.ElementsAs(ctx, &details, false).HasError())
	require.Len(t, details, 1)
	assert.False(t, details[0].ResourceTags.IsNull())
	assert.Equal(t, "port-1", mockPortService.CapturedResourceTagPortUID)
}

func TestFromAPIPortDetail(t *testing.T) {
	t.Run("Maps all fields correctly", func(t *testing.T) {
		port := &megaport.Port{
			UID:                   "port-abc-123",
			Name:                  "My Test Port",
			ProvisioningStatus:    "LIVE",
			CreatedBy:             "user@example.com",
			PortSpeed:             10000,
			Market:                "Sydney",
			LocationID:            65,
			MarketplaceVisibility: true,
			VXCPermitted:          true,
			VXCAutoApproval:       false,
			SecondaryName:         "Secondary Port",
			LAGPrimary:            false,
			CompanyUID:            "company-abc",
			CompanyName:           "Acme Corp",
			CostCentre:            "CC-001",
			ContractTermMonths:    12,
			Locked:                false,
			AdminLocked:           true,
			Cancelable:            true,
			DiversityZone:         "zone-a",
		}

		tags := map[string]string{
			"env":   "production",
			"owner": "team-a",
		}

		detail, diags := fromAPIPortDetail(port, tags)
		assert.False(t, diags.HasError())

		assert.Equal(t, "port-abc-123", detail.UID.ValueString())
		assert.Equal(t, "My Test Port", detail.Name.ValueString())
		assert.Equal(t, "LIVE", detail.ProvisioningStatus.ValueString())
		assert.Equal(t, "user@example.com", detail.CreatedBy.ValueString())
		assert.Equal(t, int64(10000), detail.PortSpeed.ValueInt64())
		assert.Equal(t, "Sydney", detail.Market.ValueString())
		assert.Equal(t, int64(65), detail.LocationID.ValueInt64())
		assert.Equal(t, true, detail.MarketplaceVisibility.ValueBool())
		assert.Equal(t, true, detail.VXCPermitted.ValueBool())
		assert.Equal(t, false, detail.VXCAutoApproval.ValueBool())
		assert.Equal(t, "Secondary Port", detail.SecondaryName.ValueString())
		assert.Equal(t, false, detail.LAGPrimary.ValueBool())
		assert.Equal(t, "company-abc", detail.CompanyUID.ValueString())
		assert.Equal(t, "Acme Corp", detail.CompanyName.ValueString())
		assert.Equal(t, "CC-001", detail.CostCentre.ValueString())
		assert.Equal(t, int64(12), detail.ContractTermMonths.ValueInt64())
		assert.Equal(t, false, detail.Locked.ValueBool())
		assert.Equal(t, true, detail.AdminLocked.ValueBool())
		assert.Equal(t, true, detail.Cancelable.ValueBool())
		assert.Equal(t, "zone-a", detail.DiversityZone.ValueString())
		assert.False(t, detail.ResourceTags.IsNull())
	})

	t.Run("Nil time fields produce null strings", func(t *testing.T) {
		port := &megaport.Port{
			UID:               "port-nil-times",
			CreateDate:        nil,
			LiveDate:          nil,
			TerminateDate:     nil,
			ContractStartDate: nil,
			ContractEndDate:   nil,
		}

		detail, diags := fromAPIPortDetail(port, nil)
		assert.False(t, diags.HasError())

		assert.True(t, detail.CreateDate.IsNull())
		assert.True(t, detail.LiveDate.IsNull())
		assert.True(t, detail.TerminateDate.IsNull())
		assert.True(t, detail.ContractStartDate.IsNull())
		assert.True(t, detail.ContractEndDate.IsNull())
	})

	t.Run("Nil tags produce null map", func(t *testing.T) {
		port := &megaport.Port{
			UID: "port-nil-tags",
		}

		detail, diags := fromAPIPortDetail(port, nil)
		assert.False(t, diags.HasError())

		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Empty resource tags produce null map", func(t *testing.T) {
		port := &megaport.Port{
			UID: "port-empty-tags",
		}

		detail, diags := fromAPIPortDetail(port, map[string]string{})
		assert.False(t, diags.HasError())

		assert.True(t, detail.ResourceTags.IsNull())
	})
}

func TestFromAPIPortDetail_ResourceTagsOptIn(t *testing.T) {
	t.Run("Tags nil when not fetched", func(t *testing.T) {
		port := &megaport.Port{UID: "port-1"}
		detail, diags := fromAPIPortDetail(port, nil)
		assert.False(t, diags.HasError())
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Tags populated when fetched", func(t *testing.T) {
		port := &megaport.Port{UID: "port-1"}
		tags := map[string]string{"env": "prod"}
		detail, diags := fromAPIPortDetail(port, tags)
		assert.False(t, diags.HasError())
		assert.False(t, detail.ResourceTags.IsNull())
	})
}

// Ensure portsModel compiles with the schema.
func TestPortsModel_Structure(t *testing.T) {
	model := portsModel{
		ProductUID: types.StringValue("port-123"),
		Ports:      types.ListNull(types.ObjectType{AttrTypes: portDetailAttrs}),
	}
	assert.Equal(t, "port-123", model.ProductUID.ValueString())
	assert.True(t, model.Ports.IsNull())
}
