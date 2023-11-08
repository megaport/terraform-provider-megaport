// Copyright 2020 Megaport Pty Ltd
//
// Licensed under the Mozilla Public License, Version 2.0 (the
// "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//       https://mozilla.org/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data_megaport

import (
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/megaportgo/types"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportInternet() *schema.Resource {
	return &schema.Resource{
		Read:   dataMegaportInternetRead,
		Schema: schema_megaport.DataMegaportInternetSchema(),
	}
}

func dataMegaportInternetRead(d *schema.ResourceData, m interface{}) error {
	location := m.(*terraform_utility.MegaportClient).Location
	partner := m.(*terraform_utility.MegaportClient).Partner

	targetMetro, ok := d.GetOk("metro")
	if !ok || targetMetro == "" {
		return fmt.Errorf("invalid or missing metro")
	}

	partnerPorts, err := partner.GetAllPartnerMegaports()
	if err != nil {
		return err
	}

	locations, err := location.GetAllLocations()
	if err != nil {
		return err
	}

	// Filter down location list to targets.
	var candidateLocations []types.Location
	for _, loc := range locations {
		if loc.Metro == targetMetro {
			candidateLocations = append(candidateLocations, loc)
		}
	}

	if len(candidateLocations) < 1 {
		return fmt.Errorf("no valid locations found within metro %q", targetMetro)
	}

	var candidatePorts []types.PartnerMegaport
	for _, port := range partnerPorts {
		// Skip any ports that we're not allowed to order to.
		if !port.VXCPermitted {
			continue
		}

		if port.ConnectType == "TRANSIT" && slices.ContainsFunc(candidateLocations, func(l types.Location) bool {
			return l.ID == port.LocationId
		}) {
			// Check diversity zone and skip this port if we don't have a match.
			if dz, ok := d.GetOk("requested_diversity_zone"); ok && dz.(string) != "" {
				if port.DiversityZone != dz.(string) {
					continue
				}
			}

			candidatePorts = append(candidatePorts, port)
		}
	}

	if len(candidatePorts) < 1 {
		return fmt.Errorf("no valid internet ports found in the requested metro and diversity zone")
	}

	// Use the first port from the slice, this way the API can control the preference without a client change.
	d.SetId(candidatePorts[0].ProductUID)

	return nil
}
