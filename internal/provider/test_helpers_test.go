package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	mrand "math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	megaport "github.com/megaport/megaportgo"
	"golang.org/x/crypto/ssh"
)

// ── Acceptance Test Rate Limiter ──────────────────────────────────────────────

// accTestSemaphore limits the number of concurrent acceptance tests that
// provision real infrastructure, preventing staging API overload.
const maxConcurrentAccTests = 20

var accTestSemaphore = make(chan struct{}, maxConcurrentAccTests)

// acquireAccTestSlot blocks until a slot is available in the concurrency pool.
// Returns a release function that must be deferred.
func acquireAccTestSlot(t *testing.T) func() {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test helper requires TF_ACC to be set")
	}
	accTestSemaphore <- struct{}{}
	return func() { <-accTestSemaphore }
}

// ── Types ─────────────────────────────────────────────────────────────────────

type cspCredentials struct {
	AzureServiceKeys          []string `json:"azure_service_keys"`
	AzureServiceKeysWithPeers []string `json:"azure_service_keys_with_peers"`
	OracleVirtualCircuitIDs   []string `json:"oracle_virtual_circuit_ids"`
	GooglePairingKeys         []string `json:"google_pairing_keys"`
}

// ── MVE Location Picker ───────────────────────────────────────────────────────

// mveTestLocationCandidates is a curated list of staging location IDs ordered by
// available CPU capacity (highest first). This ensures parallel MVE tests claim
// the locations most likely to have slots. Refresh from Metabase capacity data.
// Last updated: 2026-04-10.
var mveTestLocationCandidates = []int{
	// Tier 1: 30+ available cores (best bets)
	4,   // Melbourne mel-nxt1 — red: 16+98 cores
	527, // Paris par-ix5 — red: 60 cores
	65,  // Bay Area sjc-tx2 — red: 46 cores
	36,  // Singapore sin-sg1 — red: 30+47 cores
	130, // Frankfurt fra-ix6 — red: 20+64, blue: 6+6 cores
	558, // Tokyo tky-aty — red: 32 cores
	572, // Osaka osk-eq1 — red: 30+27 cores
	47,  // Hong Kong hkg-mgi — red: 30 cores
	256, // London Telehouse North — blue: 30 cores
	5,   // Brisbane bne-nxt1 — blue: 8+36 cores
	131, // Frankfurt fra-eq5 — blue: 18+25 cores
	515, // Paris par-eq2 — blue: 26 cores

	// Tier 2: 15-29 available cores
	23,  // Melbourne mel-mdc — blue: 22+18 cores
	354, // Bay Area sjc-vxc — blue: 22 cores
	89,  // London lon-tc1 — red: 26 cores
	346, // Ashburn ash-rw3 — blue: 20 cores
	68,  // Ashburn ash-cs2 — red: 20 cores
	573, // Calgary cgy-ro1 — blue: 24 cores
	574, // Calgary cgy-es1 — red: 24 cores
	122, // Berlin ber-ipb2 — blue: 24 cores
	98,  // Stockholm sto-ix5 — red: 24 cores
	413, // London lon-vl1 — red: 24 cores
	552, // Miami mia-qt1 — red: 24 cores
	62,  // New York nyc-tx1 — blue: 20 cores
	321, // Denver den-irm — blue: 14+30 cores
	330, // Phoenix phx-io1 — blue: 18+4 cores
	234, // Miami mia-vzn — blue: 18 cores
	85,  // Amsterdam ams-eq1 — blue: 18+13 cores
	50,  // Perth per-nxt1 — blue: 12, red: 14 cores

	// Tier 3: 8-14 available cores (fallback)
	59,  // Los Angeles lax-eq1 — blue: 10, red: 6+22 cores
	90,  // London lon-eq5 — blue: 12+7 cores
	94,  // Dublin dub-tc1 — blue: 14+14 cores
	116, // Atlanta atl-tx1 — blue: 12+19 cores
	93,  // Toronto tor-co2 — red: 10+26 cores
	320, // Denver den-cs1 — blue: 10, red: 10+32 cores
	100, // Las Vegas las-sw7 — blue: 12 cores
	57,  // Seattle sea-eq2 — blue Supermicro: 29 cores
	2,   // Sydney syd-sy1 — blue: 4+12 cores
	37,  // Singapore sin-sg2 — blue: 8 cores
	383, // Brisbane bne-nxt2 — red Supermicro: 30 cores
}

// mveClaimedLocations tracks locations already handed out by findMVETestLocation
// so that parallel tests each get a unique location and don't compete for capacity.
var (
	mveClaimedMu        sync.Mutex
	mveClaimedLocations = map[int]bool{}
)

// mveProbeOpts configures how the MVE location probe validates capacity.
type mveProbeOpts struct {
	vendorConfig  megaport.VendorConfig
	diversityZone string
	vnicCount     int
	// locationFilter, if set, restricts candidates to locations it returns true for.
	locationFilter func(*megaport.LocationV3) bool
}

// findMVETestLocation returns a staging location with confirmed Aruba SMALL MVE
// capacity in the "red" diversity zone. Each call returns a different location.
//
//nolint:unparam // minCPUCores kept for future use when API populates the field
func findMVETestLocation(t *testing.T, minCPUCores int) (id int, name string) {
	return findMVETestLocationWithOpts(t, mveProbeOpts{
		vendorConfig: &megaport.ArubaConfig{
			Vendor:      "aruba",
			ImageID:     MVEArubaImageID,
			ProductSize: "SMALL",
			MVELabel:    "MVE 2/8",
			AccountName: "probe",
			AccountKey:  "probe",
			SystemTag:   "Preconfiguration-aruba-test-1",
		},
		diversityZone: "red",
		vnicCount:     2,
	})
}

