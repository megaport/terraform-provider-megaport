// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	megaport "github.com/megaport/megaportgo"
)

// TestMain wires up the terraform-plugin-testing sweeper framework. With no
// -sweep flag it just runs the tests, so normal suites are unaffected.
func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	// VXCs attach to MCR/MVE/port/NAT gateway and IXs to MCR/MVE/port, so
	// both are swept first. MCR, MVE, port, and NAT gateway don't attach to
	// one another, so their relative order doesn't matter.
	resource.AddTestSweepers("megaport_vxc", &resource.Sweeper{
		Name: "megaport_vxc",
		F:    sweepResource(cleanupOrphanedVXCs),
	})
	resource.AddTestSweepers("megaport_ix", &resource.Sweeper{
		Name: "megaport_ix",
		F:    sweepResource(cleanupOrphanedIXs),
	})
	resource.AddTestSweepers("megaport_mcr", &resource.Sweeper{
		Name:         "megaport_mcr",
		Dependencies: []string{"megaport_vxc", "megaport_ix"},
		F:            sweepResource(cleanupOrphanedMCRs),
	})
	resource.AddTestSweepers("megaport_mve", &resource.Sweeper{
		Name:         "megaport_mve",
		Dependencies: []string{"megaport_vxc", "megaport_ix"},
		F:            sweepResource(cleanupOrphanedMVEs),
	})
	resource.AddTestSweepers("megaport_port", &resource.Sweeper{
		Name:         "megaport_port",
		Dependencies: []string{"megaport_vxc", "megaport_ix"},
		F:            sweepResource(cleanupOrphanedPorts),
	})
	resource.AddTestSweepers("megaport_nat_gateway", &resource.Sweeper{
		Name:         "megaport_nat_gateway",
		Dependencies: []string{"megaport_vxc"},
		F:            sweepResource(cleanupOrphanedNATGateways),
	})
	// cleanupOrphanedPorts already covers LAG ports (ListPorts returns them
	// alongside single ports). This is a no-op alias depending on
	// megaport_port so -sweep=megaport_lag_port still works without running
	// cleanupOrphanedPorts a second time under -sweep=all.
	resource.AddTestSweepers("megaport_lag_port", &resource.Sweeper{
		Name:         "megaport_lag_port",
		Dependencies: []string{"megaport_port"},
		F:            func(_ string) error { return nil },
	})
}

// resourceCleaner lists live orphans of one resource type whose name carries
// TestNamePrefix and, when del is true, deletes them. It returns the count of
// live orphans found.
//
// Endpoint resources (MCR/MVE/port/NAT gateway) delete with SafeDelete so the
// API refuses when something is still attached, guarding against a
// force-delete cascading into a resource that lacks the test prefix. A parent
// still carrying an attachment is left for a later sweep rather than
// force-cancelled.
type resourceCleaner func(ctx context.Context, client *megaport.Client, del bool) (int, error)

// sweepResource adapts a resourceCleaner to the framework's SweeperFunc. The
// region argument is unused: the provider has no regions.
func sweepResource(clean resourceCleaner) resource.SweeperFunc {
	return func(_ string) error {
		client, err := getTestClient()
		if err != nil {
			return err
		}
		_, err = clean(context.Background(), client, true)
		return err
	}
}

// sweepable reports whether a resource is a live orphan created by the suite.
// Only resources whose name carries TestNamePrefix (see RandomTestName) are
// ever swept.
func sweepable(name, status string) bool {
	return strings.HasPrefix(name, TestNamePrefix) && !strings.EqualFold(status, "DECOMMISSIONED")
}

func cleanupOrphanedVXCs(ctx context.Context, client *megaport.Client, del bool) (int, error) {
	vxcs, err := client.VXCService.ListVXCs(ctx, &megaport.ListVXCsRequest{})
	if err != nil {
		return 0, fmt.Errorf("listing VXCs: %w", err)
	}
	var n int
	for _, vxc := range vxcs {
		if !sweepable(vxc.Name, vxc.ProvisioningStatus) {
			continue
		}
		n++
		log.Printf("[sweep] VXC  %s (%s) status=%s", vxc.Name, vxc.UID, vxc.ProvisioningStatus)
		if !del {
			continue
		}
		if delErr := client.VXCService.DeleteVXC(ctx, vxc.UID, &megaport.DeleteVXCRequest{DeleteNow: true}); delErr != nil {
			log.Printf("[sweep]   delete failed: %v", delErr)
		}
	}
	return n, nil
}

