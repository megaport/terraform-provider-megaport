package provider

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	megaport "github.com/megaport/megaportgo"
)

// ── Types ─────────────────────────────────────────────────────────────────────

type cspCredentials struct {
	AzureServiceKeys          []string `json:"azure_service_keys"`
	AzureServiceKeysWithPeers []string `json:"azure_service_keys_with_peers"`
	GooglePairingKeys         []string `json:"google_pairing_keys"`
}

// ── MVE Location Picker ───────────────────────────────────────────────────────

// mveTestLocationCandidates is a curated list of staging location IDs with known
// MVE demo capacity, ordered roughly by historical reliability across regions.
// Refresh from internal Metabase capacity data as needed.
var mveTestLocationCandidates = []int{
	// AU/NZ
	3, 4, 5, 2, 10, 50, 383, 454,
	// US
	59, 61, 67, 69, 71, 57, 66, 68, 116, 226, 321, 346, 77, 79, 100, 320, 354, 380, 530,
	// EU
	85, 88, 90, 130, 131, 94, 527, 515, 256, 637, 298, 518, 430, 917,
	// APAC
	36, 37, 46, 54, 155, 560, 558, 571, 572, 257,
	// Americas (non-US)
	93, 419, 573, 1484,
}

// findMVETestLocation returns a staging location ID with confirmed MVE capacity,
// validated via the API validate endpoint. Calls t.Skip if none found.
//
// Strategy: try the curated candidate list first (fast path), then fall back to
// a full sweep of all locations. Each candidate is probed with ValidateMVEOrder
// using a minimal Aruba SMALL config — the validate endpoint is the only reliable
// way to check actual demo capacity in staging since mveMaxCpuCoreCount is not
// populated by the API.
//
//nolint:unparam // minCPUCores kept for future use when API populates the field
func findMVETestLocation(t *testing.T, minCPUCores int) (id int, name string) {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return 0, ""
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("skipping: could not list locations: %v", err)
		return 0, ""
	}

	byID := make(map[int]*megaport.LocationV3, len(locations))
	for _, loc := range locations {
		byID[loc.ID] = loc
	}

	probe := func(loc *megaport.LocationV3) bool {
		if !strings.EqualFold(loc.Status, "active") || !loc.HasMVESupport() {
			return false
		}
		err := client.MVEService.ValidateMVEOrder(ctx, &megaport.BuyMVERequest{
			LocationID: loc.ID,
			Name:       "probe",
			Term:       1,
			VendorConfig: &megaport.ArubaConfig{
				Vendor:      "aruba",
				ImageID:     MVEArubaImageIDMVE,
				ProductSize: "SMALL",
				AccountName: "probe",
				AccountKey:  "probe",
				SystemTag:   "Preconfiguration-aruba-test-1",
			},
			Vnics: []megaport.MVENetworkInterface{
				{Description: "Data Plane"},
				{Description: "Control Plane"},
			},
		})
		return err == nil
	}

	// Fast path: curated candidates
	for _, candidateID := range mveTestLocationCandidates {
		if loc, ok := byID[candidateID]; ok && probe(loc) {
			t.Logf("findMVETestLocation: using location %d (%s)", loc.ID, loc.Name)
			return loc.ID, loc.Name
		}
	}

	// Slow path: full sweep
	for _, loc := range locations {
		if probe(loc) {
			t.Logf("findMVETestLocation: using location %d (%s) (fallback sweep)", loc.ID, loc.Name)
			return loc.ID, loc.Name
		}
	}

	t.Skip("skipping: no location with available MVE capacity found in staging")
	return 0, ""
}

// ── Port/MCR Location Pickers ─────────────────────────────────────────────────

// findPortTestLocation returns a staging location ID that supports Megaport
// ports at the given speed (Mbps). Calls t.Skip if none found.
//
//nolint:unparam // name return is available for callers that want it
func findPortTestLocation(t *testing.T, speedMbps int) (id int, name string) {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return 0, ""
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("skipping: could not list locations: %v", err)
		return 0, ""
	}
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") && portLocationHasCapacity(loc, speedMbps) {
			t.Logf("findPortTestLocation: using location %d (%s)", loc.ID, loc.Name)
			return loc.ID, loc.Name
		}
	}
	t.Skipf("skipping: no ACTIVE location with %d Mbps Megaport port capacity", speedMbps)
	return 0, ""
}