// findMVEWithPartnerTestLocation returns a single ACTIVE staging location that
// has confirmed Aruba SMALL MVE capacity AND hosts a VXC-permitted partner port
// of connectType. Provisioning the MVE and the partner-facing VXC end at one
// location keeps them in the same region, which the API enforces for partner
// types like TRANSIT. Picking the MVE and partner locations independently can
// straddle regions under parallel load. Claims the location in the MVE pool.
func findMVEWithPartnerTestLocation(t *testing.T, connectType string) (id int) {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return 0
	}
	partnerPorts, err := client.PartnerService.ListPartnerMegaports(ctx)
	if err != nil {
		t.Skipf("skipping: could not list partner ports: %v", err)
		return 0
	}
	partnerLocs := map[int]bool{}
	for _, pp := range partnerPorts {
		if strings.EqualFold(pp.ConnectType, connectType) && pp.VXCPermitted {
			partnerLocs[pp.LocationId] = true
		}
	}
	if len(partnerLocs) == 0 {
		t.Skipf("skipping: no VXC-permitted %s partner ports found", connectType)
		return 0
	}
	id, _ = findMVETestLocationWithOpts(t, mveProbeOpts{
		vendorConfig: &megaport.ArubaConfig{
			Vendor:      "aruba",
			ImageID:     MVEArubaImageID,
			ProductSize: "SMALL",
			MVELabel:    "MVE 2/8",
			AccountName: "probe",
			AccountKey:  "probe",
			SystemTag:   "Preconfiguration-aruba-test-1",
		},
		diversityZone: "red",
		vnicCount:     2,
		locationFilter: func(loc *megaport.LocationV3) bool {
			return partnerLocs[loc.ID]
		},
	})
	return id
}

// findMVETestLocationHighCapacity returns a staging location with enough capacity
// for multiple simultaneous Aruba SMALL MVEs (e.g., the MVE-to-MVE VXC test that
// creates 4 MVEs at the same site). It validates by probing N times.
func findMVETestLocationHighCapacity(t *testing.T, count int) (id int, name string) {
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
		// Validate N MVEs at once to confirm bulk capacity.
		for i := range count {
			err := client.MVEService.ValidateMVEOrder(ctx, &megaport.BuyMVERequest{
				LocationID: loc.ID,
				Name:       fmt.Sprintf("probe-%d", i),
				Term:       1,
				VendorConfig: &megaport.ArubaConfig{
					Vendor:      "aruba",
					ImageID:     MVEArubaImageID,
					ProductSize: "SMALL",
					MVELabel:    "MVE 2/8",
					AccountName: fmt.Sprintf("probe-%d", i),
					AccountKey:  fmt.Sprintf("probe-%d", i),
					SystemTag:   "Preconfiguration-aruba-test-1",
				},
				Vnics: []megaport.MVENetworkInterface{
					{Description: "Data Plane"},
					{Description: "Management Plane"},
					{Description: "Control Plane"},
				},
			})
			if err != nil {
				return false
			}
		}
		return true
	}

	for _, candidateID := range mveTestLocationCandidates {
		loc, ok := byID[candidateID]
		if !ok {
			continue
		}
		mveClaimedMu.Lock()
		claimed := mveClaimedLocations[candidateID]
		mveClaimedMu.Unlock()
		if claimed {
			continue
		}
		if !probe(loc) {
			continue
		}
		mveClaimedMu.Lock()
		if mveClaimedLocations[candidateID] {
			mveClaimedMu.Unlock()
			continue
		}
		mveClaimedLocations[candidateID] = true
		mveClaimedMu.Unlock()
		t.Cleanup(func() {
			mveClaimedMu.Lock()
			defer mveClaimedMu.Unlock()
			delete(mveClaimedLocations, candidateID)
		})
		t.Logf("findMVETestLocationHighCapacity: using location %d (%s) for %d MVEs", loc.ID, loc.Name, count)
		return loc.ID, loc.Name
	}
	for _, loc := range locations {
		mveClaimedMu.Lock()
		claimed := mveClaimedLocations[loc.ID]
		mveClaimedMu.Unlock()
		if claimed {
			continue
		}
		if !probe(loc) {
			continue
		}
		mveClaimedMu.Lock()
		if mveClaimedLocations[loc.ID] {
			mveClaimedMu.Unlock()
			continue
		}
		locID := loc.ID
		mveClaimedLocations[locID] = true
		mveClaimedMu.Unlock()
		t.Cleanup(func() {
			mveClaimedMu.Lock()
			defer mveClaimedMu.Unlock()
			delete(mveClaimedLocations, locID)
		})
		t.Logf("findMVETestLocationHighCapacity: using location %d (%s) for %d MVEs (sweep)", locID, loc.Name, count)
		return locID, loc.Name
	}
	t.Skipf("skipping: no location with capacity for %d MVEs found", count)
	return 0, ""
}

// findMVETestLocationBlueZone returns a staging location with Aruba SMALL MVE
// capacity in the "blue" diversity zone with 3 vNICs.
func findMVETestLocationBlueZone(t *testing.T) (id int, name string) {
	return findMVETestLocationWithOpts(t, mveProbeOpts{
		vendorConfig: &megaport.ArubaConfig{
			Vendor:      "aruba",
			ImageID:     MVEArubaImageID,
			ProductSize: "SMALL",
			MVELabel:    "MVE 2/8",
			AccountName: "probe",
			AccountKey:  "probe",
			SystemTag:   "Preconfiguration-aruba-test-1",
		},
		diversityZone: "blue",
		vnicCount:     3,
	})
}

// findMVEVersaTestLocation returns a staging location with Versa MVE capacity.
func findMVEVersaTestLocation(t *testing.T) (id int, name string) {
	return findMVETestLocationWithOpts(t, mveProbeOpts{
		vendorConfig: &megaport.VersaConfig{
			Vendor:            "versa",
			ImageID:           20,
			ProductSize:       "SMALL",
			MVELabel:          "MVE 2/8",
			DirectorAddress:   "director1.versa.com",
			ControllerAddress: "controller1.versa.com",
			LocalAuth:         "SDWAN-Branch@Versa.com",
			RemoteAuth:        "Controller-1-staging@Versa.com",
			SerialNumber:      "Megaport-Hub1",
		},
		diversityZone: "red",
		vnicCount:     2,
	})
}

// mveAdminPasswordSafeAlphabet is the character set used to build random admin
// passwords and password hashes for acceptance tests. It is the intersection of
// what the Megaport API accepts for the Cisco FTDv `adminPassword` field
// (anything except `"`, CR, LF) and the sha256crypt salt/encrypted alphabet
// `[a-zA-Z0-9./]` enforced by megalith's HashAdminPasswordValidator.
const mveAdminPasswordSafeAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789./"

// randomFromAlphabet returns a string of length n drawn uniformly from alphabet
// using crypto/rand. It is intended for test-only credentials.
func randomFromAlphabet(t *testing.T, n int, alphabet string) string {
	t.Helper()
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("crypto/rand.Read: %v", err)
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out)
}

