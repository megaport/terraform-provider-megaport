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

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportPartnerPort() *schema.Resource {
	return &schema.Resource{
		Read:   dataMegaportPartnerPortRead,
		Schema: schema_megaport.DataPartnerPortSchema(),
	}
}

func dataMegaportPartnerPortRead(d *schema.ResourceData, m interface{}) error {
	partner := m.(*terraform_utility.MegaportClient).Partner

	partnerPorts, retrievalErr := partner.GetAllPartnerMegaports()

	if retrievalErr != nil {
		return retrievalErr
	}

	productNameLookupErr := partner.FilterPartnerMegaportByProductName(&partnerPorts, d.Get("product_name").(string), true)
	connectTypeLookupErr := partner.FilterPartnerMegaportByConnectType(&partnerPorts, d.Get("connect_type").(string), true)
	locationLookupErr := partner.FilterPartnerMegaportByLocationId(&partnerPorts, d.Get("location_id").(int))
	companyNameLookupErr := partner.FilterPartnerMegaportByCompanyName(&partnerPorts, d.Get("company_name").(string), true)

	if productNameLookupErr != nil {
		return errors.New(ProductNameFilterTooStrictError)
	}

	if connectTypeLookupErr != nil {
		return errors.New(ConnectTypeFilterTooStrictError)
	}

	if locationLookupErr != nil {
		return errors.New(NoMatchingPartnerPortsAtLocationError)
	}

	if companyNameLookupErr != nil {
		return errors.New(CompanyNameFilterTooStrictError)
	}

	if len(partnerPorts) != 1 {
		return errors.New(TooManyPartnerPortsError)
	}

	// we choose the first port to match the criteria.
	chosenPort := partnerPorts[0]
	d.SetId(chosenPort.ProductUID)
	d.Set("connect_type", chosenPort.ConnectType)
	d.Set("product_name", chosenPort.ProductName)
	d.Set("company_name", chosenPort.CompanyName)
	d.Set("location_id", chosenPort.LocationId)
	d.Set("company_uid", chosenPort.CompanyUID)
	d.Set("speed", chosenPort.Speed)

	return nil
}