// findMCRTestLocation returns a staging location ID that supports MCR at the
// given speed (Mbps). Calls t.Skip if none found.
//
//nolint:unparam // speedMbps is always 2500 today; kept for future test flexibility
func findMCRTestLocation(t *testing.T, speedMbps int) (id int, name string) {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return 0, ""
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("skipping: could not list locations: %v", err)
		return 0, ""
	}
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") && mcrLocationHasCapacity(loc, speedMbps) {
			t.Logf("findMCRTestLocation: using location %d (%s)", loc.ID, loc.Name)
			return loc.ID, loc.Name
		}
	}
	t.Skipf("skipping: no ACTIVE location with %d Mbps MCR capacity", speedMbps)
	return 0, ""
}

// portLocationHasCapacity returns true when at least one diversity zone at loc
// lists speedMbps (or higher) in MegaportSpeedMbps.
func portLocationHasCapacity(loc *megaport.LocationV3, speedMbps int) bool {
	if loc.DiversityZones == nil {
		return false
	}
	check := func(zone *megaport.LocationV3DiversityZone) bool {
		if zone == nil {
			return false
		}
		for _, s := range zone.MegaportSpeedMbps {
			if s >= speedMbps {
				return true
			}
		}
		return false
	}
	return check(loc.DiversityZones.Red) || check(loc.DiversityZones.Blue)
}

// mcrLocationHasCapacity returns true when at least one diversity zone at loc
// lists speedMbps (or higher) in McrSpeedMbps.
func mcrLocationHasCapacity(loc *megaport.LocationV3, speedMbps int) bool {
	if loc.DiversityZones == nil {
		return false
	}
	check := func(zone *megaport.LocationV3DiversityZone) bool {
		if zone == nil {
			return false
		}
		for _, s := range zone.McrSpeedMbps {
			if s >= speedMbps {
				return true
			}
		}
		return false
	}
	return check(loc.DiversityZones.Red) || check(loc.DiversityZones.Blue)
}

// ── CSP Credential Pickers ────────────────────────────────────────────────────

// pickAzureServiceKey returns the first Azure service key from the pool that
// has available VXC capacity, validated via /v2/secure/azure/{key}. Keys are
// shuffled before probing so concurrent test suites are unlikely to race on the
// same key. Calls t.Skip if no valid key is found.
func pickAzureServiceKey(t *testing.T) string {
	t.Helper()
	creds := loadCSPCredentials()
	if len(creds.AzureServiceKeys) == 0 {
		t.Skip("skipping: no Azure service keys in testdata/csp_credentials.json")
		return ""
	}
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return ""
	}

	//nolint:gosec // weak random is fine for test key shuffling
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	keys := make([]string, len(creds.AzureServiceKeys))
	copy(keys, creds.AzureServiceKeys)
	r.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })

	for _, key := range keys {
		_, err := client.VXCService.LookupPartnerPorts(ctx, &megaport.LookupPartnerPortsRequest{
			Partner:   "AZURE",
			Key:       key,
			PortSpeed: 1000,
		})
		if err == nil {
			t.Logf("pickAzureServiceKey: using key %s", key)
			return key
		}
		t.Logf("pickAzureServiceKey: key %s unavailable: %v", key, err)
	}

	t.Skip("skipping: no Azure service key with available capacity found")
	return ""
}

// pickGCPPairingKey returns the first GCP pairing key from the pool that has
// available VXC capacity, validated via /v2/secure/google/{key}. Keys are
// shuffled before probing. Calls t.Skip if no valid key is found.
func pickGCPPairingKey(t *testing.T) string {
	t.Helper()
	creds := loadCSPCredentials()
	if len(creds.GooglePairingKeys) == 0 {
		t.Skip("skipping: no GCP pairing keys in testdata/csp_credentials.json")
		return ""
	}
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return ""
	}

	//nolint:gosec // weak random is fine for test key shuffling
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	keys := make([]string, len(creds.GooglePairingKeys))
	copy(keys, creds.GooglePairingKeys)
	r.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })

	for _, key := range keys {
		_, err := client.VXCService.LookupPartnerPorts(ctx, &megaport.LookupPartnerPortsRequest{
			Partner:   "GOOGLE",
			Key:       key,
			PortSpeed: 1000,
		})
		if err == nil {
			t.Logf("pickGCPPairingKey: using key %s", key)
			return key
		}
		t.Logf("pickGCPPairingKey: key %s unavailable: %v", key, err)
	}

	t.Skip("skipping: no GCP pairing key with available capacity found")
	return ""
}

func loadCSPCredentials() cspCredentials {
	data, err := os.ReadFile("testdata/csp_credentials.json")
	if err != nil {
		return cspCredentials{}
	}
	var creds cspCredentials
	_ = json.Unmarshal(data, &creds)
	return creds
}

// ── Staging Health Check ──────────────────────────────────────────────────────