// mveTestCiscoAdminPassword returns a freshly-generated plaintext admin password
// suitable for Cisco FTDv acceptance tests. The API requires 9–100 characters
// and disallows `"`, CR, LF. We generate 24 characters from a safe alphabet so
// every test run uses unique credentials on the (temporary) provisioned MVE.
func mveTestCiscoAdminPassword(t *testing.T) string {
	t.Helper()
	return randomFromAlphabet(t, 24, mveAdminPasswordSafeAlphabet)
}

// mveTestPaloAltoAdminPasswordHash returns a freshly-generated sha256crypt-format
// admin password hash suitable for Palo Alto VM-Series acceptance tests. The API
// only checks the structural format `$5$<salt>$<encrypted>` with an 8–16 char
// salt and 43-char encrypted segment over `[a-zA-Z0-9./]`, so we synthesize a
// random hash that satisfies the validator without needing a real sha256crypt
// implementation. The corresponding plaintext is never used — the MVE is
// destroyed at the end of the test before anyone could log in.
func mveTestPaloAltoAdminPasswordHash(t *testing.T) string {
	t.Helper()
	salt := randomFromAlphabet(t, 16, mveAdminPasswordSafeAlphabet)
	encrypted := randomFromAlphabet(t, 43, mveAdminPasswordSafeAlphabet)
	return "$5$" + salt + "$" + encrypted
}

// mveTestSSHKeyOnce caches a freshly-generated RSA 2048 OpenSSH public key for
// the lifetime of the test binary so Palo Alto MVE tests don't need an external
// MEGAPORT_TEST_SSH_PUBLIC_KEY env var. The corresponding private key is
// discarded — tests only validate the order, never log into the MVE.
var (
	mveTestSSHKeyOnce sync.Once
	mveTestSSHKeyVal  string
	mveTestSSHKeyErr  error
)

func mveTestSSHPublicKey(t *testing.T) string {
	t.Helper()
	mveTestSSHKeyOnce.Do(func() {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			mveTestSSHKeyErr = fmt.Errorf("generate RSA key: %w", err)
			return
		}
		pub, err := ssh.NewPublicKey(&key.PublicKey)
		if err != nil {
			mveTestSSHKeyErr = fmt.Errorf("derive ssh public key: %w", err)
			return
		}
		mveTestSSHKeyVal = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
	})
	if mveTestSSHKeyErr != nil {
		t.Fatalf("mveTestSSHPublicKey: %v", mveTestSSHKeyErr)
	}
	return mveTestSSHKeyVal
}

// findMVECiscoTestLocation returns a staging location with confirmed capacity for
// a Cisco Firewall (FTDv) MVE. It dynamically resolves the Cisco Firewall image ID
// so the probe stays valid as staging images change.
// The caller must set MEGAPORT_TEST_SSH_PUBLIC_KEY — tests skip if absent.
func findMVECiscoTestLocation(t *testing.T) (locationID int, imageID int, locationName string) {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return 0, 0, ""
	}

	images, err := client.MVEService.ListMVEImages(ctx)
	if err != nil {
		t.Skipf("skipping: could not list MVE images: %v", err)
		return 0, 0, ""
	}

	var ciscoImageID int
	for _, img := range images {
		if strings.EqualFold(img.Vendor, "Cisco") && strings.Contains(strings.ToLower(img.Product), "firewall") && img.ReleaseImage {
			ciscoImageID = img.ID
			break
		}
	}
	if ciscoImageID == 0 {
		t.Skip("skipping: no released Cisco Firewall MVE image found in staging")
		return 0, 0, ""
	}

	locID, locName := findMVETestLocationWithOpts(t, mveProbeOpts{
		vendorConfig: &megaport.CiscoConfig{
			Vendor:        "cisco",
			ImageID:       ciscoImageID,
			ProductSize:   "MEDIUM",
			MVELabel:      "MVE 4/16",
			ManageLocally: true,
			AdminPassword: mveTestCiscoAdminPassword(t),
		},
		diversityZone: "red",
		vnicCount:     4,
	})
	return locID, ciscoImageID, locName
}

// findMVEPaloAltoTestLocation returns a staging location with confirmed capacity for
// a Palo Alto VM-Series MVE. It dynamically resolves the image ID and uses an
// ephemeral RSA 2048 OpenSSH public key generated by mveTestSSHPublicKey.
func findMVEPaloAltoTestLocation(t *testing.T) (locationID int, imageID int, locationName string) {
	t.Helper()
	sshPublicKey := mveTestSSHPublicKey(t)

	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return 0, 0, ""
	}

	images, err := client.MVEService.ListMVEImages(ctx)
	if err != nil {
		t.Skipf("skipping: could not list MVE images: %v", err)
		return 0, 0, ""
	}

	var paloAltoImageID int
	for _, img := range images {
		if strings.EqualFold(img.Vendor, "Palo Alto") && strings.Contains(strings.ToLower(img.Product), "vm-series") && img.ReleaseImage {
			paloAltoImageID = img.ID
			break
		}
	}
	if paloAltoImageID == 0 {
		t.Skip("skipping: no released Palo Alto VM-Series MVE image found in staging")
		return 0, 0, ""
	}

	locID, locName := findMVETestLocationWithOpts(t, mveProbeOpts{
		vendorConfig: &megaport.PaloAltoConfig{
			Vendor:            "palo_alto",
			ImageID:           paloAltoImageID,
			ProductSize:       "MEDIUM",
			MVELabel:          "MVE 4/16",
			AdminPasswordHash: mveTestPaloAltoAdminPasswordHash(t),
			SSHPublicKey:      sshPublicKey,
		},
		diversityZone: "red",
		vnicCount:     2,
	})
	return locID, paloAltoImageID, locName
}

