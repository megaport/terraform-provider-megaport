package provider

import (
	"testing"

	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

func TestFilters(t *testing.T) {
	type filterTestCases struct {
		description   string
		filters       []func(*megaport.PartnerMegaport) bool
		expectedPorts []*megaport.PartnerMegaport
	}

	ports := []*megaport.PartnerMegaport{
		{ConnectType: "ct1", ProductUID: "pid1", ProductName: "pn1", CompanyUID: "cid1", CompanyName: "cn1", DiversityZone: "red", LocationId: 1, VXCPermitted: true},
		{ConnectType: "ct1", ProductUID: "pid2", ProductName: "pn2", CompanyUID: "cid1", CompanyName: "cn1", DiversityZone: "red", LocationId: 1, VXCPermitted: false},
		{ConnectType: "ct1", ProductUID: "pid3", ProductName: "pn3", CompanyUID: "cid1", CompanyName: "cn1", DiversityZone: "red", LocationId: 1, VXCPermitted: false},
		{ConnectType: "ct2", ProductUID: "pid4", ProductName: "pn4", CompanyUID: "cid2", CompanyName: "cn2", DiversityZone: "blue", LocationId: 2, VXCPermitted: true},
	}

	for _, scenario := range []filterTestCases{
		{
			description: "vxcPermittedTrue",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByVXCPermitted(true),
			},
			expectedPorts: []*megaport.PartnerMegaport{
				ports[0],
				ports[3],
			},
		},
		{
			description: "vxcPermittedFalse",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByVXCPermitted(false),
			},
			expectedPorts: []*megaport.PartnerMegaport{
				ports[1],
				ports[2],
			},
		},
		{
			description: "companyName",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByCompanyName("cn1"),
			},
			expectedPorts: []*megaport.PartnerMegaport{
				ports[0],
				ports[1],
				ports[2],
			},
		},
		{
			description: "productName",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByProductName("pn1"),
			},
			expectedPorts: []*megaport.PartnerMegaport{
				ports[0],
			},
		},
		{
			description: "diversityZone",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByDiversityZone("blue"),
			},
			expectedPorts: []*megaport.PartnerMegaport{
				ports[3],
			},
		},
		{
			description: "connectType",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByConnectType("ct2"),
			},
			expectedPorts: []*megaport.PartnerMegaport{
				ports[3],
			},
		},
		{
			description: "location",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByLocationID(1),
			},
			expectedPorts: []*megaport.PartnerMegaport{
				ports[0],
				ports[1],
				ports[2],
			},
		},
		{
			description: "locationDiversityNoResult",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByLocationID(1), filterByDiversityZone("blue"),
			},
			expectedPorts: []*megaport.PartnerMegaport{},
		},
		{
			description: "locationDiversityProductName",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByLocationID(1), filterByDiversityZone("red"), filterByProductName("pn2"),
			},
			expectedPorts: []*megaport.PartnerMegaport{
				ports[1],
			},
		},
		{
			description: "allFilters",
			filters: []func(*megaport.PartnerMegaport) bool{
				filterByLocationID(2),
				filterByDiversityZone("blue"),
				filterByProductName("pn4"),
				filterByCompanyName("cn2"),
				filterByVXCPermitted(true),
				filterByConnectType("ct2"),
			},
			expectedPorts: []*megaport.PartnerMegaport{
				ports[3],
			},
		},
		{
			description:   "noFilters",
			filters:       []func(*megaport.PartnerMegaport) bool{},
			expectedPorts: ports,
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			filtered := runFiltersAndSort(ports, scenario.filters)
			assert.ElementsMatch(t, filtered, scenario.expectedPorts)
		})
	}
}

func TestRank(t *testing.T) {
	ports := []*megaport.PartnerMegaport{
		{ProductName: "p1", Rank: 0},
		{ProductName: "p6", Rank: 4},
		{ProductName: "p3", Rank: 2},
		{ProductName: "p2", Rank: 1},
		{ProductName: "p4", Rank: 3},
		{ProductName: "p5", Rank: 3},
	}

	// should sort by rank then by product name
	sorted := runFiltersAndSort(ports, nil)

	assert.ElementsMatch(t, sorted, []*megaport.PartnerMegaport{
		ports[0],
		ports[3],
		ports[2],
		ports[4],
		ports[5],
		ports[1],
	})
}
