package provider

import (
	"strings"
	"testing"

	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

func TestMVEImageFilters(t *testing.T) {
	type filterTestCases struct {
		description       string
		filters           []func(*megaport.MVEImage) bool
		expectedMVEImages []*megaport.MVEImage
	}

	images := []*megaport.MVEImage{
		{
			ID:           1,
			Version:      "v1",
			Product:      "p1",
			Vendor:       "v1",
			ProductCode:  "pc1",
			ReleaseImage: true,
		},
		{
			ID:           2,
			Version:      "v2",
			Product:      "p2",
			Vendor:       "v2",
			ProductCode:  "pc2",
			ReleaseImage: false,
		},
		{
			ID:           3,
			Version:      "v3",
			Product:      "p3",
			Vendor:       "v3",
			ProductCode:  "pc3",
			ReleaseImage: true,
		},
		{
			ID:           4,
			Version:      "v4",
			Product:      "p4",
			Vendor:       "v4",
			ProductCode:  "pc4",
			ReleaseImage: false,
		},
		{
			ID:           5,
			Version:      "10.1.0",
			Product:      "VM-Series",
			Vendor:       "palo_alto",
			ProductCode:  "pa-vm",
			ReleaseImage: true,
		},
		{
			ID:           6,
			Version:      "10.1.0",
			Product:      "VM-Series",
			Vendor:       "palo alto",
			ProductCode:  "pa-vm",
			ReleaseImage: true,
		},
		{
			ID:           7,
			Version:      "10.2.0",
			Product:      "VM-Series",
			Vendor:       "Palo Alto Networks",
			ProductCode:  "pa-vm",
			ReleaseImage: true,
		},
	}

	for _, scenario := range []filterTestCases{
		{
			description: "releaseImageTrue",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByIsReleaseImage(true),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[0],
				images[2],
				images[4],
				images[5],
				images[6],
			},
		},
		{
			description: "releaseImageFalse",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByIsReleaseImage(false),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[1],
				images[3],
			},
		},
		{
			description: "product",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByProduct("p1"),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[0],
			},
		},
		{
			description: "vendor",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByVendor("v1"),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[0],
			},
		},
		{
			description: "productCode",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByProductCode("pc1"),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[0],
			},
		},
		{
			description: "version",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByVersion("v1"),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[0],
			},
		},
		{
			description: "id",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByID(1),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[0],
			},
		},
		{
			description: "multipleFilters",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByIsReleaseImage(true),
				filterMVEImageByVendor("v1"),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[0],
			},
		},
		// Add specific tests for Palo Alto vendor name variations
		{
			description: "vendor_palo_alto_with_underscore",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByVendor("palo_alto"),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[4],
			},
		},
		{
			description: "vendor_palo_alto_with_space",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByVendor("palo alto"),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[5],
			},
		},
		{
			description: "vendor_palo_alto_networks_full_name",
			filters: []func(*megaport.MVEImage) bool{
				filterMVEImageByVendor("Palo Alto Networks"),
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[6],
			},
		},
		// Test for normalized search that should match all Palo Alto variants
		{
			description: "vendor_palo_alto_normalized_search",
			filters: []func(*megaport.MVEImage) bool{
				func(i *megaport.MVEImage) bool {
					// Custom filter - keep if contains "palo" and "alto" in any format
					if i.Vendor == "" {
						return true
					}
					vendorLower := strings.ToLower(i.Vendor)
					return !(strings.Contains(vendorLower, "palo") &&
						(strings.Contains(vendorLower, "alto") ||
							strings.Contains(vendorLower, "_alto")))
				},
			},
			expectedMVEImages: []*megaport.MVEImage{
				images[4],
				images[5],
				images[6],
			},
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			filtered := runImageFiltersAndSort(images, scenario.filters)
			assert.ElementsMatch(t, filtered, scenario.expectedMVEImages)
		})
	}
}

// TestFilterMVEImageByVendor tests various vendor name formats
func TestFilterMVEImageByVendor(t *testing.T) {
	testCases := []struct {
		name         string
		filterValue  string
		vendor       string
		shouldBeKept bool
	}{
		{"exact match lowercase", "palo alto", "palo alto", true},
		{"exact match with underscore", "palo_alto", "palo_alto", true},
		{"exact match with full name", "Palo Alto Networks", "Palo Alto Networks", true},
		{"case insensitive", "PALO ALTO", "palo alto", true},
		{"case insensitive underscore", "PALO_ALTO", "palo_alto", true},
		{"no match", "palo alto", "cisco", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			img := &megaport.MVEImage{
				Vendor: tc.vendor,
			}
			filter := filterMVEImageByVendor(tc.filterValue)
			// Our filter returns true if it should be REMOVED
			shouldRemove := filter(img)
			assert.Equal(t, !tc.shouldBeKept, shouldRemove,
				"Filter should return %v for vendor '%s' with filter '%s'",
				!tc.shouldBeKept, tc.vendor, tc.filterValue)
		})
	}
}