// findMVETestLocationWithOpts returns a staging location ID with confirmed MVE
// capacity for the given probe options. Each call returns a unique location.
func findMVETestLocationWithOpts(t *testing.T, opts mveProbeOpts) (id int, name string) {
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
		if opts.locationFilter != nil && !opts.locationFilter(loc) {
			return false
		}
		vnics := make([]megaport.MVENetworkInterface, opts.vnicCount)
		for i := range vnics {
			vnics[i] = megaport.MVENetworkInterface{Description: fmt.Sprintf("vNIC %d", i)}
		}
		err := client.MVEService.ValidateMVEOrder(ctx, &megaport.BuyMVERequest{
			LocationID:    loc.ID,
			Name:          "probe",
			Term:          1,
			DiversityZone: opts.diversityZone,
			VendorConfig:  opts.vendorConfig,
			Vnics:         vnics,
		})
		return err == nil
	}

	claim := func(locID int, locName, source string) (int, string) {
		mveClaimedMu.Lock()
		defer mveClaimedMu.Unlock()
		if mveClaimedLocations[locID] {
			return 0, "" // already taken
		}
		mveClaimedLocations[locID] = true
		t.Cleanup(func() {
			mveClaimedMu.Lock()
			defer mveClaimedMu.Unlock()
			delete(mveClaimedLocations, locID)
		})
		t.Logf("findMVETestLocation: using location %d (%s) [%s]", locID, locName, source)
		return locID, locName
	}

	// Fast path: curated candidates
	for _, candidateID := range mveTestLocationCandidates {
		if loc, ok := byID[candidateID]; ok && probe(loc) {
			if id, name := claim(loc.ID, loc.Name, "curated"); id != 0 {
				return id, name
			}
		}
	}

	// Slow path: full sweep
	for _, loc := range locations {
		if probe(loc) {
			if id, name := claim(loc.ID, loc.Name, "sweep"); id != 0 {
				return id, name
			}
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
var (
	portClaimedMu        sync.Mutex
	portClaimedLocations = map[int]bool{}
)

//nolint:unparam // name is used in log messages and may be used by future callers
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
	portClaimedMu.Lock()
	defer portClaimedMu.Unlock()
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") && portLocationHasCapacity(loc, speedMbps) && !portClaimedLocations[loc.ID] {
			locID := loc.ID
			portClaimedLocations[locID] = true
			t.Cleanup(func() {
				portClaimedMu.Lock()
				defer portClaimedMu.Unlock()
				delete(portClaimedLocations, locID)
			})
			t.Logf("findPortTestLocation: using location %d (%s)", locID, loc.Name)
			return locID, loc.Name
		}
	}
	t.Skipf("skipping: no unclaimed ACTIVE location with %d Mbps Megaport port capacity", speedMbps)
	return 0, ""
}

// findMCRTestLocation returns a staging location ID that supports MCR at the
// given speed (Mbps). Calls t.Skip if none found.
//
//nolint:unparam // speedMbps is intentionally parameterized for different MCR speed requirements
var (
	mcrClaimedMu        sync.Mutex
	mcrClaimedLocations = map[int]bool{}
)

//nolint:unparam // speedMbps is intentionally parameterized and used by callers with different requested MCR speeds
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
	mcrClaimedMu.Lock()
	defer mcrClaimedMu.Unlock()
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") && mcrLocationHasCapacity(loc, speedMbps) && !mcrClaimedLocations[loc.ID] {
			locID := loc.ID
			mcrClaimedLocations[locID] = true
			t.Cleanup(func() {
				mcrClaimedMu.Lock()
				defer mcrClaimedMu.Unlock()
				delete(mcrClaimedLocations, locID)
			})
			t.Logf("findMCRTestLocation: using location %d (%s)", locID, loc.Name)
			return locID, loc.Name
		}
	}
	t.Skipf("skipping: no unclaimed ACTIVE location with %d Mbps MCR capacity", speedMbps)
	return 0, ""
}

// findMCRWithPartnerTestLocation returns a single ACTIVE staging location that
// supports an MCR at mcrSpeedMbps AND hosts a VXC-permitted partner port of
// connectType. Provisioning both VXC ends at one location keeps them in the
// same region, which the API enforces for partner types like TRANSIT
// ("LongHaul Transit VXC must be in the same region"). Picking the MCR and
// partner locations independently can straddle regions under parallel load.
// Claims the location in the MCR pool.
func findMCRWithPartnerTestLocation(t *testing.T, mcrSpeedMbps int, connectType string) (id int) {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return 0
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("skipping: could not list locations: %v", err)
		return 0
	}
	partnerPorts, err := client.PartnerService.ListPartnerMegaports(ctx)
	if err != nil {
		t.Skipf("skipping: could not list partner ports: %v", err)
		return 0
	}
	partnerLocs := map[int]bool{}
	for _, pp := range partnerPorts {
		if strings.EqualFold(pp.ConnectType, connectType) && pp.VXCPermitted {
			partnerLocs[pp.LocationId] = true
		}
	}
	mcrClaimedMu.Lock()
	defer mcrClaimedMu.Unlock()
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") && mcrLocationHasCapacity(loc, mcrSpeedMbps) && partnerLocs[loc.ID] && !mcrClaimedLocations[loc.ID] {
			locID := loc.ID
			mcrClaimedLocations[locID] = true
			t.Cleanup(func() {
				mcrClaimedMu.Lock()
				defer mcrClaimedMu.Unlock()
				delete(mcrClaimedLocations, locID)
			})
			t.Logf("findMCRWithPartnerTestLocation(%s): using location %d (%s)", connectType, locID, loc.Name)
			return locID
		}
	}
	t.Skipf("skipping: no unclaimed ACTIVE location with %d Mbps MCR capacity and %s partner ports", mcrSpeedMbps, connectType)
	return 0
}

// findNATGatewayTestLocation returns a staging location ID that supports NAT
// Gateway at the given speed (Mbps) in the "red" diversity zone. Calls t.Skip
// if none found.
//
// Every NAT Gateway acceptance test pins diversity_zone = "red" in its HCL,
// so this helper must verify the red zone specifically. The aggregate
// LocationV3.SupportsNATGatewaySpeed returns true if either zone advertises
// the speed, which can hand back a location whose red zone lacks it — the
// networkdesign/validate endpoint then rejects the order with
// "NAT Gateway speed N is not available in location M".
var (
	natGatewayClaimedMu        sync.Mutex
	natGatewayClaimedLocations = map[int]bool{}
)

func locationRedSupportsNATGatewaySpeed(loc *megaport.LocationV3, speedMbps int) bool {
	if loc == nil || loc.DiversityZones == nil || loc.DiversityZones.Red == nil {
		return false
	}
	for _, s := range loc.DiversityZones.Red.NATGatewaySpeedMbps {
		if s == speedMbps {
			return true
		}
	}
	return false
}

