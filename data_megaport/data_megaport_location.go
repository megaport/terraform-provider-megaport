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
	"errors"
	"github.com/megaport/megaportgo/types"
	"strconv"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/megaportgo/location"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)



func MegaportLocation() *schema.Resource {
	return &schema.Resource{
		Read:   dataMegaportLocationRead,
		Schema: schema_megaport.DataLocationSchema(),
	}
}

func dataMegaportLocationRead(d *schema.ResourceData, m interface{}) error {
	var locations []types.Location
	var foundLocation *types.Location = nil

	if v, ok := d.GetOk("name"); ok {
		exactLocation, exactErr := location.GetLocationByName(v.(string))

		if exactErr == nil {
			foundLocation = &exactLocation
		} else if d.Get("match_exact").(bool) {
			return errors.New(NoLocationsFoundError)
		} else {
			fuzzyLocations, fuzzyErr := location.GetLocationByNameFuzzy(v.(string))

			if fuzzyErr == nil {
				if len(fuzzyLocations) == 1 {
					result := fuzzyLocations[0]
					foundLocation = &result
				} else if len(fuzzyLocations) > 1 {
					locations = fuzzyLocations
				}
			}

			if len(locations) == 0 && foundLocation == nil {
				allLocations, allErr := location.GetAllLocations()

				if allErr != nil {
					return allErr
				}

				locations = allLocations
			}
		}
	}

	if foundLocation == nil {
		if v, ok := d.GetOk("market_code"); ok {
			location.FilterLocationsByMarketCode(v.(string), &locations)
		}

		if v, ok := d.GetOk("has_mcr"); ok {
			location.FilterLocationsByMcrAvailability(v.(bool), &locations)
		}
	}

	if len(locations) == 1 {
		result := locations[0]
		foundLocation = &result
	} else if len(locations) > 1 {
		return errors.New(TooManyLocationsError)
	} else if foundLocation == nil {
		return errors.New(NoLocationsFoundError)
	}

	d.SetId(strconv.Itoa(foundLocation.ID))
	d.Set("name", foundLocation.Name)
	d.Set("country", foundLocation.Country)
	d.Set("live_date", foundLocation.LiveDate)
	d.Set("site_code", foundLocation.SiteCode)
	d.Set("address", foundLocation.Address)
	d.Set("latitude", foundLocation.Latitude)
	d.Set("longitude", foundLocation.Longitude)
	d.Set("market", foundLocation.Market)
	d.Set("metro", foundLocation.Metro)
	d.Set("mcr_available", foundLocation.VRouterAvailable)
	d.Set("id", foundLocation.ID)
	d.Set("status", foundLocation.Status)

	return nil
}