func cleanupOrphanedMVEs(ctx context.Context, client *megaport.Client, del bool) (int, error) {
	mves, err := client.MVEService.ListMVEs(ctx, &megaport.ListMVEsRequest{})
	if err != nil {
		return 0, fmt.Errorf("listing MVEs: %w", err)
	}
	var n int
	for _, mve := range mves {
		if !sweepable(mve.Name, mve.ProvisioningStatus) {
			continue
		}
		n++
		log.Printf("[sweep] MVE  %s (%s) status=%s", mve.Name, mve.UID, mve.ProvisioningStatus)
		if !del {
			continue
		}
		if _, delErr := client.MVEService.DeleteMVE(ctx, &megaport.DeleteMVERequest{MVEID: mve.UID, SafeDelete: true}); delErr != nil {
			log.Printf("[sweep]   delete failed: %v", delErr)
		}
	}
	return n, nil
}

func cleanupOrphanedMCRs(ctx context.Context, client *megaport.Client, del bool) (int, error) {
	mcrs, err := client.MCRService.ListMCRs(ctx, &megaport.ListMCRsRequest{})
	if err != nil {
		return 0, fmt.Errorf("listing MCRs: %w", err)
	}
	var n int
	for _, mcr := range mcrs {
		if !sweepable(mcr.Name, mcr.ProvisioningStatus) {
			continue
		}
		n++
		log.Printf("[sweep] MCR  %s (%s) status=%s", mcr.Name, mcr.UID, mcr.ProvisioningStatus)
		if !del {
			continue
		}
		if _, delErr := client.MCRService.DeleteMCR(ctx, &megaport.DeleteMCRRequest{MCRID: mcr.UID, DeleteNow: true, SafeDelete: true}); delErr != nil {
			log.Printf("[sweep]   delete failed: %v", delErr)
		}
	}
	return n, nil
}

func cleanupOrphanedIXs(ctx context.Context, client *megaport.Client, del bool) (int, error) {
	ixs, err := client.IXService.ListIXs(ctx, &megaport.ListIXsRequest{})
	if err != nil {
		return 0, fmt.Errorf("listing IXs: %w", err)
	}
	var n int
	for _, ix := range ixs {
		if !sweepable(ix.ProductName, ix.ProvisioningStatus) {
			continue
		}
		n++
		log.Printf("[sweep] IX   %s (%s) status=%s", ix.ProductName, ix.ProductUID, ix.ProvisioningStatus)
		if !del {
			continue
		}
		if delErr := client.IXService.DeleteIX(ctx, ix.ProductUID, &megaport.DeleteIXRequest{DeleteNow: true}); delErr != nil {
			log.Printf("[sweep]   delete failed: %v", delErr)
		}
	}
	return n, nil
}

func cleanupOrphanedPorts(ctx context.Context, client *megaport.Client, del bool) (int, error) {
	ports, err := client.PortService.ListPorts(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing ports: %w", err)
	}
	var n int
	for _, port := range ports {
		if !sweepable(port.Name, port.ProvisioningStatus) {
			continue
		}
		n++
		log.Printf("[sweep] Port %s (%s) status=%s", port.Name, port.UID, port.ProvisioningStatus)
		if !del {
			continue
		}
		if _, delErr := client.PortService.DeletePort(ctx, &megaport.DeletePortRequest{PortID: port.UID, DeleteNow: true, SafeDelete: true}); delErr != nil {
			log.Printf("[sweep]   delete failed: %v", delErr)
		}
	}
	return n, nil
}

func cleanupOrphanedNATGateways(ctx context.Context, client *megaport.Client, del bool) (int, error) {
	gws, err := client.NATGatewayService.ListNATGateways(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing NAT Gateways: %w", err)
	}
	var n int
	for _, gw := range gws {
		if !sweepable(gw.ProductName, gw.ProvisioningStatus) {
			continue
		}
		n++
		log.Printf("[sweep] NGW  %s (%s) status=%s", gw.ProductName, gw.ProductUID, gw.ProvisioningStatus)
		if !del {
			continue
		}
		// DeleteNATGateway has no SafeDelete option, so provisioned gateways
		// go through DeleteProduct directly to keep the attachment guard.
		// DESIGN records were never purchased (nothing attached) and need the
		// design-only delete path that DeleteNATGateway routes to.
		var delErr error
		if strings.EqualFold(gw.ProvisioningStatus, megaport.STATUS_DESIGN) {
			delErr = client.NATGatewayService.DeleteNATGateway(ctx, gw.ProductUID)
		} else {
			_, delErr = client.ProductService.DeleteProduct(ctx, &megaport.DeleteProductRequest{ProductID: gw.ProductUID, DeleteNow: true, SafeDelete: true})
		}
		if delErr != nil {
			log.Printf("[sweep]   delete failed: %v", delErr)
		}
	}
	return n, nil
}