func findNATGatewayTestLocation(t *testing.T, speedMbps int) (id int, name string) { //nolint:unparam // name return is available for callers that want it
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
	natGatewayClaimedMu.Lock()
	defer natGatewayClaimedMu.Unlock()
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") && locationRedSupportsNATGatewaySpeed(loc, speedMbps) && !natGatewayClaimedLocations[loc.ID] {
			locID := loc.ID
			natGatewayClaimedLocations[locID] = true
			t.Cleanup(func() {
				natGatewayClaimedMu.Lock()
				defer natGatewayClaimedMu.Unlock()
				delete(natGatewayClaimedLocations, locID)
			})
			t.Logf("findNATGatewayTestLocation: using location %d (%s) for speed %d (red zone)", locID, loc.Name, speedMbps)
			return locID, loc.Name
		}
	}
	t.Skipf("skipping: no unclaimed ACTIVE location with %d Mbps NAT Gateway capacity in red zone", speedMbps)
	return 0, ""
}

// findNATGatewayAndPortTestLocation returns a single staging location ID that
// supports both NAT Gateway at the given speed and Megaport ports at 1000 Mbps.
// Used for tests that need to provision a NAT Gateway and a Port at the same
// site (e.g. VXC A-End = NAT Gateway, B-End = Port).
func findNATGatewayAndPortTestLocation(t *testing.T, natSpeedMbps int) (id int, name string) { //nolint:unparam // name return is available for callers that want it
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
	natGatewayClaimedMu.Lock()
	portClaimedMu.Lock()
	defer natGatewayClaimedMu.Unlock()
	defer portClaimedMu.Unlock()
	for _, loc := range locations {
		if !strings.EqualFold(loc.Status, "active") {
			continue
		}
		if !locationRedSupportsNATGatewaySpeed(loc, natSpeedMbps) || !portLocationHasCapacity(loc, 1000) {
			continue
		}
		if natGatewayClaimedLocations[loc.ID] || portClaimedLocations[loc.ID] {
			continue
		}
		locID := loc.ID
		natGatewayClaimedLocations[locID] = true
		portClaimedLocations[locID] = true
		t.Cleanup(func() {
			natGatewayClaimedMu.Lock()
			portClaimedMu.Lock()
			defer natGatewayClaimedMu.Unlock()
			defer portClaimedMu.Unlock()
			delete(natGatewayClaimedLocations, locID)
			delete(portClaimedLocations, locID)
		})
		t.Logf("findNATGatewayAndPortTestLocation: using location %d (%s) for NAT speed %d", locID, loc.Name, natSpeedMbps)
		return locID, loc.Name
	}
	t.Skipf("skipping: no unclaimed ACTIVE location with %d Mbps NAT Gateway + 1000 Mbps port capacity", natSpeedMbps)
	return 0, ""
}

// findVXCPortTestLocations returns count unique staging location IDs that
// support Megaport ports at 1000 Mbps. Uses the same portClaimedLocations
// mechanism as findPortTestLocation so parallel tests don't collide.
// Calls t.Skip if not enough locations are found.
//
// findAnyActiveLocationID returns the ID of any active staging location.
// It does NOT claim the location, so use this only for data-source-only tests
// that don't provision real resources.
func findAnyActiveLocationID(t *testing.T) int {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return 0
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("skipping: could not list locations: %v", err)
		return 0
	}
	for _, loc := range locations {
		if strings.EqualFold(loc.Status, "active") {
			return loc.ID
		}
	}
	t.Skip("skipping: no active locations found")
	return 0
}

//nolint:unparam // count is 1 today but callers may vary it
func findVXCPortTestLocations(t *testing.T, count int) []int {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return nil
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("skipping: could not list locations: %v", err)
		return nil
	}
	portClaimedMu.Lock()
	defer portClaimedMu.Unlock()
	var ids []int
	for _, loc := range locations {
		if len(ids) >= count {
			break
		}
		if strings.EqualFold(loc.Status, "active") && portLocationHasCapacity(loc, 1000) && !portClaimedLocations[loc.ID] {
			portClaimedLocations[loc.ID] = true
			locID := loc.ID
			t.Cleanup(func() {
				portClaimedMu.Lock()
				defer portClaimedMu.Unlock()
				delete(portClaimedLocations, locID)
			})
			t.Logf("findVXCPortTestLocations: claimed location %d (%s)", loc.ID, loc.Name)
			ids = append(ids, loc.ID)
		}
	}
	if len(ids) < count {
		t.Skipf("skipping: found only %d of %d unclaimed ACTIVE locations with 1000 Mbps port capacity", len(ids), count)
		return nil
	}
	return ids
}

// findVXCPortAndMCRTestLocations returns count unique staging location IDs that
// support both Megaport ports at 1000 Mbps and MCRs at mcrSpeedMbps. Use this
// for tests that create both a port and an MCR at the same location.
//
//nolint:unparam // count is 1 today but callers may vary it
func findVXCPortAndMCRTestLocations(t *testing.T, count int, mcrSpeedMbps int) []int {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return nil
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("skipping: could not list locations: %v", err)
		return nil
	}
	var ids []int
	for _, loc := range locations {
		if len(ids) >= count {
			break
		}
		if !strings.EqualFold(loc.Status, "active") || !portLocationHasCapacity(loc, 1000) || !mcrLocationHasCapacity(loc, mcrSpeedMbps) {
			continue
		}
		portClaimedMu.Lock()
		mcrClaimedMu.Lock()
		if !portClaimedLocations[loc.ID] && !mcrClaimedLocations[loc.ID] {
			portClaimedLocations[loc.ID] = true
			mcrClaimedLocations[loc.ID] = true
			locID := loc.ID
			t.Cleanup(func() {
				portClaimedMu.Lock()
				mcrClaimedMu.Lock()
				delete(portClaimedLocations, locID)
				delete(mcrClaimedLocations, locID)
				mcrClaimedMu.Unlock()
				portClaimedMu.Unlock()
			})
			t.Logf("findVXCPortAndMCRTestLocations: claimed location %d (%s)", loc.ID, loc.Name)
			ids = append(ids, loc.ID)
		}
		mcrClaimedMu.Unlock()
		portClaimedMu.Unlock()
	}
	if len(ids) < count {
		t.Skipf("skipping: found only %d of %d unclaimed ACTIVE locations with 1000 Mbps port + %d Mbps MCR capacity", len(ids), count, mcrSpeedMbps)
		return nil
	}
	return ids
}

