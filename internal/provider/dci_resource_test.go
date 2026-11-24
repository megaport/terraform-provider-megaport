package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	megaport "github.com/megaport/megaportgo"
)

const dciResourceName = "megaport_dci.dci"

// dciAccValues holds the values of a DCI configuration. A nil costCentre leaves the attribute out,
// and so does an autoRenew of false, which is the default.
type dciAccValues struct {
	name       string
	costCentre *string
	aEnd, bEnd int
	speed      int
	term       int
	autoRenew  bool
}

// config returns the configuration of a DCI with these values.
func (v dciAccValues) config() string {
	optional := ""
	if v.costCentre != nil {
		optional += fmt.Sprintf("cost_centre = %q\n", *v.costCentre)
	}
	if v.autoRenew {
		optional += "auto_renew = true\n"
	}
	return providerConfig + fmt.Sprintf(`
	resource "megaport_dci" "dci" {
		name                 = %q
		a_end_location_id    = %d
		b_end_location_id    = %d
		speed                = %d
		contract_term_months = %d
		%s
	}`, v.name, v.aEnd, v.bEnd, v.speed, v.term, optional)
}

// TestAccMegaportDCI_Basic orders a DCI, imports it, changes its name and cost centre,
// checks that other changes are refused at plan time, and checks that the destroy cancels it.
func TestAccMegaportDCI_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locations := findVXCPortTestLocations(t, 2)
	name := RandomTestName()
	costCentre := RandomTestName()
	nameNew := RandomTestName()
	costCentreNew := RandomTestName()

	created := dciAccValues{name: name, costCentre: &costCentre, aEnd: locations[0], bEnd: locations[1], speed: 1000, term: 12}
	updated := created
	updated.name, updated.costCentre = nameNew, &costCentreNew

	// ModifyPlan refuses each of these changes rather than planning a replace.
	speedChanged := updated
	speedChanged.speed = 10000
	locationsChanged := updated
	locationsChanged.aEnd, locationsChanged.bEnd = updated.bEnd, updated.aEnd
	termChanged := updated
	termChanged.term = 24
	autoRenewChanged := updated
	autoRenewChanged.autoRenew = true
	refused := regexp.MustCompile("Changing a DCI is not supported")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDCICanceled,
		Steps: []resource.TestStep{
			{
				Config: created.config(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dciResourceName, "name", name),
					resource.TestCheckResourceAttr(dciResourceName, "cost_centre", costCentre),
					resource.TestCheckResourceAttr(dciResourceName, "a_end_location_id", fmt.Sprint(created.aEnd)),
					resource.TestCheckResourceAttr(dciResourceName, "b_end_location_id", fmt.Sprint(created.bEnd)),
					resource.TestCheckResourceAttr(dciResourceName, "speed", "1000"),
					resource.TestCheckResourceAttr(dciResourceName, "contract_term_months", "12"),
					resource.TestCheckResourceAttr(dciResourceName, "auto_renew", "false"),
					resource.TestCheckResourceAttrSet(dciResourceName, "provisioning_status"),
					resource.TestCheckResourceAttrSet(dciResourceName, "solution_uid"),
					resource.TestCheckResourceAttrSet(dciResourceName, "a_end_port_uid"),
					resource.TestCheckResourceAttrSet(dciResourceName, "b_end_port_uid"),
					resource.TestCheckResourceAttrSet(dciResourceName, "vxc_uid"),
					checkDCIHolds(name, costCentre),
				),
			},
			{
				ResourceName:                         dciResourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "solution_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					return state.RootModule().Resources[dciResourceName].Primary.Attributes["solution_uid"], nil
				},
				ImportStateVerifyIgnore: []string{"provisioning_status"},
			},
			// The update sets the state from the plan, so checkDCIHolds reads the API to check the change was made.
			{
				Config: updated.config(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(dciResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dciResourceName, "name", nameNew),
					resource.TestCheckResourceAttr(dciResourceName, "cost_centre", costCentreNew),
					checkDCIHolds(nameNew, costCentreNew),
				),
			},
			{
				Config:      speedChanged.config(),
				ExpectError: refused,
			},
			{
				Config:      locationsChanged.config(),
				ExpectError: refused,
			},
			{
				Config:      termChanged.config(),
				ExpectError: refused,
			},
			{
				Config:      autoRenewChanged.config(),
				ExpectError: refused,
			},
		},
	})
}

