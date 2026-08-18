package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	megaport "github.com/megaport/megaportgo"
)

func TestIsIPsecTunnelsConfiguredError(t *testing.T) {
	t.Parallel()
	apiErr := func(status int, msg string) error {
		return &megaport.ErrorResponse{Response: &http.Response{StatusCode: status}, Message: msg}
	}
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"400 configured tunnels", apiErr(http.StatusBadRequest, "Invalid ipsec_tunnels. Requested limit is less than number of configured tunnels."), true},
		{"400 configured tunnels wrapped", fmt.Errorf("delete failed: %w", apiErr(http.StatusBadRequest, "number of configured tunnels")), true},
		{"400 unrelated message", apiErr(http.StatusBadRequest, "Could not find a service with UID"), false},
		{"404 configured tunnels", apiErr(http.StatusNotFound, "configured tunnels"), false},
		{"plain error", errors.New("configured tunnels"), false},
		{"nil response", &megaport.ErrorResponse{Message: "configured tunnels"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isIPsecTunnelsConfiguredError(tc.err); got != tc.want {
				t.Errorf("isIPsecTunnelsConfiguredError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAccMegaportMCRIpsecAddon_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	costCentreName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create MCR and IPSec add-on with 10 tunnels
			{
				Config: providerConfig + testAccMCRIpsecAddonConfig(locationID, mcrName, costCentreName, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_ipsec_addon.test", "tunnel_count", "10"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.test", "add_on_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.test", "mcr_id"),
				),
			},
			// Plan-only check — no drift
			{
				Config:             providerConfig + testAccMCRIpsecAddonConfig(locationID, mcrName, costCentreName, 10),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update to 20 tunnels
			{
				Config: providerConfig + testAccMCRIpsecAddonConfig(locationID, mcrName, costCentreName, 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_ipsec_addon.test", "tunnel_count", "20"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.test", "add_on_uid"),
				),
			},
			// Import
			{
				ResourceName:                         "megaport_mcr_ipsec_addon.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "add_on_uid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["megaport_mcr_ipsec_addon.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: megaport_mcr_ipsec_addon.test")
					}
					return rs.Primary.Attributes["mcr_id"] + ":" + rs.Primary.Attributes["add_on_uid"], nil
				},
			},
		},
	})
}

func TestParseImportIDStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantLeft  string
		wantRight string
		wantErr   bool
	}{
		{
			name:      "valid",
			input:     "mcr-uid:addon-uid",
			wantLeft:  "mcr-uid",
			wantRight: "addon-uid",
		},
		{
			name:    "missing colon",
			input:   "no-colon-here",
			wantErr: true,
		},
		{
			name:    "empty left",
			input:   ":addon-uid",
			wantErr: true,
		},
		{
			name:    "empty right",
			input:   "mcr-uid:",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "multiple colons",
			input:   "a:b:c",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			left, right, err := parseImportIDStrings(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if left != tc.wantLeft || right != tc.wantRight {
				t.Fatalf("parseImportIDStrings(%q) = (%q, %q), want (%q, %q)", tc.input, left, right, tc.wantLeft, tc.wantRight)
			}
		})
	}
}

func testAccMCRIpsecAddonConfig(locationID int, mcrName, costCentreName string, tunnelCount int) string {
	return fmt.Sprintf(`
data "megaport_location" "test_location" {
	id = %d
}

resource "megaport_mcr" "mcr" {
	product_name         = "%s"
	port_speed           = 1000
	location_id          = data.megaport_location.test_location.id
	contract_term_months = 1
	cost_centre          = "%s"

	prefix_filter_lists = []

	lifecycle {
		ignore_changes = [prefix_filter_lists]
	}
}

resource "megaport_mcr_ipsec_addon" "test" {
	mcr_id       = megaport_mcr.mcr.product_uid
	tunnel_count = %d
}
`, locationID, mcrName, costCentreName, tunnelCount)
}

// TestMCRIpsecAddonImportState checks that ImportState seeds the two
// identifiers and nothing else, leaving the rest to Read. The nil client is the
// assertion that it makes no API call: any call would panic.
func TestMCRIpsecAddonImportState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mcrIpsecAddonResource{}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	objType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	// The framework hands ImportState a wholly null state and rejects the
	// response if it comes back unchanged.
	emptyState := tftypes.NewValue(objType, nil)

	resp := fwresource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: emptyState.Copy()}}
	r.ImportState(ctx, fwresource.ImportStateRequest{ID: "mcr-uid-123:addon-uid-456"}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
	}
	if resp.State.Raw.Equal(emptyState) {
		t.Fatal("ImportState wrote no state, which the framework rejects")
	}

	var got mcrIpsecAddonResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading imported state: %v", diags.Errors())
	}
	if got.MCRID.ValueString() != "mcr-uid-123" {
		t.Errorf("mcr_id = %q, want %q", got.MCRID.ValueString(), "mcr-uid-123")
	}
	if got.AddOnUID.ValueString() != "addon-uid-456" {
		t.Errorf("add_on_uid = %q, want %q", got.AddOnUID.ValueString(), "addon-uid-456")
	}
	if !got.TunnelCount.IsNull() {
		t.Errorf("tunnel_count is set after import; Read populates it")
	}

	// A malformed ID still fails here, before Read runs.
	bad := fwresource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: emptyState.Copy()}}
	r.ImportState(ctx, fwresource.ImportStateRequest{ID: "no-separator"}, &bad)
	if !bad.Diagnostics.HasError() {
		t.Error("malformed import ID: want an error diagnostic, got none")
	}
}
