package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

// mockCompanyService records the calls to GetCurrentCompany.
type mockCompanyService struct {
	result *megaport.CurrentCompany
	err    error
	calls  int
}

func (m *mockCompanyService) GetCurrentCompany(_ context.Context) (*megaport.CurrentCompany, error) {
	m.calls++
	return m.result, m.err
}

// mockProductAvailabilityService records the company UID of each call.
type mockProductAvailabilityService struct {
	result     []*megaport.CompanyProductAvailability
	err        error
	gotCompany []string
}

func (m *mockProductAvailabilityService) ListCompanyProductAvailability(_ context.Context, companyUID string) ([]*megaport.CompanyProductAvailability, error) {
	m.gotCompany = append(m.gotCompany, companyUID)
	return m.result, m.err
}

// productAvailabilityReadRequest builds the request and response for a data
// source that takes no arguments.
func productAvailabilityReadRequest(t *testing.T, ds *productAvailabilityDataSource) (datasource.ReadRequest, *datasource.ReadResponse) {
	t.Helper()
	ctx := context.Background()

	schemaResp := datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	tfType := schemaResp.Schema.Type().TerraformType(ctx)
	configRaw := tftypes.NewValue(tfType, map[string]tftypes.Value{
		"company_uid": tftypes.NewValue(tftypes.String, nil),
		"products":    tftypes.NewValue(schemaResp.Schema.Attributes["products"].GetType().TerraformType(ctx), nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	return req, resp
}

func newProductAvailabilityDataSource(company *mockCompanyService, availability *mockProductAvailabilityService) *productAvailabilityDataSource {
	return &productAvailabilityDataSource{client: &megaport.Client{
		CompanyService:             company,
		ProductAvailabilityService: availability,
	}}
}

func TestProductAvailabilityDataSource_Read(t *testing.T) {
	ctx := context.Background()
	company := &mockCompanyService{result: &megaport.CurrentCompany{CompanyUID: "company-uid-1"}}
	availability := &mockProductAvailabilityService{result: []*megaport.CompanyProductAvailability{
		{
			Category:    megaport.AvailableProductCategoryProduct,
			DisplayName: "Port",
			Variant:     "PORT",
			Markets: []megaport.CompanyProductMarket{
				{Market: "MEGAPORT_AUSTRALIA", MarketName: "Australia", MarketOwner: "Megaport (Australia) Pty Ltd", CountryCode: "AU", Source: megaport.AvailabilitySourceMarket},
				{Market: "MEGAPORT_US", MarketName: "United States", MarketOwner: "Megaport (USA) Inc.", CountryCode: "US", Source: megaport.AvailabilitySourceOverride},
			},
		},
		{
			Category:    megaport.AvailableProductCategoryAddon,
			DisplayName: "IPsec",
			Variant:     "MCR_IPSEC",
			Markets:     []megaport.CompanyProductMarket{},
		},
		nil,
	}}
	ds := newProductAvailabilityDataSource(company, availability)

	req, resp := productAvailabilityReadRequest(t, ds)
	ds.Read(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state productAvailabilityDataSourceModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	assert.Equal(t, 1, company.calls)
	assert.Equal(t, "company-uid-1", state.CompanyUID.ValueString())
	assert.Equal(t, []string{"company-uid-1"}, availability.gotCompany)
	require.Len(t, state.Products, 2)

	port := state.Products[0]
	assert.Equal(t, "PRODUCT", port.Category.ValueString())
	assert.Equal(t, "Port", port.DisplayName.ValueString())
	assert.Equal(t, "PORT", port.Variant.ValueString())
	require.Len(t, port.Markets, 2)
	assert.Equal(t, "MEGAPORT_AUSTRALIA", port.Markets[0].Market.ValueString())
	assert.Equal(t, "Australia", port.Markets[0].MarketName.ValueString())
	assert.Equal(t, "Megaport (Australia) Pty Ltd", port.Markets[0].MarketOwner.ValueString())
	assert.Equal(t, "AU", port.Markets[0].CountryCode.ValueString())
	assert.Equal(t, "MARKET", port.Markets[0].Source.ValueString())
	assert.Equal(t, "OVERRIDE", port.Markets[1].Source.ValueString())

	assert.Equal(t, "ADDON", state.Products[1].Category.ValueString())
	var addonMarkets types.List
	require.False(t, resp.State.GetAttribute(ctx, path.Root("products").AtListIndex(1).AtName("markets"), &addonMarkets).HasError())
	assert.False(t, addonMarkets.IsNull(), "markets must be an empty list, not null")
	assert.Empty(t, addonMarkets.Elements())
}

func TestProductAvailabilityDataSource_Read_CompanyError(t *testing.T) {
	ctx := context.Background()
	company := &mockCompanyService{err: errors.New("me failed")}
	availability := &mockProductAvailabilityService{}
	ds := newProductAvailabilityDataSource(company, availability)

	req, resp := productAvailabilityReadRequest(t, ds)
	ds.Read(ctx, req, resp)

	require.Equal(t, 1, resp.Diagnostics.ErrorsCount())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "me failed")
	assert.Empty(t, availability.gotCompany, "availability must not be called when the company lookup fails")
	assert.True(t, resp.State.Raw.IsNull(), "no state on error")
}

func TestProductAvailabilityDataSource_Read_AvailabilityError(t *testing.T) {
	ctx := context.Background()
	company := &mockCompanyService{result: &megaport.CurrentCompany{CompanyUID: "company-uid-1"}}
	availability := &mockProductAvailabilityService{err: errors.New("availability failed")}
	ds := newProductAvailabilityDataSource(company, availability)

	req, resp := productAvailabilityReadRequest(t, ds)
	ds.Read(ctx, req, resp)

	require.Equal(t, 1, resp.Diagnostics.ErrorsCount())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "availability failed")
	assert.True(t, resp.State.Raw.IsNull(), "no state on error")
}

func TestProductAvailabilityDataSource_Read_NoProducts(t *testing.T) {
	ctx := context.Background()
	company := &mockCompanyService{result: &megaport.CurrentCompany{CompanyUID: "company-uid-1"}}
	availability := &mockProductAvailabilityService{result: nil}
	ds := newProductAvailabilityDataSource(company, availability)

	req, resp := productAvailabilityReadRequest(t, ds)
	ds.Read(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var products types.List
	require.False(t, resp.State.GetAttribute(ctx, path.Root("products"), &products).HasError())
	assert.False(t, products.IsNull(), "products must be an empty list, not null")
	assert.False(t, products.IsUnknown())
	assert.Empty(t, products.Elements())
}

func TestAccMegaportProductAvailabilityDataSource_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "megaport_product_availability" "this" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.megaport_product_availability.this", "company_uid"),
					resource.TestCheckResourceAttrSet("data.megaport_product_availability.this", "products.#"),
				),
			},
		},
	})
}
