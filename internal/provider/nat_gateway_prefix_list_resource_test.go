package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

// The schema defaults ge and le to 0. A 0 must not reach the API, which
// rejects it on every prefix except /0.
func TestNATGatewayPrefixListToAPIRequest_ZeroBoundIsUnset(t *testing.T) {
	ctx := context.Background()
	entries, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(natGatewayPrefixListEntryAttrs), []natGatewayPrefixListEntryModel{
		{Action: types.StringValue("deny"), Prefix: types.StringValue("192.168.0.0/16"), Ge: types.Int64Value(0), Le: types.Int64Value(0)},
		{Action: types.StringValue("permit"), Prefix: types.StringValue("10.0.0.0/8"), Ge: types.Int64Value(24), Le: types.Int64Value(32)},
	})
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	m := &natGatewayPrefixListResourceModel{
		Description:   types.StringValue("test"),
		AddressFamily: types.StringValue("IPv4"),
		Entries:       entries,
	}

	out, diags := m.toAPIRequest(ctx)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	require.Len(t, out.Entries, 2)
	require.Nil(t, out.Entries[0].Ge)
	require.Nil(t, out.Entries[0].Le)
	require.Equal(t, megaport.PtrTo(24), out.Entries[1].Ge)
	require.Equal(t, megaport.PtrTo(32), out.Entries[1].Le)
}

// The API omits a bound that nothing set, which state holds as the schema default of 0.
func TestNATGatewayPrefixListFromAPI_AbsentBoundIsZero(t *testing.T) {
	ctx := context.Background()
	m := &natGatewayPrefixListResourceModel{}
	diags := m.fromAPI(ctx, &megaport.NATGatewayPrefixList{
		ID:            1,
		Description:   "test",
		AddressFamily: "IPv4",
		Entries: []megaport.NATGatewayPrefixListEntry{
			{Action: "deny", Prefix: "192.168.0.0/16"},
			{Action: "permit", Prefix: "10.0.0.0/8", Ge: megaport.PtrTo(24), Le: megaport.PtrTo(32)},
		},
	})
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	var entries []natGatewayPrefixListEntryModel
	require.False(t, m.Entries.ElementsAs(ctx, &entries, false).HasError())
	require.Len(t, entries, 2)
	require.Equal(t, types.Int64Value(0), entries[0].Ge)
	require.Equal(t, types.Int64Value(0), entries[0].Le)
	require.Equal(t, types.Int64Value(24), entries[1].Ge)
	require.Equal(t, types.Int64Value(32), entries[1].Le)
}