// findVXCPortTestLocationsWithPartner is like findVXCPortTestLocations but also
// requires that the returned locations have at least one partner port of the
// given connect type (e.g. "AWS", "TRANSIT"). Use this for tests whose HCL
// includes a megaport_partner data source filtered by location.
func findVXCPortTestLocationsWithPartner(t *testing.T, count int, connectType string) []int { //nolint:unparam // count is parameterized for API consistency with other find* helpers
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return nil
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("skipping: could not list locations: %v", err)
		return nil
	}
	partnerPorts, err := client.PartnerService.ListPartnerMegaports(ctx)
	if err != nil {
		t.Skipf("skipping: could not list partner ports: %v", err)
		return nil
	}
	partnerLocs := map[int]bool{}
	for _, pp := range partnerPorts {
		if strings.EqualFold(pp.ConnectType, connectType) && pp.VXCPermitted {
			partnerLocs[pp.LocationId] = true
		}
	}
	portClaimedMu.Lock()
	defer portClaimedMu.Unlock()
	var ids []int
	for _, loc := range locations {
		if len(ids) >= count {
			break
		}
		if strings.EqualFold(loc.Status, "active") && portLocationHasCapacity(loc, 1000) && partnerLocs[loc.ID] && !portClaimedLocations[loc.ID] {
			portClaimedLocations[loc.ID] = true
			locID := loc.ID
			t.Cleanup(func() {
				portClaimedMu.Lock()
				defer portClaimedMu.Unlock()
				delete(portClaimedLocations, locID)
			})
			t.Logf("findVXCPortTestLocationsWithPartner(%s): claimed location %d (%s)", connectType, loc.ID, loc.Name)
			ids = append(ids, loc.ID)
		}
	}
	if len(ids) < count {
		t.Skipf("skipping: found only %d of %d unclaimed ACTIVE locations with 1000 Mbps port capacity and %s partner ports", len(ids), count, connectType)
		return nil
	}
	return ids
}

// findVXCPortTestLocationsWithPartners is like findVXCPortTestLocationsWithPartner
// but requires ALL of the given connect types at each location (e.g., "AWS" AND
// "TRANSIT"). Use this for tests that need multiple partner types at the same site.
func findVXCPortTestLocationsWithPartners(t *testing.T, count int, connectTypes ...string) []int {
	t.Helper()
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return nil
	}
	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("skipping: could not list locations: %v", err)
		return nil
	}
	partnerPorts, err := client.PartnerService.ListPartnerMegaports(ctx)
	if err != nil {
		t.Skipf("skipping: could not list partner ports: %v", err)
		return nil
	}
	// Build a set of connect types present at each location.
	locTypes := map[int]map[string]bool{}
	for _, pp := range partnerPorts {
		if !pp.VXCPermitted {
			continue
		}
		if locTypes[pp.LocationId] == nil {
			locTypes[pp.LocationId] = map[string]bool{}
		}
		locTypes[pp.LocationId][strings.ToUpper(pp.ConnectType)] = true
	}
	hasAll := func(locID int) bool {
		m := locTypes[locID]
		for _, ct := range connectTypes {
			if !m[strings.ToUpper(ct)] {
				return false
			}
		}
		return true
	}
	portClaimedMu.Lock()
	defer portClaimedMu.Unlock()
	var ids []int
	for _, loc := range locations {
		if len(ids) >= count {
			break
		}
		if strings.EqualFold(loc.Status, "active") && portLocationHasCapacity(loc, 1000) && hasAll(loc.ID) && !portClaimedLocations[loc.ID] {
			portClaimedLocations[loc.ID] = true
			locID := loc.ID
			t.Cleanup(func() {
				portClaimedMu.Lock()
				defer portClaimedMu.Unlock()
				delete(portClaimedLocations, locID)
			})
			t.Logf("findVXCPortTestLocationsWithPartners(%v): claimed location %d (%s)", connectTypes, loc.ID, loc.Name)
			ids = append(ids, loc.ID)
		}
	}
	if len(ids) < count {
		t.Skipf("skipping: found only %d of %d unclaimed ACTIVE locations with 1000 Mbps port capacity and %v partner ports", len(ids), count, connectTypes)
		return nil
	}
	return ids
}

// portLocationHasCapacity returns true when at least one diversity zone at loc
// lists speedMbps in MegaportSpeedMbps.
func portLocationHasCapacity(loc *megaport.LocationV3, speedMbps int) bool {
	if loc.DiversityZones == nil {
		return false
	}
	check := func(zone *megaport.LocationV3DiversityZone) bool {
		if zone == nil {
			return false
		}
		for _, s := range zone.MegaportSpeedMbps {
			if s == speedMbps {
				return true
			}
		}
		return false
	}
	return check(loc.DiversityZones.Red) || check(loc.DiversityZones.Blue)
}

// mcrLocationHasCapacity returns true when at least one diversity zone at loc
// lists speedMbps in McrSpeedMbps.
func mcrLocationHasCapacity(loc *megaport.LocationV3, speedMbps int) bool {
	if loc.DiversityZones == nil {
		return false
	}
	check := func(zone *megaport.LocationV3DiversityZone) bool {
		if zone == nil {
			return false
		}
		for _, s := range zone.McrSpeedMbps {
			if s == speedMbps {
				return true
			}
		}
		return false
	}
	return check(loc.DiversityZones.Red) || check(loc.DiversityZones.Blue)
}

// ── CSP Credential Pickers ────────────────────────────────────────────────────

// cspPickResult holds a validated CSP key along with the partner port UID and
// location that the key resolves to. Tests should use LocationID for their MCR
// to ensure the CSP interconnect is reachable.
type cspPickResult struct {
	Key            string
	PartnerPortUID string
	LocationID     int
}

// pickAzureServiceKey returns the first Azure service key from the pool that
// has available VXC capacity. It resolves the partner port and its location so
// the caller can place an MCR at a compatible site. Calls t.Skip if none found.
func pickAzureServiceKey(t *testing.T) cspPickResult {
	t.Helper()
	return pickCSPKey(t, "AZURE", "azure")
}

// pickGCPPairingKey returns the first GCP pairing key from the pool that has
// available VXC capacity. Calls t.Skip if none found.
func pickGCPPairingKey(t *testing.T) cspPickResult {
	t.Helper()
	return pickCSPKey(t, "GOOGLE", "google")
}

// cspClaimedKeys tracks CSP keys and partner port UIDs already handed out so
// parallel tests don't reuse the same key or hit the same Azure port (different
// keys can map to the same ExpressRoute circuit, causing VLAN conflicts).
var (
	cspClaimedMu    sync.Mutex
	cspClaimedKeys  = map[string]bool{}
	cspClaimedPorts = map[string]bool{}
)

