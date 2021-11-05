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
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/megaportgo/types"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportAWSConnection() *schema.Resource {
	return &schema.Resource{
		Create:        resourceMegaportAWSConnectionCreate,
		Read:          resourceMegaportAWSConnectionRead,
		Update:        resourceMegaportAWSConnectionUpdate,
		Delete:        resourceMegaportAWSConnectionUpdateDelete,
		Schema:        schema_megaport.ResourceAWSConnectionVXCSchema(),
		CustomizeDiff: resourceMegaportAWSConnectionDiff,
	}
}

func resourceMegaportAWSConnectionDiff(d *schema.ResourceDiff, meta interface{}) error {

	vlan_match, partner_config_absent := false, false
	old_assigned_vlan, new_requested_vlan := 0, 0

	if !d.HasChange("a_end") {
		log.Println("[DEBUG]", "a_end had no change")
		return nil
	}

	// compare states
	old, new := d.GetChange("a_end")

	// cast existing state to map
	if old_schema, aok := old.(*schema.Set); aok && len(old_schema.List()) > 0 {

		old_a_map := old_schema.List()[0].(map[string]interface{})
		old_assigned_vlan = old_a_map["assigned_vlan"].(int)

	} else {
		log.Println("[DEBUG]", "No a_end - do not modify")
		return nil
	}

	// cast planned changes to map
	if new_schema, aok := new.(*schema.Set); aok && len(new_schema.List()) > 0 {

		new_a_map := new_schema.List()[0].(map[string]interface{})
		if partner_schema, aok := new_a_map["partner_configuration"].(*schema.Set); aok && len(partner_schema.List()) <= 0 {

			// partner_configuration is absent - a_end should be veto'd if there is not change to vlan settings
			partner_config_absent = true
			new_requested_vlan = new_a_map["requested_vlan"].(int)

		}

	}

	if new_requested_vlan == 0 {
		vlan_match = true
	} else if new_requested_vlan == old_assigned_vlan {
		log.Printf("[DEBUG] newly requested vlan matches previous state computed value, requested_vlan (%q) = assigned_vlan(%q)", fmt.Sprint(new_requested_vlan), fmt.Sprint(old_assigned_vlan))
		vlan_match = true
	}

	if partner_config_absent && vlan_match {
		log.Println("[DEBUG] CLEARING a_end")
		d.Clear("a_end")
	}

	return nil
}

func resourceMegaportAWSConnectionCreate(d *schema.ResourceData, m interface{}) error {
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	cspSettings := d.Get("csp_settings").(*schema.Set).List()[0].(map[string]interface{})
	vlan := 0

	attachToId := cspSettings["attached_to"].(string)
	name := d.Get("vxc_name").(string)
	rateLimit := d.Get("rate_limit").(int)

	connectType := types.CONNECT_TYPE_AWS_VIF
	hostedConnection := cspSettings["hosted_connection"].(bool)
	if hostedConnection {
		connectType = types.CONNECT_TYPE_AWS_HOSTED_CONNECTION
	}

	partnerConfig := types.AWSVXCOrderAEndPartnerConfig{
		Interfaces: []types.PartnerConfigInterface{},
	}

	// infer a_end configuration
	if a_end, a_ok := d.GetOk("a_end"); a_ok && len(a_end.(*schema.Set).List()) > 0 {

		// cast to a map
		a_end_map := a_end.(*schema.Set).List()[0].(map[string]interface{})

		// infer vlan
		if newVlan, v_ok := a_end_map["requested_vlan"].(int); v_ok {
			vlan = newVlan
		}

		if partner_config, p_ok := a_end_map["partner_configuration"].(*schema.Set); p_ok && len(partner_config.List()) > 0 {

			partner_config_map := partner_config.List()[0].(map[string]interface{})

			// init config props
			ip_addresses_list := []string{}
			bfd_configuration := types.BfdConfig{}
			bgp_connection_list := []types.BgpConnectionConfig{}
			partnerConfigInterface := types.PartnerConfigInterface{}

			// extract ip addresses list
			if ip_addresses, ip_ok := partner_config_map["ip_addresses"].([]interface{}); ip_ok {

				for _, ip_address := range ip_addresses {
					i := ip_address.(string)
					ip_addresses_list = append(ip_addresses_list, i)
				}

				partnerConfigInterface.IpAddresses = ip_addresses_list
			}

			// extract BFD settings
			if bfd_config, bfd_ok := partner_config_map["bfd_configuration"].(*schema.Set); bfd_ok && len(bfd_config.List()) > 0 {

				bfd_config_map := bfd_config.List()[0].(map[string]interface{})
				bfd_configuration = types.BfdConfig{
					TxInterval: bfd_config_map["tx_internal"].(int),
					RxInterval: bfd_config_map["rx_internal"].(int),
					Multiplier: bfd_config_map["multiplier"].(int),
				}

				partnerConfigInterface.Bfd = bfd_configuration
			}

			// extract bgp connections
			if bgp_connections, bgp_ok := partner_config_map["bgp_connection"].([]interface{}); bgp_ok {

				for _, bgp_connection := range bgp_connections {

					i := bgp_connection.(map[string]interface{})

					new_bgp_connection := types.BgpConnectionConfig{
						PeerAsn:        i["peer_asn"].(int),
						LocalIpAddress: i["local_ip_address"].(string),
						PeerIpAddress:  i["peer_ip_address"].(string),
						Password:       i["password"].(string),
						Shutdown:       i["shutdown"].(bool),
						Description:    i["description"].(string),
						MedIn:          i["med_in"].(int),
						MedOut:         i["med_out"].(int),
						BfdEnabled:     i["bfd_enabled"].(bool),
					}

					bgp_connection_list = append(bgp_connection_list, new_bgp_connection)

				}

				partnerConfigInterface.BgpConnections = bgp_connection_list
			}

			// add to config
			partnerConfig.Interfaces = append(partnerConfig.Interfaces, partnerConfigInterface)

		}

	}

	aEndConfiguration := types.AWSVXCOrderAEndConfiguration{
		VLAN:          vlan,
		PartnerConfig: partnerConfig,
	}

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
		},
	}

	vxcId, buyErr := vxc.BuyAWSVXC(
		attachToId,
		name,
		rateLimit,
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
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	vxcDetails, retrievalErr := vxc.GetVXCDetails(d.Id())

	if retrievalErr != nil {
		return retrievalErr
	}

	// base VXC read
	ResourceMegaportVXCRead(d, m)

	if cspConnectionList, ok := vxcDetails.Resources.CspConnection.([]interface{}); ok {

		for _, conn := range cspConnectionList {

			cspConnection := conn.(map[string]interface{})
			flattenCspConnection(d, cspConnection)

		}

	} else if cspConnection, ok := vxcDetails.Resources.CspConnection.(map[string]interface{}); ok {

		flattenCspConnection(d, cspConnection)

	} else {
		return errors.New(CannotSetVIFError)
	}

	return nil

}