// TestAccMegaportNATGatewayPrefixList_Basic exercises the create → read →
// import → update lifecycle of megaport_nat_gateway_prefix_list against a
// freshly-provisioned NAT Gateway in staging, including the ge/le numeric
// round-trip (the API sends these as strings but the SDK normalises them).
func TestAccMegaportNATGatewayPrefixList_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	speed, sessionCount, err := getNATGatewayTestConfig()
	if err != nil {
		t.Skipf("Skipping NAT Gateway prefix list test: %v", err)
	}
	locationID, _ := findNATGatewayTestLocation(t, speed)
	natGWName := RandomTestName()
	plDescription := RandomTestName()
	plDescriptionUpdated := RandomTestName()

	natResourceName := "megaport_nat_gateway.test"
	plResourceName := "megaport_nat_gateway_prefix_list.test"

	configInitial := providerConfig + fmt.Sprintf(`
data "megaport_location" "test_location" {
    id = %d
}

resource "megaport_nat_gateway" "test" {
    product_name         = "%s"
    location_id          = data.megaport_location.test_location.id
    speed                = %d
    session_count        = %d
    contract_term_months = 1
    diversity_zone       = "red"
    asn                  = 64512
}

resource "megaport_nat_gateway_prefix_list" "test" {
    nat_gateway_product_uid = megaport_nat_gateway.test.product_uid
    description             = "%s"
    address_family          = "IPv4"
    entries = [
        {
            action = "permit"
            prefix = "10.0.0.0/8"
            ge     = 24
            le     = 32
        },
        {
            action = "deny"
            prefix = "192.168.0.0/16"
        }
    ]
}
`, locationID, natGWName, speed, sessionCount, plDescription)

	configUpdated := providerConfig + fmt.Sprintf(`
data "megaport_location" "test_location" {
    id = %d
}

resource "megaport_nat_gateway" "test" {
    product_name         = "%s"
    location_id          = data.megaport_location.test_location.id
    speed                = %d
    session_count        = %d
    contract_term_months = 1
    diversity_zone       = "red"
    asn                  = 64512
}

resource "megaport_nat_gateway_prefix_list" "test" {
    nat_gateway_product_uid = megaport_nat_gateway.test.product_uid
    description             = "%s"
    address_family          = "IPv4"
    entries = [
        {
            action = "permit"
            prefix = "10.0.0.0/8"
            ge     = 25
            le     = 32
        },
        {
            action = "permit"
            prefix = "172.16.0.0/12"
            ge     = 24
            le     = 28
        },
        {
            action = "deny"
            prefix = "192.168.0.0/16"
        }
    ]
}
`, locationID, natGWName, speed, sessionCount, plDescriptionUpdated)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create + verify
			{
				Config: configInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkNATGatewayProvisioned(natResourceName),
					resource.TestCheckResourceAttr(plResourceName, "description", plDescription),
					resource.TestCheckResourceAttr(plResourceName, "address_family", "IPv4"),
					resource.TestCheckResourceAttr(plResourceName, "entries.#", "2"),
					resource.TestCheckResourceAttr(plResourceName, "entries.0.action", "permit"),
					resource.TestCheckResourceAttr(plResourceName, "entries.0.prefix", "10.0.0.0/8"),
					resource.TestCheckResourceAttr(plResourceName, "entries.0.ge", "24"),
					resource.TestCheckResourceAttr(plResourceName, "entries.0.le", "32"),
					resource.TestCheckResourceAttr(plResourceName, "entries.1.action", "deny"),
					resource.TestCheckResourceAttr(plResourceName, "entries.1.prefix", "192.168.0.0/16"),
					resource.TestCheckResourceAttrSet(plResourceName, "id"),
					resource.TestCheckResourceAttrPair(
						plResourceName, "nat_gateway_product_uid",
						natResourceName, "product_uid",
					),
				),
			},
			// ImportState
			{
				ResourceName:                         plResourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources[plResourceName]
					if !ok {
						return "", fmt.Errorf("resource %s not found", plResourceName)
					}
					return fmt.Sprintf("%s:%s",
						rs.Primary.Attributes["nat_gateway_product_uid"],
						rs.Primary.Attributes["id"],
					), nil
				},
			},
			// Update
			{
				Config: configUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(plResourceName, "description", plDescriptionUpdated),
					resource.TestCheckResourceAttr(plResourceName, "entries.#", "3"),
					resource.TestCheckResourceAttr(plResourceName, "entries.0.ge", "25"),
					resource.TestCheckResourceAttr(plResourceName, "entries.1.prefix", "172.16.0.0/12"),
					resource.TestCheckResourceAttr(plResourceName, "entries.1.ge", "24"),
					resource.TestCheckResourceAttr(plResourceName, "entries.1.le", "28"),
					resource.TestCheckResourceAttr(plResourceName, "entries.2.action", "deny"),
				),
			},
		},
	})
}

// TestAccMegaportNATGatewayPrefixList_IPv6 covers the IPv6 address family path.
func TestAccMegaportNATGatewayPrefixList_IPv6(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	speed, sessionCount, err := getNATGatewayTestConfig()
	if err != nil {
		t.Skipf("Skipping NAT Gateway prefix list IPv6 test: %v", err)
	}
	locationID, _ := findNATGatewayTestLocation(t, speed)
	natGWName := RandomTestName()
	plDescription := RandomTestName()

	natResourceName := "megaport_nat_gateway.test"
	plResourceName := "megaport_nat_gateway_prefix_list.test"

	config := providerConfig + fmt.Sprintf(`
data "megaport_location" "test_location" {
    id = %d
}

resource "megaport_nat_gateway" "test" {
    product_name         = "%s"
    location_id          = data.megaport_location.test_location.id
    speed                = %d
    session_count        = %d
    contract_term_months = 1
    diversity_zone       = "red"
    asn                  = 64512
}

resource "megaport_nat_gateway_prefix_list" "test" {
    nat_gateway_product_uid = megaport_nat_gateway.test.product_uid
    description             = "%s"
    address_family          = "IPv6"
    entries = [
        {
            action = "permit"
            prefix = "2001:db8::/32"
            ge     = 48
            le     = 64
        },
        {
            action = "deny"
            prefix = "fc00::/7"
        }
    ]
}
`, locationID, natGWName, speed, sessionCount, plDescription)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkNATGatewayProvisioned(natResourceName),
					resource.TestCheckResourceAttr(plResourceName, "address_family", "IPv6"),
					resource.TestCheckResourceAttr(plResourceName, "entries.#", "2"),
					resource.TestCheckResourceAttr(plResourceName, "entries.0.prefix", "2001:db8::/32"),
					resource.TestCheckResourceAttr(plResourceName, "entries.0.ge", "48"),
					resource.TestCheckResourceAttr(plResourceName, "entries.0.le", "64"),
					resource.TestCheckResourceAttr(plResourceName, "entries.1.prefix", "fc00::/7"),
				),
			},
		},
	})
}
