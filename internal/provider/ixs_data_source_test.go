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

// MockIXService is a mock of the IX service for testing
type MockIXService struct {
	ListIXsResult []*megaport.IX
	ListIXsErr    error
	GetIXResult   *megaport.IX
	GetIXErr      error
}

func (m *MockIXService) ListIXs(ctx context.Context, req *megaport.ListIXsRequest) ([]*megaport.IX, error) {
	if m.ListIXsErr != nil {
		return nil, m.ListIXsErr
	}
	if m.ListIXsResult != nil {
		return m.ListIXsResult, nil
	}
	return []*megaport.IX{}, nil
}

func (m *MockIXService) GetIX(ctx context.Context, id string) (*megaport.IX, error) {
	if m.GetIXErr != nil {
		return nil, m.GetIXErr
	}
	return m.GetIXResult, nil
}

// Implement other required methods of the IXService interface with minimal stubs
func (m *MockIXService) BuyIX(ctx context.Context, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
	return nil, nil
}

func (m *MockIXService) ValidateIXOrder(ctx context.Context, req *megaport.BuyIXRequest) error {
	return nil
}

func (m *MockIXService) UpdateIX(ctx context.Context, id string, req *megaport.UpdateIXRequest) (*megaport.IX, error) {
	return nil, nil
}

func (m *MockIXService) DeleteIX(ctx context.Context, id string, req *megaport.DeleteIXRequest) error {
	return nil
}