// TestStagingHealthCheck verifies staging environment preconditions.
// Run before a full acceptance suite to catch problems early:
//
//	go test -v -run TestStagingHealthCheck ./internal/provider/
func TestStagingHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("health check requires API access")
	}
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Fatalf("staging API unreachable: %v", err)
	}

	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Fatalf("could not list locations: %v", err)
	}

	// MVE capacity — count locations that declare MVE support. Note: staging does not
	// populate mveMaxCpuCoreCount, so this is an approximation based on the MveAvailable
	// flag. Actual capacity is only confirmed by findMVETestLocation via ValidateMVEOrder.
	mveCount := 0
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") && loc.HasMVESupport() {
			mveCount++
		}
	}
	t.Logf("Locations reporting MVE support (approximate — staging does not populate mveMaxCpuCoreCount): %d", mveCount)
	if mveCount == 0 {
		t.Error("WARN: no MVE capacity available — MVE tests will be skipped")
	}

	// Port capacity (1G)
	portCount := 0
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") && portLocationHasCapacity(loc, 1000) {
			portCount++
		}
	}
	t.Logf("Locations with 1000 Mbps port capacity: %d", portCount)
	if portCount == 0 {
		t.Error("WARN: no 1G port capacity available — port tests will be skipped")
	}

	// MCR capacity (2500 Mbps)
	mcrCount := 0
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") && mcrLocationHasCapacity(loc, 2500) {
			mcrCount++
		}
	}
	t.Logf("Locations with 2500 Mbps MCR capacity: %d", mcrCount)
	if mcrCount == 0 {
		t.Error("WARN: no 2.5G MCR capacity available — MCR tests will be skipped")
	}

	// Partner ports
	ports, err := client.PartnerService.ListPartnerMegaports(ctx)
	if err != nil {
		t.Errorf("could not list partner ports: %v", err)
	} else {
		for _, partner := range []string{"AWS", "AZURE", "GOOGLE"} {
			found := false
			for _, p := range ports {
				if p.ConnectType == partner && p.VXCPermitted {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("WARN: no %s partner port found — %s VXC tests will be skipped", partner, partner)
			}
		}
	}

	// CSP credentials pool — probe each key for live capacity
	creds := loadCSPCredentials()
	t.Logf("Azure service keys in pool: %d", len(creds.AzureServiceKeys))
	t.Logf("GCP pairing keys in pool: %d", len(creds.GooglePairingKeys))

	azureAvailable := 0
	for _, key := range creds.AzureServiceKeys {
		_, err := client.VXCService.LookupPartnerPorts(ctx, &megaport.LookupPartnerPortsRequest{
			Partner:   "AZURE",
			Key:       key,
			PortSpeed: 1000,
		})
		if err == nil {
			azureAvailable++
		}
	}
	t.Logf("Azure service keys with available capacity: %d/%d", azureAvailable, len(creds.AzureServiceKeys))
	if azureAvailable == 0 {
		t.Error("WARN: no Azure service key has available capacity — Azure VXC tests will be skipped")
	}

	gcpAvailable := 0
	for _, key := range creds.GooglePairingKeys {
		_, err := client.VXCService.LookupPartnerPorts(ctx, &megaport.LookupPartnerPortsRequest{
			Partner:   "GOOGLE",
			Key:       key,
			PortSpeed: 1000,
		})
		if err == nil {
			gcpAvailable++
		}
	}
	t.Logf("GCP pairing keys with available capacity: %d/%d", gcpAvailable, len(creds.GooglePairingKeys))
	if gcpAvailable == 0 {
		t.Error("WARN: no GCP pairing key has available capacity — GCP VXC tests will be skipped")
	}
}

// ── Diagnostics ───────────────────────────────────────────────────────────────

// TestListMVECapacity prints all staging locations with available MVE capacity.
// Never fails. Run manually to discover locations or to refresh the pool file:
//
//	go test -v -run TestListMVECapacity ./internal/provider/
func TestListMVECapacity(t *testing.T) {
	if testing.Short() {
		t.Skip("diagnostic only")
	}
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("no client: %v", err)
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("list failed: %v", err)
	}
	t.Logf("%-6s %-10s %-8s %s", "ID", "MaxCores", "Status", "Name")
	for _, loc := range locations {
		if !loc.HasMVESupport() {
			continue
		}
		cores := loc.GetMVEMaxCpuCores()
		coresStr := "nil"
		if cores != nil {
			coresStr = fmt.Sprintf("%d", *cores)
		}
		t.Logf("%-6d %-10s %-8s %s", loc.ID, coresStr, loc.Status, loc.Name)
	}
	t.Skip("diagnostic complete")
}

