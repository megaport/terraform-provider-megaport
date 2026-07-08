package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
	} {
		t.Run(scenario.description, func(t *testing.T) {
			filtered := runImageFiltersAndSort(images, scenario.filters)
			assert.ElementsMatch(t, filtered, scenario.expectedMVEImages)
		})
	}
}

func TestFromAPIMVEImage(t *testing.T) {
	t.Run("with available sizes", func(t *testing.T) {
		apiImage := &megaport.MVEImage{
			ID:                56,
			Version:           "6.4.15",
			Product:           "FortiGate-VM",
			Vendor:            "Fortinet",
			VendorDescription: "Fortinet FortiGate Virtual Appliance",
			ReleaseImage:      true,
			ProductCode:       "fortigate",
			AvailableSizes:    []string{"MVE 2/8", "MVE 4/16", "MVE 8/32"},
		}

		model := &mveImageDetailsModel{}
		model.fromAPIMVEImage(apiImage)

		assert.Equal(t, int64(56), model.ID.ValueInt64())
		assert.Equal(t, "6.4.15", model.Version.ValueString())
		assert.Equal(t, "FortiGate-VM", model.Product.ValueString())
		assert.Equal(t, "Fortinet", model.Vendor.ValueString())
		assert.Equal(t, "Fortinet FortiGate Virtual Appliance", model.VendorDescription.ValueString())
		assert.True(t, model.ReleaseImage.ValueBool())
		assert.Equal(t, "fortigate", model.ProductCode.ValueString())

		// Verify available sizes
		assert.False(t, model.AvailableSizes.IsNull())
		sizes := make([]string, 0)
		diags := model.AvailableSizes.ElementsAs(context.Background(), &sizes, false)
		assert.Empty(t, diags)
		assert.Equal(t, []string{"MVE 2/8", "MVE 4/16", "MVE 8/32"}, sizes)
	})

	t.Run("without available sizes", func(t *testing.T) {
		apiImage := &megaport.MVEImage{
			ID:                57,
			Version:           "7.0.14",
			Product:           "FortiGate-VM",
			Vendor:            "Fortinet",
			VendorDescription: "",
			ReleaseImage:      false,
			ProductCode:       "fortigate",
			AvailableSizes:    nil, // No sizes available (v3 API compatibility)
		}

		model := &mveImageDetailsModel{}
		model.fromAPIMVEImage(apiImage)

		assert.Equal(t, int64(57), model.ID.ValueInt64())
		assert.Equal(t, "7.0.14", model.Version.ValueString())
		assert.False(t, model.ReleaseImage.ValueBool())
		// AvailableSizes should be null when empty
		assert.True(t, model.AvailableSizes.IsNull())
	})

	t.Run("Palo Alto vendor name normalization", func(t *testing.T) {
		apiImage := &megaport.MVEImage{
			ID:             88,
			Version:        "vION 3102v-6.4.1-b12",
			Product:        "Prisma SD-WAN 310xv",
			Vendor:         "Palo Alto", // API returns with space
			ReleaseImage:   true,
			ProductCode:    "prisma-3108",
			AvailableSizes: []string{"MVE 2/8"},
		}

		model := &mveImageDetailsModel{}
		model.fromAPIMVEImage(apiImage)

		// Should be normalized to PALO_ALTO
		assert.Equal(t, "PALO_ALTO", model.Vendor.ValueString())
	})

	t.Run("empty available sizes slice", func(t *testing.T) {
		apiImage := &megaport.MVEImage{
			ID:             99,
			Version:        "1.0.0",
			Product:        "Test Product",
			Vendor:         "Test Vendor",
			ReleaseImage:   true,
			ProductCode:    "test",
			AvailableSizes: []string{}, // Empty slice
		}

		model := &mveImageDetailsModel{}
		model.fromAPIMVEImage(apiImage)

		// Empty slice should result in null list
		assert.True(t, model.AvailableSizes.IsNull())
	})
}