// TestAccMegaportDCI_CostCentreRemoval checks that removing cost_centre from the configuration keeps the current value,
// and that an empty cost_centre clears it.
func TestAccMegaportDCI_CostCentreRemoval(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locations := findVXCPortTestLocations(t, 2)
	name := RandomTestName()
	costCentre := RandomTestName()
	noCostCentre := ""

	created := dciAccValues{name: name, costCentre: &costCentre, aEnd: locations[0], bEnd: locations[1], speed: 1000, term: 12}
	removed := created
	removed.costCentre = nil
	cleared := created
	cleared.costCentre = &noCostCentre

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDCICanceled,
		Steps: []resource.TestStep{
			{
				Config: created.config(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dciResourceName, "cost_centre", costCentre),
					checkDCIHolds(name, costCentre),
				),
			},
			// Removing cost_centre from the configuration keeps the current value.
			{
				Config: removed.config(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dciResourceName, "cost_centre", costCentre),
					checkDCIHolds(name, costCentre),
				),
			},
			// An empty cost_centre clears it.
			{
				Config: cleared.config(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(dciResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dciResourceName, "cost_centre", ""),
					checkDCIHolds(name, ""),
				),
			},
		},
	})
}

// checkDCIHolds fails when the DCI the API reports is not an active DCI with the given name and cost centre,
// or when one of its Ports or its VXC has a different cost centre. The API sets the cost centre on every member.
func checkDCIHolds(name, costCentre string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[dciResourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", dciResourceName)
		}
		client, err := getTestClient()
		if err != nil {
			return err
		}
		ctx := context.Background()

		uid := rs.Primary.Attributes["solution_uid"]
		solution, err := client.SolutionService.GetSolution(ctx, uid)
		if err != nil {
			return fmt.Errorf("could not read DCI %s: %w", uid, err)
		}
		if !solution.Active || solution.Type != megaport.SolutionTypeDCI {
			return fmt.Errorf("DCI %s has active %t and type %q, want an active %s", uid, solution.Active, solution.Type, megaport.SolutionTypeDCI)
		}
		if solution.Name != name || solution.ServiceLevelReference != costCentre {
			return fmt.Errorf("DCI %s has name %q and cost centre %q, want %q and %q",
				uid, solution.Name, solution.ServiceLevelReference, name, costCentre)
		}

		for _, attr := range []string{"a_end_port_uid", "b_end_port_uid"} {
			portUID := rs.Primary.Attributes[attr]
			port, err := client.PortService.GetPort(ctx, portUID)
			if err != nil {
				return fmt.Errorf("could not read Port %s of DCI %s: %w", portUID, uid, err)
			}
			if port.CostCentre != costCentre {
				return fmt.Errorf("Port %s of DCI %s has cost centre %q, want %q", portUID, uid, port.CostCentre, costCentre)
			}
		}
		vxcUID := rs.Primary.Attributes["vxc_uid"]
		vxc, err := client.VXCService.GetVXC(ctx, vxcUID)
		if err != nil {
			return fmt.Errorf("could not read VXC %s of DCI %s: %w", vxcUID, uid, err)
		}
		if vxc.CostCentre != costCentre {
			return fmt.Errorf("VXC %s of DCI %s has cost centre %q, want %q", vxcUID, uid, vxc.CostCentre, costCentre)
		}
		return nil
	}
}

// testAccCheckDCICanceled checks that every DCI in state is inactive, with both its Ports and its VXC canceled.
// GetSolution still reads a canceled DCI.
func testAccCheckDCICanceled(state *terraform.State) error {
	client, err := getTestClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	canceled := func(status string) bool {
		return status == megaport.STATUS_CANCELLED || status == megaport.STATUS_DECOMMISSIONED
	}

	for _, rs := range state.RootModule().Resources {
		if rs.Type != "megaport_dci" {
			continue
		}
		uid := rs.Primary.Attributes["solution_uid"]
		solution, err := client.SolutionService.GetSolution(ctx, uid)
		if err != nil {
			return fmt.Errorf("could not read DCI %s after destroy: %w", uid, err)
		}
		if solution.Active {
			return fmt.Errorf("DCI %s is still active after destroy", uid)
		}

		// A cancel that stops part-way can leave a member live while the DCI reads inactive.
		for _, attr := range []string{"a_end_port_uid", "b_end_port_uid"} {
			portUID := rs.Primary.Attributes[attr]
			if portUID == "" {
				continue
			}
			port, err := client.PortService.GetPort(ctx, portUID)
			if err != nil {
				return fmt.Errorf("could not read Port %s of DCI %s after destroy: %w", portUID, uid, err)
			}
			if !canceled(port.ProvisioningStatus) {
				return fmt.Errorf("Port %s of DCI %s has status %s after destroy", portUID, uid, port.ProvisioningStatus)
			}
		}
		if vxcUID := rs.Primary.Attributes["vxc_uid"]; vxcUID != "" {
			vxc, err := client.VXCService.GetVXC(ctx, vxcUID)
			if err != nil {
				return fmt.Errorf("could not read VXC %s of DCI %s after destroy: %w", vxcUID, uid, err)
			}
			if !canceled(vxc.ProvisioningStatus) {
				return fmt.Errorf("VXC %s of DCI %s has status %s after destroy", vxcUID, uid, vxc.ProvisioningStatus)
			}
		}
	}
	return nil
}