// TestListPortCapacity prints all staging locations with port/MCR capacity.
// Never fails. Run manually to refresh testdata/port_test_locations.json:
//
//	go test -v -run TestListPortCapacity ./internal/provider/
func TestListPortCapacity(t *testing.T) {
	if testing.Short() {
		t.Skip("diagnostic only")
	}
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("no client: %v", err)
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("list failed: %v", err)
	}
	t.Logf("%-6s %-8s %s", "ID", "Status", "Name")
	for _, loc := range locations {
		if loc.DiversityZones == nil {
			continue
		}
		if !portLocationHasCapacity(loc, 1) && !mcrLocationHasCapacity(loc, 1) {
			continue
		}
		t.Logf("%-6d %-8s %s", loc.ID, loc.Status, loc.Name)
	}
	t.Skip("diagnostic complete")
}

// cleanupDelete controls whether TestCleanupOrphanedResources deletes resources.
// Pass -cleanup-delete on the go test command line to enable deletion.
var cleanupDelete = flag.Bool("cleanup-delete", false, "delete orphaned test resources in TestCleanupOrphanedResources")

// TestCleanupOrphanedResources lists (and optionally deletes) staging resources
// whose name starts with "tf-acc-test-". VXCs are deleted first (before their
// endpoints). Never fails — always skips at the end.
//
//	# List only:
//	go test -v -run TestCleanupOrphanedResources ./internal/provider/
//
//	# Delete:
//	go test -v -run TestCleanupOrphanedResources -cleanup-delete ./internal/provider/
func TestCleanupOrphanedResources(t *testing.T) {
	if testing.Short() {
		t.Skip("cleanup requires API access")
	}
	const prefix = "tf-acc-test-"
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("no client: %v", err)
	}

	// VXCs first — must be deleted before their A/B-end resources
	vxcs, err := client.VXCService.ListVXCs(ctx, &megaport.ListVXCsRequest{})
	if err != nil {
		t.Logf("WARN: could not list VXCs: %v", err)
	}
	for _, vxc := range vxcs {
		if !strings.HasPrefix(vxc.Name, prefix) {
			continue
		}
		t.Logf("VXC  %s (%s) status=%s", vxc.Name, vxc.UID, vxc.ProvisioningStatus)
		if *cleanupDelete {
			if delErr := client.VXCService.DeleteVXC(ctx, vxc.UID, &megaport.DeleteVXCRequest{DeleteNow: true}); delErr != nil {
				t.Logf("  delete failed: %v", delErr)
			} else {
				t.Logf("  deleted")
			}
		}
	}

	// MVEs
	mves, err := client.MVEService.ListMVEs(ctx, &megaport.ListMVEsRequest{})
	if err != nil {
		t.Logf("WARN: could not list MVEs: %v", err)
	}
	for _, mve := range mves {
		if !strings.HasPrefix(mve.Name, prefix) {
			continue
		}
		t.Logf("MVE  %s (%s) status=%s", mve.Name, mve.UID, mve.ProvisioningStatus)
		if *cleanupDelete {
			if _, delErr := client.MVEService.DeleteMVE(ctx, &megaport.DeleteMVERequest{MVEID: mve.UID}); delErr != nil {
				t.Logf("  delete failed: %v", delErr)
			} else {
				t.Logf("  deleted")
			}
		}
	}

	// MCRs
	mcrs, err := client.MCRService.ListMCRs(ctx, &megaport.ListMCRsRequest{})
	if err != nil {
		t.Logf("WARN: could not list MCRs: %v", err)
	}
	for _, mcr := range mcrs {
		if !strings.HasPrefix(mcr.Name, prefix) {
			continue
		}
		t.Logf("MCR  %s (%s) status=%s", mcr.Name, mcr.UID, mcr.ProvisioningStatus)
		if *cleanupDelete {
			if _, delErr := client.MCRService.DeleteMCR(ctx, &megaport.DeleteMCRRequest{MCRID: mcr.UID, DeleteNow: true}); delErr != nil {
				t.Logf("  delete failed: %v", delErr)
			} else {
				t.Logf("  deleted")
			}
		}
	}

	// Ports (last — must come after VXCs that connect to them)
	ports, err := client.PortService.ListPorts(ctx)
	if err != nil {
		t.Logf("WARN: could not list ports: %v", err)
	}
	for _, port := range ports {
		if !strings.HasPrefix(port.Name, prefix) {
			continue
		}
		t.Logf("Port %s (%s) status=%s", port.Name, port.UID, port.ProvisioningStatus)
		if *cleanupDelete {
			if _, delErr := client.PortService.DeletePort(ctx, &megaport.DeletePortRequest{PortID: port.UID, DeleteNow: true}); delErr != nil {
				t.Logf("  delete failed: %v", delErr)
			} else {
				t.Logf("  deleted")
			}
		}
	}

	t.Skip("cleanup scan complete")
}
