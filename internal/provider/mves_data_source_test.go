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

// MockMVEService is a mock of the MVE service for testing
type MockMVEService struct {
	ListMVEsResult            []*megaport.MVE
	ListMVEsErr               error
	GetMVEResult              *megaport.MVE
	GetMVEErr                 error
	ListMVEResourceTagsFunc   func(ctx context.Context, mveID string) (map[string]string, error)
	ListMVEResourceTagsErr    error
	ListMVEResourceTagsResult map[string]string
	CapturedResourceTagMVEUID string
}

func (m *MockMVEService) ListMVEs(ctx context.Context, req *megaport.ListMVEsRequest) ([]*megaport.MVE, error) {
	if m.ListMVEsErr != nil {
		return nil, m.ListMVEsErr
	}
	if m.ListMVEsResult != nil {
		return m.ListMVEsResult, nil
	}
	return []*megaport.MVE{}, nil
}

func (m *MockMVEService) GetMVE(ctx context.Context, mveId string) (*megaport.MVE, error) {
	if m.GetMVEErr != nil {
		return nil, m.GetMVEErr
	}
	return m.GetMVEResult, nil
}

func (m *MockMVEService) ListMVEResourceTags(ctx context.Context, mveID string) (map[string]string, error) {
	m.CapturedResourceTagMVEUID = mveID
	if m.ListMVEResourceTagsFunc != nil {
		return m.ListMVEResourceTagsFunc(ctx, mveID)
	}
	if m.ListMVEResourceTagsErr != nil {
		return nil, m.ListMVEResourceTagsErr
	}
	if m.ListMVEResourceTagsResult != nil {
		return m.ListMVEResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the MVEService interface with minimal stubs
func (m *MockMVEService) BuyMVE(ctx context.Context, req *megaport.BuyMVERequest) (*megaport.BuyMVEResponse, error) {
	return nil, nil
}

func (m *MockMVEService) ValidateMVEOrder(ctx context.Context, req *megaport.BuyMVERequest) error {
	return nil
}

func (m *MockMVEService) ModifyMVE(ctx context.Context, req *megaport.ModifyMVERequest) (*megaport.ModifyMVEResponse, error) {
	return nil, nil
}

func (m *MockMVEService) DeleteMVE(ctx context.Context, req *megaport.DeleteMVERequest) (*megaport.DeleteMVEResponse, error) {
	return nil, nil
}

func (m *MockMVEService) ListMVEImages(ctx context.Context) ([]*megaport.MVEImage, error) {
	return nil, nil
}

func (m *MockMVEService) ListAvailableMVESizes(ctx context.Context) ([]*megaport.MVESize, error) {
	return nil, nil
}

func (m *MockMVEService) UpdateMVEResourceTags(ctx context.Context, mveID string, tags map[string]string) error {
	return nil
}

// mvesReadRequest builds a datasource.ReadRequest and ReadResponse for the mves
// data source schema. Non-nil productUID sets product_uid; non-nil
// includeResourceTags sets include_resource_tags; otherwise each is null.
func mvesReadRequest(t *testing.T, ds *mvesDataSource, productUID *string, includeResourceTags *bool) (datasource.ReadRequest, *datasource.ReadResponse) {
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

	mvesAttrType := schemaResp.Schema.Attributes["mves"].GetType().TerraformType(ctx)
	configRaw := tftypes.NewValue(tfType, map[string]tftypes.Value{
		"product_uid":           uidVal,
		"include_resource_tags": tagsVal,
		"mves":                  tftypes.NewValue(mvesAttrType, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	return req, resp
}

func TestReadMVEs_ListAll(t *testing.T) {
	ctx := context.Background()
	mockMVEService := &MockMVEService{
		ListMVEsResult: []*megaport.MVE{
			{UID: "mve-1", Name: "MVE One"},
			{UID: "mve-2", Name: "MVE Two"},
		},
	}
	ds := &mvesDataSource{client: &megaport.Client{MVEService: mockMVEService}}

	req, resp := mvesReadRequest(t, ds, nil, nil)
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state mvesModel
	diags := resp.State.Get(ctx, &state)
	require.False(t, diags.HasError())

	assert.True(t, state.ProductUID.IsNull())

	var details []mveDetailModel
	diags = state.MVEs.ElementsAs(ctx, &details, false)
	require.False(t, diags.HasError())

	require.Len(t, details, 2)
	assert.Equal(t, "mve-1", details[0].UID.ValueString())
	assert.Equal(t, "MVE One", details[0].Name.ValueString())
	assert.Equal(t, "mve-2", details[1].UID.ValueString())
	assert.Equal(t, "MVE Two", details[1].Name.ValueString())
}

func TestReadMVEs_GetByUID(t *testing.T) {
	ctx := context.Background()
	mockMVEService := &MockMVEService{
		GetMVEResult: &megaport.MVE{UID: "mve-1", Name: "MVE One"},
	}
	ds := &mvesDataSource{client: &megaport.Client{MVEService: mockMVEService}}

	uid := "mve-1"
	req, resp := mvesReadRequest(t, ds, &uid, nil)
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state mvesModel
	diags := resp.State.Get(ctx, &state)
	require.False(t, diags.HasError())

	assert.Equal(t, "mve-1", state.ProductUID.ValueString())

	var details []mveDetailModel
	diags = state.MVEs.ElementsAs(ctx, &details, false)
	require.False(t, diags.HasError())

	require.Len(t, details, 1)
	assert.Equal(t, "mve-1", details[0].UID.ValueString())
	assert.Equal(t, "MVE One", details[0].Name.ValueString())
}

func TestReadMVEs_ListError(t *testing.T) {
	ctx := context.Background()
	mockMVEService := &MockMVEService{ListMVEsErr: errors.New("API error")}
	ds := &mvesDataSource{client: &megaport.Client{MVEService: mockMVEService}}

	req, resp := mvesReadRequest(t, ds, nil, nil)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "API error")
}

func TestReadMVEs_GetByUIDError(t *testing.T) {
	ctx := context.Background()
	mockMVEService := &MockMVEService{GetMVEErr: errors.New("MVE not found")}
	ds := &mvesDataSource{client: &megaport.Client{MVEService: mockMVEService}}

	uid := "mve-nonexistent"
	req, resp := mvesReadRequest(t, ds, &uid, nil)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "MVE not found")
}

func TestReadMVEs_GetByUIDReturnsNil(t *testing.T) {
	ctx := context.Background()
	mockMVEService := &MockMVEService{GetMVEResult: nil, GetMVEErr: nil}
	ds := &mvesDataSource{client: &megaport.Client{MVEService: mockMVEService}}

	uid := "mve-nonexistent"
	req, resp := mvesReadRequest(t, ds, &uid, nil)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "not found")
}

func TestReadMVEs_TagsNotFetchedByDefault(t *testing.T) {
	ctx := context.Background()
	tagsCalled := false
	mockMVEService := &MockMVEService{
		ListMVEsResult: []*megaport.MVE{{UID: "mve-1", Name: "MVE One"}},
		ListMVEResourceTagsFunc: func(_ context.Context, _ string) (map[string]string, error) {
			tagsCalled = true
			return map[string]string{"env": "test"}, nil
		},
	}
	ds := &mvesDataSource{client: &megaport.Client{MVEService: mockMVEService}}

	req, resp := mvesReadRequest(t, ds, nil, nil)
	ds.Read(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())
	assert.False(t, tagsCalled, "ListMVEResourceTags should not be called when include_resource_tags is null")

	var state mvesModel
	require.False(t, resp.State.Get(ctx, &state).HasError())
	var details []mveDetailModel
	require.False(t, state.MVEs.ElementsAs(ctx, &details, false).HasError())
	require.Len(t, details, 1)
	assert.True(t, details[0].ResourceTags.IsNull())
}

func TestReadMVEs_TagsFetchedWhenOptedIn(t *testing.T) {
	ctx := context.Background()
	tagsCalled := false
	mockMVEService := &MockMVEService{
		ListMVEsResult: []*megaport.MVE{{UID: "mve-1", Name: "MVE One"}},
		ListMVEResourceTagsFunc: func(_ context.Context, _ string) (map[string]string, error) {
			tagsCalled = true
			return map[string]string{"env": "test"}, nil
		},
	}
	ds := &mvesDataSource{client: &megaport.Client{MVEService: mockMVEService}}

	yes := true
	req, resp := mvesReadRequest(t, ds, nil, &yes)
	ds.Read(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())
	assert.True(t, tagsCalled, "ListMVEResourceTags should be called when include_resource_tags=true")

	var state mvesModel
	require.False(t, resp.State.Get(ctx, &state).HasError())
	var details []mveDetailModel
	require.False(t, state.MVEs.ElementsAs(ctx, &details, false).HasError())
	require.Len(t, details, 1)
	assert.False(t, details[0].ResourceTags.IsNull())
	assert.Equal(t, "mve-1", mockMVEService.CapturedResourceTagMVEUID)
}

func TestFromAPIMVEDetail(t *testing.T) {
	t.Run("Maps all fields correctly", func(t *testing.T) {
		mve := &megaport.MVE{
			UID:                   "mve-abc-123",
			Name:                  "My Test MVE",
			ProvisioningStatus:    "LIVE",
			CreatedBy:             "user@example.com",
			Market:                "Sydney",
			LocationID:            65,
			MarketplaceVisibility: true,
			VXCPermitted:          true,
			VXCAutoApproval:       false,
			SecondaryName:         "Secondary MVE",
			CompanyUID:            "company-abc",
			CompanyName:           "Acme Corp",
			CostCentre:            "CC-001",
			ContractTermMonths:    12,
			Locked:                false,
			AdminLocked:           true,
			Cancelable:            true,
			Vendor:                "cisco",
			Size:                  "MEDIUM",
			DiversityZone:         "zone-a",
			AttributeTags: map[string]string{
				"tag1": "val1",
				"tag2": "val2",
			},
		}

		tags := map[string]string{
			"env":   "production",
			"owner": "team-a",
		}

		detail, diags := fromAPIMVEDetail(mve, tags)
		assert.False(t, diags.HasError())

		assert.Equal(t, "mve-abc-123", detail.UID.ValueString())
		assert.Equal(t, "My Test MVE", detail.Name.ValueString())
		assert.Equal(t, "LIVE", detail.ProvisioningStatus.ValueString())
		assert.Equal(t, "user@example.com", detail.CreatedBy.ValueString())
		assert.Equal(t, "Sydney", detail.Market.ValueString())
		assert.Equal(t, int64(65), detail.LocationID.ValueInt64())
		assert.Equal(t, true, detail.MarketplaceVisibility.ValueBool())
		assert.Equal(t, true, detail.VXCPermitted.ValueBool())
		assert.Equal(t, false, detail.VXCAutoApproval.ValueBool())
		assert.Equal(t, "Secondary MVE", detail.SecondaryName.ValueString())
		assert.Equal(t, "company-abc", detail.CompanyUID.ValueString())
		assert.Equal(t, "Acme Corp", detail.CompanyName.ValueString())
		assert.Equal(t, "CC-001", detail.CostCentre.ValueString())
		assert.Equal(t, int64(12), detail.ContractTermMonths.ValueInt64())
		assert.Equal(t, false, detail.Locked.ValueBool())
		assert.Equal(t, true, detail.AdminLocked.ValueBool())
		assert.Equal(t, true, detail.Cancelable.ValueBool())
		assert.Equal(t, "cisco", detail.Vendor.ValueString())
		assert.Equal(t, "MEDIUM", detail.Size.ValueString())
		assert.Equal(t, "zone-a", detail.DiversityZone.ValueString())
		assert.False(t, detail.AttributeTags.IsNull())
		assert.False(t, detail.ResourceTags.IsNull())
	})

	t.Run("Nil time fields produce null values", func(t *testing.T) {
		mve := &megaport.MVE{
			UID:               "mve-nil-times",
			CreateDate:        nil,
			LiveDate:          nil,
			TerminateDate:     nil,
			ContractStartDate: nil,
			ContractEndDate:   nil,
		}

		detail, diags := fromAPIMVEDetail(mve, nil)
		assert.False(t, diags.HasError())

		assert.True(t, detail.CreateDate.IsNull())
		assert.True(t, detail.LiveDate.IsNull())
		assert.True(t, detail.TerminateDate.IsNull())
		assert.True(t, detail.ContractStartDate.IsNull())
		assert.True(t, detail.ContractEndDate.IsNull())
	})

	t.Run("Nil tags produce null maps", func(t *testing.T) {
		mve := &megaport.MVE{
			UID:           "mve-nil-tags",
			AttributeTags: nil,
		}

		detail, diags := fromAPIMVEDetail(mve, nil)
		assert.False(t, diags.HasError())

		assert.True(t, detail.AttributeTags.IsNull())
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Empty non-nil resource tags produce empty map (fetched but empty)", func(t *testing.T) {
		mve := &megaport.MVE{
			UID: "mve-empty-tags",
		}

		detail, diags := fromAPIMVEDetail(mve, map[string]string{})
		assert.False(t, diags.HasError())

		assert.False(t, detail.ResourceTags.IsNull())
		assert.Equal(t, 0, len(detail.ResourceTags.Elements()))
	})
}

func TestFromAPIMVEDetail_ResourceTagsOptIn(t *testing.T) {
	t.Run("Tags nil when not fetched", func(t *testing.T) {
		mve := &megaport.MVE{UID: "mve-1"}
		detail, diags := fromAPIMVEDetail(mve, nil)
		assert.False(t, diags.HasError())
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Tags populated when fetched", func(t *testing.T) {
		mve := &megaport.MVE{UID: "mve-1"}
		tags := map[string]string{"env": "prod"}
		detail, diags := fromAPIMVEDetail(mve, tags)
		assert.False(t, diags.HasError())
		assert.False(t, detail.ResourceTags.IsNull())
	})
}

// Ensure mvesModel compiles with the schema.
func TestMVEsModel_Structure(t *testing.T) {
	model := mvesModel{
		ProductUID: types.StringValue("mve-123"),
		MVEs:       types.ListNull(types.ObjectType{AttrTypes: mveDetailAttrs}),
	}
	assert.Equal(t, "mve-123", model.ProductUID.ValueString())
	assert.True(t, model.MVEs.IsNull())
}