func flattenCspConnection(d *schema.ResourceData, cspConnection map[string]interface{}) error {

	log.Println("flattenCspConnection - processing")
	connectType := cspConnection["connectType"].(string)
	resourceType := cspConnection["resource_name"].(string)

	// infer a_end configuration
	a_end, a_ok := d.GetOk("a_end")

	if !a_ok {
		return errors.New("a_end could not be unmarshalled")
	}

	if connectType == "AWS" {

		if _, exists := cspConnection["vif_id"]; exists {
			d.Set("aws_id", cspConnection["vif_id"].(string))
		}

	} else if connectType == "VROUTER" &&
		resourceType == "a_csp_connection" {

		if partner_interfaces, ok := cspConnection["interfaces"].([]interface{}); ok {

			// initialise return list
			partner_configuration_list := []interface{}{}

			for _, partner_interface := range partner_interfaces {

				partner_configuration := map[string]interface{}{}

				partner_interface_map, pi_ok := partner_interface.(map[string]interface{})
				if !pi_ok {
					log.Println("Error casting partner_interface_map")
				}

				// add ip addresses to configuration
				partner_configuration["ip_addresses"] = partner_interface_map["ipAddresses"]

				// extract bfd settings
				bfd_map, bfd_ok := partner_interface_map["bfd"].(map[string]interface{})
				if bfd_ok {

					// add bfd to configuration
					partner_configuration["bfd_configuration"] = []interface{}{map[string]interface{}{
						"tx_internal": bfd_map["txInterval"],
						"rx_internal": bfd_map["rxInterval"],
						"multiplier":  bfd_map["multiplier"],
					}}

				}

				// extract bgp configurations
				bgp_connection_list := []interface{}{}
				if bgpConnections, bgp_ok := partner_interface_map["bgpConnections"].([]interface{}); bgp_ok {
					for _, bgpConnection := range bgpConnections {

						bgp_connection_map, bgpm_ok := bgpConnection.(map[string]interface{})
						if bgpm_ok {

							new_bgp_connection := map[string]interface{}{
								"peer_asn":         bgp_connection_map["peerAsn"],
								"local_ip_address": bgp_connection_map["localIpAddress"],
								"peer_ip_address":  bgp_connection_map["peerIpAddress"],
								"password":         bgp_connection_map["password"],
								"shutdown":         bgp_connection_map["shutdown"],
								"description":      bgp_connection_map["description"],
								"med_in":           bgp_connection_map["medIn"],
								"med_out":          bgp_connection_map["medOut"],
								"bfd_enabled":      bgp_connection_map["bfdEnabled"],
							}

							bgp_connection_list = append(bgp_connection_list, new_bgp_connection)
						}

					} // end bgp connections loop

					// add bgp to configuration
					partner_configuration["bgp_connection"] = bgp_connection_list

				} // end bgp connection inspection

				// add configuration to list
				partner_configuration_list = append(partner_configuration_list, partner_configuration)

			} // end interface loop

			// cast a_end to a map
			a_end_map := a_end.(*schema.Set).List()[0].(map[string]interface{})

			// forming
			aEndConfiguration := []interface{}{map[string]interface{}{
				"port_id":               a_end_map["UID"],
				"owner_uid":             a_end_map["owner_uid"],
				"name":                  a_end_map["name"],
				"location":              a_end_map["location"],
				"assigned_vlan":         a_end_map["assigned_vlan"],
				"requested_vlan":        a_end_map["requested_vlan"],
				"partner_configuration": partner_configuration_list,
			}}

			if aEndErr := d.Set("a_end", aEndConfiguration); aEndErr != nil {
				log.Printf("%s", aEndErr)
				return aEndErr
			}

		}

	}

	return nil
}

func resourceMegaportAWSConnectionUpdate(d *schema.ResourceData, m interface{}) error {
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