func TestMVEImageFiltersWithAvailableSizes(t *testing.T) {
	// Test that filters work correctly with images that have AvailableSizes populated
	images := []*megaport.MVEImage{
		{
			ID:             1,
			Version:        "v1",
			Product:        "FortiGate-VM",
			Vendor:         "Fortinet",
			ProductCode:    "fortigate",
			ReleaseImage:   true,
			AvailableSizes: []string{"MVE 2/8", "MVE 4/16"},
		},
		{
			ID:             2,
			Version:        "v2",
			Product:        "C8000",
			Vendor:         "Cisco",
			ProductCode:    "c8000",
			ReleaseImage:   true,
			AvailableSizes: []string{"MVE 2/8", "MVE 4/16", "MVE 8/32", "MVE 12/48"},
		},
	}

	// modelSizes converts a filtered SDK image through fromAPIMVEImage and
	// returns the resulting available_sizes as a []string, so the test
	// exercises the conversion the data source actually performs, not just
	// the filter's slice passthrough.
	modelSizes := func(t *testing.T, image *megaport.MVEImage) []string {
		t.Helper()
		model := &mveImageDetailsModel{}
		model.fromAPIMVEImage(image)
		assert.False(t, model.AvailableSizes.IsNull())
		sizes := make([]string, 0)
		diags := model.AvailableSizes.ElementsAs(context.Background(), &sizes, false)
		assert.Empty(t, diags)
		return sizes
	}

	t.Run("filter by vendor preserves available sizes", func(t *testing.T) {
		filters := []func(*megaport.MVEImage) bool{
			filterMVEImageByVendor("Fortinet"),
		}
		filtered := runImageFiltersAndSort(images, filters)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "Fortinet", filtered[0].Vendor)
		assert.Equal(t, []string{"MVE 2/8", "MVE 4/16"}, modelSizes(t, filtered[0]))
	})

	t.Run("filter by product code preserves available sizes", func(t *testing.T) {
		filters := []func(*megaport.MVEImage) bool{
			filterMVEImageByProductCode("c8000"),
		}
		filtered := runImageFiltersAndSort(images, filters)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "Cisco", filtered[0].Vendor)
		assert.Equal(t, []string{"MVE 2/8", "MVE 4/16", "MVE 8/32", "MVE 12/48"}, modelSizes(t, filtered[0]))
	})
}

func TestMVEImageDetailsAttrs(t *testing.T) {
	// Verify that mveImageDetailsAttrs contains all expected fields including available_sizes
	expectedFields := []string{
		"id",
		"version",
		"product",
		"vendor",
		"vendor_description",
		"release_image",
		"product_code",
		"available_sizes",
	}

	for _, field := range expectedFields {
		t.Run(field, func(t *testing.T) {
			_, exists := mveImageDetailsAttrs[field]
			assert.True(t, exists, "mveImageDetailsAttrs should contain field: %s", field)
		})
	}

	// Verify available_sizes is the correct type
	availableSizesType := mveImageDetailsAttrs["available_sizes"]
	assert.Equal(t, types.ListType{ElemType: types.StringType}, availableSizesType)
}

// TestAccMegaportMVEImagesDataSource_AvailableSizes exercises the read-only
// mve_images data source end-to-end against the API and asserts the v4
// available_sizes field flows through to state. No infrastructure is provisioned.
func TestAccMegaportMVEImagesDataSource_AvailableSizes(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
				data "megaport_mve_images" "aruba" {
					vendor_filter = "Aruba"
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.megaport_mve_images.aruba", "mve_images.0.id"),
					resource.TestCheckResourceAttr("data.megaport_mve_images.aruba", "mve_images.0.vendor", "Aruba"),
					resource.TestCheckResourceAttrSet("data.megaport_mve_images.aruba", "mve_images.0.available_sizes.#"),
					resource.TestCheckResourceAttrSet("data.megaport_mve_images.aruba", "mve_images.0.available_sizes.0"),
				),
			},
		},
	})
}
