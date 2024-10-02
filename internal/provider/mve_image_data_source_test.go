package provider

import (
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