// pickCSPKey is the shared implementation for CSP key pickers. It validates
// each key via LookupPartnerPorts, then resolves the partner port's location
// via ListPartnerMegaports so the test can place its MCR at a compatible site.
// Each key is claimed exclusively so parallel tests get unique keys.
func pickCSPKey(t *testing.T, partner, connectType string) cspPickResult {
	t.Helper()
	creds, err := loadCSPCredentials()
	if err != nil {
		t.Skipf("skipping: %v", err)
		return cspPickResult{}
	}

	var keys []string
	switch partner {
	case "AZURE":
		keys = creds.AzureServiceKeys
	case "GOOGLE":
		keys = creds.GooglePairingKeys
	}
	if len(keys) == 0 {
		t.Skipf("skipping: no %s keys in testdata/csp_credentials.json", partner)
		return cspPickResult{}
	}

	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("skipping: could not get test client: %v", err)
		return cspPickResult{}
	}

	// Build a location lookup from partner ports so we can resolve port UID → location.
	partnerPorts, err := client.PartnerService.ListPartnerMegaports(ctx)
	if err != nil {
		t.Skipf("skipping: could not list partner ports: %v", err)
		return cspPickResult{}
	}
	portLocation := make(map[string]int, len(partnerPorts))
	for _, pp := range partnerPorts {
		if strings.EqualFold(pp.ConnectType, connectType) {
			portLocation[pp.ProductUID] = pp.LocationId
		}
	}

	//nolint:gosec // weak random is fine for test key shuffling
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	shuffled := make([]string, len(keys))
	copy(shuffled, keys)
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	mask := func(s string) string {
		if len(s) <= 4 {
			return "***"
		}
		return "..." + s[len(s)-4:]
	}

	for _, key := range shuffled {
		masked := mask(key)
		cspClaimedMu.Lock()
		if cspClaimedKeys[key] {
			cspClaimedMu.Unlock()
			t.Logf("pick%sKey: key %s already claimed, skipping", partner, masked)
			continue
		}
		cspClaimedMu.Unlock()

		resp, lookupErr := client.VXCService.LookupPartnerPorts(ctx, &megaport.LookupPartnerPortsRequest{
			Partner:   partner,
			Key:       key,
			PortSpeed: 1000,
		})
		if lookupErr != nil {
			t.Logf("pick%sKey: key %s unavailable: %v", partner, masked, lookupErr)
			continue
		}
		locID := portLocation[resp.ProductUID]
		if locID == 0 {
			t.Logf("pick%sKey: key %s resolved but location unknown, skipping", partner, masked)
			continue
		}

		cspClaimedMu.Lock()
		if cspClaimedKeys[key] || cspClaimedPorts[resp.ProductUID] {
			cspClaimedMu.Unlock()
			t.Logf("pick%sKey: key %s or port already claimed, skipping", partner, masked)
			continue
		}
		cspClaimedKeys[key] = true
		cspClaimedPorts[resp.ProductUID] = true
		claimedKey := key
		claimedPort := resp.ProductUID
		cspClaimedMu.Unlock()

		t.Cleanup(func() {
			cspClaimedMu.Lock()
			delete(cspClaimedKeys, claimedKey)
			delete(cspClaimedPorts, claimedPort)
			cspClaimedMu.Unlock()
		})

		t.Logf("pick%sKey: using key %s (location %d)", partner, masked, locID)
		return cspPickResult{Key: key, PartnerPortUID: resp.ProductUID, LocationID: locID}
	}

	t.Skipf("skipping: no %s key with available capacity found", partner)
	return cspPickResult{}
}

// oracleClaimedMu and oracleClaimedIDs ensure each parallel test gets a unique
// Oracle virtual circuit ID from the pool (they're fake keys matching a regex,
// but reusing the same one in concurrent tests causes "already in use" errors).
var (
	oracleClaimedMu  sync.Mutex
	oracleClaimedIDs = map[string]bool{}
)

// pickOracleVirtualCircuitID returns a unique Oracle virtual circuit ID from the
// pool that is not already attached to an existing VXC. Each candidate is probed
// via LookupPartnerPorts — if the VCID is already in use (orphaned from a prior
// test run), it is skipped. Calls t.Skip if no usable VCID is found.
func pickOracleVirtualCircuitID(t *testing.T) string {
	t.Helper()
	creds, err := loadCSPCredentials()
	if err != nil {
		t.Skipf("skipping: %v", err)
		return ""
	}
	if len(creds.OracleVirtualCircuitIDs) == 0 {
		t.Skip("skipping: no Oracle virtual circuit IDs in testdata/csp_credentials.json")
		return ""
	}

	ctx := context.Background()
	client, clientErr := getTestClient()
	if clientErr != nil {
		t.Skipf("skipping: could not get test client: %v", clientErr)
		return ""
	}

	oracleClaimedMu.Lock()
	defer oracleClaimedMu.Unlock()
	for _, id := range creds.OracleVirtualCircuitIDs {
		if oracleClaimedIDs[id] {
			continue
		}
		// Probe the API to check the VCID is not already attached to a live VXC.
		_, lookupErr := client.VXCService.LookupPartnerPorts(ctx, &megaport.LookupPartnerPortsRequest{
			Partner:   "ORACLE",
			Key:       id,
			PortSpeed: 1000,
		})
		if lookupErr != nil {
			t.Logf("pickOracleVirtualCircuitID: skipping ...%s (lookup failed: %v)", id[max(0, len(id)-4):], lookupErr)
			continue
		}
		oracleClaimedIDs[id] = true
		claimedID := id
		t.Cleanup(func() {
			oracleClaimedMu.Lock()
			delete(oracleClaimedIDs, claimedID)
			oracleClaimedMu.Unlock()
		})
		t.Logf("pickOracleVirtualCircuitID: using ...%s", id[max(0, len(id)-4):])
		return id
	}
	t.Skip("skipping: no Oracle virtual circuit ID available (all claimed or in use on API)")
	return ""
}

