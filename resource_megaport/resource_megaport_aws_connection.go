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

package resource_megaport

import (
	"errors"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/megaportgo/types"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportAWSConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportAWSConnectionCreate,
		Read:   resourceMegaportAWSConnectionRead,
		Update: resourceMegaportAWSConnectionUpdate,
		Delete: resourceMegaportAWSConnectionUpdateDelete,
		Schema: schema_megaport.ResourceAWSConnectionVXCSchema(),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceMegaportAWSConnectionCreate(d *schema.ResourceData, m interface{}) error {
	log.Println("!!! resourceMegaportAWSConnectionCreate...")
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	// assemble a end
	aEndConfiguration, aEndPortId, _ := ResourceMegaportVXCCreate_generate_AEnd(d, m)

	// csp settings
	cspSettings := d.Get("csp_settings").(*schema.Set).List()[0].(map[string]interface{})
	connectType := types.CONNECT_TYPE_AWS_VIF
	hostedConnection := cspSettings["hosted_connection"].(bool)
	if hostedConnection {
		connectType = types.CONNECT_TYPE_AWS_HOSTED_CONNECTION
	}

	// assemble b end
	bEndConfiguration := types.AWSVXCOrderBEndConfiguration{
		ProductUID: cspSettings["requested_product_id"].(string),
		PartnerConfig: types.AWSVXCOrderBEndPartnerConfig{
			ASN:               cspSettings["requested_asn"].(int),
			AmazonASN:         cspSettings["amazon_asn"].(int),
			AuthKey:           cspSettings["auth_key"].(string),
			Prefixes:          cspSettings["prefixes"].(string),
			CustomerIPAddress: cspSettings["customer_ip"].(string),
			AmazonIPAddress:   cspSettings["amazon_ip"].(string),
			ConnectType:       connectType,
			Type:              cspSettings["visibility"].(string),
			OwnerAccount:      cspSettings["amazon_account"].(string),
			ConnectionName:    cspSettings["connection_name"].(string),
		},
	}

	// make order
	vxcId, buyErr := vxc.BuyAWSVXC(
		aEndPortId,
		d.Get("vxc_name").(string),
		d.Get("rate_limit").(int),
		aEndConfiguration,
		bEndConfiguration,
	)

	if buyErr != nil {
		return buyErr
	}

	d.SetId(vxcId)
	vxc.WaitForVXCProvisioning(vxcId)
	return resourceMegaportAWSConnectionRead(d, m)
}

func resourceMegaportAWSConnectionRead(d *schema.ResourceData, m interface{}) error {
	log.Println("!!! resourceMegaportAWSConnectionRead...")
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	vxcDetails, retrievalErr := vxc.GetVXCDetails(d.Id())

	if retrievalErr != nil {
		return retrievalErr
	}

	d.Set("vxc_internal_type", "aws")

	// Aws read
	if vifId := vxc.ExtractAwsId(vxcDetails); vifId != "" {
		d.Set("aws_id", vifId)
	}

	// AWS CSP read
	PartnerConfig, err := vxc.ExtractAWSPartnerConfig(vxcDetails)
	if err != nil {
		return err
	}
	if PartnerConfig != nil {
		// only read csp_settings into state if we don't have any already (ie. we're importing)
		// otherwise erroneous options that API ignores may cause a ForceNew loop on each apply
		if _, ok := d.GetOk("csp_settings"); ok == false {
			cspSettings := make(map[string]interface{})
			cspSettings["hosted_connection"] = PartnerConfig.ConnectType == types.CONNECT_TYPE_AWS_HOSTED_CONNECTION
			cspSettings["visibility"] = "private"
			if PartnerConfig.Type != "" {
				cspSettings["visibility"] = PartnerConfig.Type
			}
			cspSettings["amazon_account"] = PartnerConfig.OwnerAccount
			cspSettings["requested_asn"] = PartnerConfig.ASN
			cspSettings["amazon_asn"] = PartnerConfig.AmazonASN
			cspSettings["auth_key"] = PartnerConfig.AuthKey
			cspSettings["prefixes"] = PartnerConfig.Prefixes
			cspSettings["customer_ip"] = PartnerConfig.CustomerIPAddress
			cspSettings["amazon_ip"] = PartnerConfig.AmazonIPAddress
			cspSettings["connection_name"] = PartnerConfig.ConnectionName
			// requested_product_id requires special handling bacause of a quirk in the API
			// the b end id is not necessarily what we requested (can be fulfilled from a comparable product not set as vxcPermitted)
			partner := m.(*terraform_utility.MegaportClient).Partner
			partnerPorts, retrievalErr := partner.GetAllPartnerMegaports()
			if retrievalErr != nil {
				return retrievalErr
			}
			var bEndMegaport *types.PartnerMegaport
			// iterate all ports instead of using FilterPartnerMegaport because that will skip anything not vxcPermitted
			for i := range partnerPorts {
				if partnerPorts[i].ProductUID == vxcDetails.BEndConfiguration.UID {
					bEndMegaport = &partnerPorts[i]
					break
				}
			}
			if bEndMegaport == nil {
				return errors.New(InvalidPartnerBEnd)
			}
			// now partially reimplement dataMegaportPartnerPortRead to get the current orderable port (will match the data lookup)
			partner.FilterPartnerMegaportByProductName(&partnerPorts, bEndMegaport.ProductName, true)
			partner.FilterPartnerMegaportByConnectType(&partnerPorts, bEndMegaport.ConnectType, true)
			partner.FilterPartnerMegaportByLocationId(&partnerPorts, bEndMegaport.LocationId)
			partner.FilterPartnerMegaportByCompanyName(&partnerPorts, bEndMegaport.CompanyName, true)
			if len(partnerPorts) == 0 {
				return errors.New(NoMatchingPartnerPortsAtLocationError)
			} else if len(partnerPorts) > 1 {
				return errors.New(TooManyPartnerPortsError)
			}
			cspSettings["requested_product_id"] = partnerPorts[0].ProductUID			
			if err := d.Set("csp_settings", []map[string]interface{}{cspSettings}); err != nil {
				return err
			}
		}
	}
	
	// base VXC read
	return ResourceMegaportVXCRead(d, m)

}

func resourceMegaportAWSConnectionUpdate(d *schema.ResourceData, m interface{}) error {
	log.Println("!!! resourceMegaportAWSConnectionUpdate...")
	vxc := m.(*terraform_utility.MegaportClient).Vxc
	aVlan := 0

	if aEndConfiguration, ok := d.GetOk("a_end"); ok {
		if newVlan, aOk := aEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int); aOk {
			aVlan = newVlan
		}
	}

	cspSettings := d.Get("csp_settings").(*schema.Set).List()[0].(map[string]interface{})
	hostedConnection := cspSettings["hosted_connection"].(bool)

	if d.HasChange("rate_limit") && hostedConnection {
		return errors.New(CannotChangeHostedConnectionRateError)
	}

	if d.HasChange("vxc_name") || d.HasChange("rate_limit") || d.HasChange("a_end") {
		_, updateErr := vxc.UpdateVXC(d.Id(), d.Get("vxc_name").(string),
			d.Get("rate_limit").(int),
			aVlan,
			0)

		if updateErr != nil {
			return updateErr
		}

		vxc.WaitForVXCUpdated(d.Id(), d.Get("vxc_name").(string),
			d.Get("rate_limit").(int),
			aVlan,
			0)
	}

	return resourceMegaportAWSConnectionRead(d, m)
}

func resourceMegaportAWSConnectionUpdateDelete(d *schema.ResourceData, m interface{}) error {
	return ResourceMegaportVXCDelete(d, m)
}