// ixsReadRequest builds a datasource.ReadRequest and ReadResponse for the ixs
// data source schema. When productUID is non-nil the config sets product_uid to
// that value; otherwise the attribute is null (triggering a list-all call).
func ixsReadRequest(t *testing.T, ds *ixsDataSource, productUID *string) (datasource.ReadRequest, *datasource.ReadResponse) {
	t.Helper()
	ctx := context.Background()

	// Obtain the schema so tfsdk objects match.
	schemaResp := datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	tfType := schemaResp.Schema.Type().TerraformType(ctx)

	// Build the product_uid tftypes value.
	var uidVal tftypes.Value
	if productUID != nil {
		uidVal = tftypes.NewValue(tftypes.String, *productUID)
	} else {
		uidVal = tftypes.NewValue(tftypes.String, nil) // null
	}

	// The ixs attribute is computed, so it is always null in config.
	ixsAttrType := schemaResp.Schema.Attributes["ixs"].GetType().TerraformType(ctx)
	configRaw := tftypes.NewValue(tfType, map[string]tftypes.Value{
		"product_uid": uidVal,
		"ixs":         tftypes.NewValue(ixsAttrType, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	return req, resp
}

func TestReadIXs_ListAll(t *testing.T) {
	ctx := context.Background()
	mockIXService := &MockIXService{
		ListIXsResult: []*megaport.IX{
			{ProductUID: "ix-1", ProductName: "IX One"},
			{ProductUID: "ix-2", ProductName: "IX Two"},
		},
	}
	ds := &ixsDataSource{client: &megaport.Client{IXService: mockIXService}}

	req, resp := ixsReadRequest(t, ds, nil)
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state ixsModel
	diags := resp.State.Get(ctx, &state)
	require.False(t, diags.HasError())

	assert.True(t, state.ProductUID.IsNull())

	var details []ixDetailModel
	diags = state.IXs.ElementsAs(ctx, &details, false)
	require.False(t, diags.HasError())

	require.Len(t, details, 2)
	assert.Equal(t, "ix-1", details[0].UID.ValueString())
	assert.Equal(t, "IX One", details[0].Name.ValueString())
	assert.Equal(t, "ix-2", details[1].UID.ValueString())
	assert.Equal(t, "IX Two", details[1].Name.ValueString())
}

func TestReadIXs_GetByUID(t *testing.T) {
	ctx := context.Background()
	mockIXService := &MockIXService{
		GetIXResult: &megaport.IX{ProductUID: "ix-1", ProductName: "IX One"},
	}
	ds := &ixsDataSource{client: &megaport.Client{IXService: mockIXService}}

	uid := "ix-1"
	req, resp := ixsReadRequest(t, ds, &uid)
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state ixsModel
	diags := resp.State.Get(ctx, &state)
	require.False(t, diags.HasError())

	assert.Equal(t, "ix-1", state.ProductUID.ValueString())

	var details []ixDetailModel
	diags = state.IXs.ElementsAs(ctx, &details, false)
	require.False(t, diags.HasError())

	require.Len(t, details, 1)
	assert.Equal(t, "ix-1", details[0].UID.ValueString())
	assert.Equal(t, "IX One", details[0].Name.ValueString())
}

func TestReadIXs_ListError(t *testing.T) {
	ctx := context.Background()
	mockIXService := &MockIXService{
		ListIXsErr: errors.New("API error"),
	}
	ds := &ixsDataSource{client: &megaport.Client{IXService: mockIXService}}

	req, resp := ixsReadRequest(t, ds, nil)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "API error")
}

func TestReadIXs_GetByUIDError(t *testing.T) {
	ctx := context.Background()
	mockIXService := &MockIXService{
		GetIXErr: errors.New("IX not found"),
	}
	ds := &ixsDataSource{client: &megaport.Client{IXService: mockIXService}}

	uid := "ix-nonexistent"
	req, resp := ixsReadRequest(t, ds, &uid)
	ds.Read(ctx, req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "IX not found")
}

func TestFromAPIIXDetail(t *testing.T) {
	t.Run("Maps all fields correctly", func(t *testing.T) {
		ix := &megaport.IX{
			ProductUID:         "ix-abc-123",
			ProductName:        "My Test IX",
			ProvisioningStatus: "LIVE",
			LocationID:         65,
			RateLimit:          1000,
			Term:               12,
			SecondaryName:      "Secondary IX",
			VLAN:               100,
			MACAddress:         "00:11:22:33:44:55",
			ASN:                64512,
			NetworkServiceType: "Los Angeles IX",
			AttributeTags: map[string]string{
				"tag1": "val1",
				"tag2": "val2",
			},
		}

		detail, diags := fromAPIIXDetail(ix)
		assert.False(t, diags.HasError())

		assert.Equal(t, "ix-abc-123", detail.UID.ValueString())
		assert.Equal(t, "My Test IX", detail.Name.ValueString())
		assert.Equal(t, "LIVE", detail.ProvisioningStatus.ValueString())
		assert.Equal(t, int64(65), detail.LocationID.ValueInt64())
		assert.Equal(t, int64(1000), detail.RateLimit.ValueInt64())
		assert.Equal(t, int64(12), detail.Term.ValueInt64())
		assert.Equal(t, "Secondary IX", detail.SecondaryName.ValueString())
		assert.Equal(t, int64(100), detail.VLAN.ValueInt64())
		assert.Equal(t, "00:11:22:33:44:55", detail.MACAddress.ValueString())
		assert.Equal(t, int64(64512), detail.ASN.ValueInt64())
		assert.Equal(t, "Los Angeles IX", detail.NetworkServiceType.ValueString())
		assert.False(t, detail.AttributeTags.IsNull())
	})

	t.Run("Nil time fields produce null values", func(t *testing.T) {
		ix := &megaport.IX{
			ProductUID: "ix-nil-times",
			CreateDate: nil,
			DeployDate: nil,
		}

		detail, diags := fromAPIIXDetail(ix)
		assert.False(t, diags.HasError())

		assert.True(t, detail.CreateDate.IsNull())
		assert.True(t, detail.DeployDate.IsNull())
	})

	t.Run("Nil tags produce null maps", func(t *testing.T) {
		ix := &megaport.IX{
			ProductUID:    "ix-nil-tags",
			AttributeTags: nil,
		}

		detail, diags := fromAPIIXDetail(ix)
		assert.False(t, diags.HasError())

		assert.True(t, detail.AttributeTags.IsNull())
	})
}

// Ensure ixsModel compiles with the schema.
func TestIXsModel_Structure(t *testing.T) {
	model := ixsModel{
		ProductUID: types.StringValue("ix-123"),
		IXs:        types.ListNull(types.ObjectType{AttrTypes: ixDetailAttrs}),
	}
	assert.Equal(t, "ix-123", model.ProductUID.ValueString())
	assert.True(t, model.IXs.IsNull())
}