func loadCSPCredentials() (cspCredentials, error) {
	// Prefer env var so CI can inject credentials from secrets.
	if raw := os.Getenv("CSP_CREDENTIALS_JSON"); raw != "" {
		var creds cspCredentials
		if err := json.Unmarshal([]byte(raw), &creds); err != nil {
			return cspCredentials{}, fmt.Errorf("CSP_CREDENTIALS_JSON: %w", err)
		}
		return creds, nil
	}
	// Fall back to local file for developer convenience.
	data, err := os.ReadFile("testdata/csp_credentials.json")
	if errors.Is(err, os.ErrNotExist) {
		return cspCredentials{}, nil // file missing is not an error — tests will skip on empty pools
	}
	if err != nil {
		return cspCredentials{}, fmt.Errorf("testdata/csp_credentials.json: %w", err)
	}
	var creds cspCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return cspCredentials{}, fmt.Errorf("testdata/csp_credentials.json: %w", err)
	}
	return creds, nil
}

// ── Staging Health Check ──────────────────────────────────────────────────────

// TestStagingHealthCheck verifies staging environment preconditions.
// Run before a full acceptance suite to catch problems early:
//
//	go test -v -run TestStagingHealthCheck ./internal/provider/
func TestStagingHealthCheck(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("health check requires TF_ACC")
	}
	ctx := context.Background()
	client, err := getTestClient()
	if err != nil {
		t.Skipf("staging API unreachable: %v", err)
	}

	locations, err := client.LocationService.ListLocationsV3(ctx)
	if err != nil {
		t.Skipf("could not list locations (company may be deactivated): %v", err)
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
		t.Log("WARN: no MVE capacity available — MVE tests will be skipped")
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
		t.Log("WARN: no 1G port capacity available — port tests will be skipped")
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
		t.Log("WARN: no 2.5G MCR capacity available — MCR tests will be skipped")
	}

	// Partner ports
	ports, err := client.PartnerService.ListPartnerMegaports(ctx)
	if err != nil {
		t.Logf("WARN: could not list partner ports: %v", err)
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
				t.Logf("WARN: no %s partner port found — %s VXC tests will be skipped", partner, partner)
			}
		}
	}

	// CSP credentials pool — probe each key for live capacity
	creds, credErr := loadCSPCredentials()
	if credErr != nil {
		t.Logf("WARN: could not load CSP credentials: %v", credErr)
	}
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
		t.Log("WARN: no Azure service key has available capacity — Azure VXC tests will be skipped")
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
		t.Log("WARN: no GCP pairing key has available capacity — GCP VXC tests will be skipped")
	}
}

// ── Diagnostics ───────────────────────────────────────────────────────────────

// TestListMVECapacity prints all staging locations with available MVE capacity.
// Never fails. Run manually to discover locations or to refresh the pool file:
//
//	go test -v -run TestListMVECapacity ./internal/provider/
func TestListMVECapacity(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("diagnostic requires TF_ACC")
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
	var total, withCores int
	for _, loc := range locations {
		if !loc.HasMVESupport() {
			continue
		}
		total++
		if cores := loc.GetMVEMaxCpuCores(); cores != nil {
			withCores++
		}
	}
	t.Logf("MVE-capable locations: %d (%d with core counts populated)", total, withCores)
	t.Skip("diagnostic complete")
}

// TestListPortCapacity prints all staging locations with port/MCR capacity.
// Never fails. Run manually to refresh testdata/port_test_locations.json:
//
//	go test -v -run TestListPortCapacity ./internal/provider/
func TestListPortCapacity(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("diagnostic requires TF_ACC")
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
	var portCount, mcrCount, bothCount int
	for _, loc := range locations {
		if loc.DiversityZones == nil {
			continue
		}
		hasPort := portLocationHasCapacity(loc, 1000)
		hasMCR := mcrLocationHasCapacity(loc, 2500)
		if hasPort {
			portCount++
		}
		if hasMCR {
			mcrCount++
		}
		if hasPort && hasMCR {
			bothCount++
		}
	}
	t.Logf("Locations with 1000 Mbps port: %d, 2500 Mbps MCR: %d, both: %d", portCount, mcrCount, bothCount)
	t.Skip("diagnostic complete")
}

// cleanupDelete controls whether TestCleanupOrphanedResources deletes resources.
// Pass -cleanup-delete on the go test command line to enable deletion.
var cleanupDelete = flag.Bool("cleanup-delete", false, "delete orphaned test resources in TestCleanupOrphanedResources")

// TestCleanupOrphanedResources lists (and with -cleanup-delete, deletes)
// staging resources whose name starts with "tf-acc-test-". It shares its
// per-type logic with the sweepers in sweep_test.go, so it is a dry-run-first
// front end to the same cleanup. Never fails; always skips at the end.
//
//	# List only:
//	go test -v -run TestCleanupOrphanedResources ./internal/provider/
//
//	# Delete:
//	go test -v -run TestCleanupOrphanedResources -cleanup-delete ./internal/provider/
func TestCleanupOrphanedResources(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("cleanup requires TF_ACC")
	}
	client, err := getTestClient()
	if err != nil {
		t.Skipf("no client: %v", err)
	}
	ctx := context.Background()
	del := *cleanupDelete

	// Same order the sweepers enforce: VXCs before their endpoints, ports last.
	cleaners := []struct {
		label string
		clean resourceCleaner
	}{
		{"VXC", cleanupOrphanedVXCs},
		{"MVE", cleanupOrphanedMVEs},
		{"MCR", cleanupOrphanedMCRs},
		{"IX", cleanupOrphanedIXs},
		{"Port", cleanupOrphanedPorts},
	}
	summary := make([]string, len(cleaners))
	for i, c := range cleaners {
		n, cleanErr := c.clean(ctx, client, del)
		if cleanErr != nil {
			t.Logf("WARN: %v", cleanErr)
		}
		summary[i] = fmt.Sprintf("%d %s", n, c.label)
	}

	t.Logf("orphaned test resources (live): %s", strings.Join(summary, ", "))
	t.Skip("cleanup scan complete")
}

// testPromoCode returns the promo code string to use in promo_code acceptance
// tests. Set MEGAPORT_TEST_PROMO_CODE to a live staging code to verify the
// code is persisted on the billing record (j_billable.promocode_id). When
// unset, tests use a placeholder — the staging API accepts unknown codes
// silently, so tests still exercise the provider contract but skip backend
// verification. Staging promo codes are short-lived (daily DB refresh), so
// hardcoding a real code would cause the tests to silently stop verifying
// anything real.
func testPromoCode() string {
	if v := os.Getenv("MEGAPORT_TEST_PROMO_CODE"); v != "" {
		return v
	}
	return "tf-acc-test-promo-initial"
}
