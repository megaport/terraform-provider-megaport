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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/megaportgo/types"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportVXC() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportVXCCreate,
		Read:   ResourceMegaportVXCRead,
		Update: ResourceMegaportVXCUpdate,
		Delete: ResourceMegaportVXCDelete,
		Schema: schema_megaport.ResourceVXCSchema(),
	}
}

func resourceMegaportVXCCreate(d *schema.ResourceData, m interface{}) error {
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	// Assemble A-end config.
	aEndConfiguration, aEndPortId, _ := ResourceMegaportVXCCreate_generate_AEnd(d, m)

	// Assemble B-end config.
	bEndMap := d.Get("b_end").(*schema.Set).List()[0].(map[string]interface{})
	bEndProductUid, _ := bEndMap["port_id"].(string)
	isPort := bEndProductUid != ""
	bEndConfiguration := types.VXCOrderBEndConfiguration{ProductUID: bEndProductUid}

	// vlan
	if vl, ok := bEndMap["requested_vlan"]; ok {
		bEndConfiguration.VLAN = vl.(int)
	}

	// MVE config.
	if !isPort {
		bEndConfiguration.ProductUID = bEndMap["mve_id"].(string)
		bEndConfiguration.VXCOrderMVEConfig = &types.VXCOrderMVEConfig{}

		if ivl, ok := bEndMap["inner_vlan"]; ok {
			bEndConfiguration.InnerVLAN = ivl.(int)
		}
		if vn, ok := bEndMap["vnic_index"]; ok {
			bEndConfiguration.NetworkInterfaceIndex = vn.(int)
		}
	}

	// make order
	vxcId, buyErr := vxc.BuyVXC(
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

	return ResourceMegaportVXCRead(d, m)
}

func ResourceMegaportVXCRead(d *schema.ResourceData, m interface{}) error {
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	vxcDetails, retrievalErr := vxc.GetVXCDetails(d.Id())
	if retrievalErr != nil {
		return retrievalErr
	}

	requested_a_vlan := vxcDetails.AEndConfiguration.VLAN
	requested_b_vlan := vxcDetails.BEndConfiguration.VLAN

	if aEndConfiguration, ok := d.GetOk("a_end"); ok {
		requested_a_vlan = aEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int)
	}

	if bEndConfiguration, ok := d.GetOk("b_end"); ok && d.Get("vxc_internal_type") == "vxc" {
		requested_b_vlan = bEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int)
	}

	d.Set("uid", vxcDetails.UID)
	d.Set("service_id", vxcDetails.ServiceID)
	d.Set("vxc_name", vxcDetails.Name)
	d.Set("vxc_type", vxcDetails.Type)
	d.Set("rate_limit", vxcDetails.RateLimit)
	d.Set("provisioning_status", vxcDetails.ProvisioningStatus)
	d.Set("created_by", vxcDetails.CreatedBy)
	d.Set("live_date", vxcDetails.LiveDate)
	d.Set("create_date", vxcDetails.CreateDate)
	d.Set("company_name", vxcDetails.CompanyName)
	d.Set("locked", vxcDetails.Locked)
	d.Set("admin_locked", vxcDetails.AdminLocked)

	// Setup A-end config.
	aEndConfiguration := map[string]interface{}{
		"owner_uid":      vxcDetails.AEndConfiguration.OwnerUID,
		"name":           vxcDetails.AEndConfiguration.Name,
		"location":       vxcDetails.AEndConfiguration.Location,
		"assigned_vlan":  vxcDetails.AEndConfiguration.VLAN,
		"requested_vlan": requested_a_vlan,
	}

	if vxcDetails.AEndConfiguration.NetworkInterfaceIndex > 0 {
		aEndConfiguration["mve_id"] = vxcDetails.AEndConfiguration.UID
		aEndConfiguration["inner_vlan"] = vxcDetails.AEndConfiguration.InnerVLAN
		aEndConfiguration["vnic_index"] = vxcDetails.AEndConfiguration.NetworkInterfaceIndex
	} else {
		aEndConfiguration["port_id"] = vxcDetails.AEndConfiguration.UID
	}

	if aEndErr := d.Set("a_end", []interface{}{aEndConfiguration}); aEndErr != nil {
		return aEndErr
	}

	// Setup B-end config.
	bEndConfiguration := map[string]interface{}{
		"owner_uid":      vxcDetails.BEndConfiguration.OwnerUID,
		"name":           vxcDetails.BEndConfiguration.Name,
		"location":       vxcDetails.BEndConfiguration.Location,
		"assigned_vlan":  vxcDetails.AEndConfiguration.VLAN,
		"requested_vlan": requested_b_vlan,
	}

	if vxcDetails.BEndConfiguration.NetworkInterfaceIndex > 0 {
		bEndConfiguration["mve_id"] = vxcDetails.BEndConfiguration.UID
		bEndConfiguration["inner_vlan"] = vxcDetails.BEndConfiguration.InnerVLAN
		bEndConfiguration["vnic_index"] = vxcDetails.BEndConfiguration.NetworkInterfaceIndex
	} else {
		bEndConfiguration["port_id"] = vxcDetails.BEndConfiguration.UID
	}

	if bEndErr := d.Set("b_end", []interface{}{bEndConfiguration}); bEndErr != nil {
		return bEndErr
	}

	if mcrAEndConfiguration, err := vxc.UnmarshallMcrAEndConfig(vxcDetails); err == nil && mcrAEndConfiguration != nil {
		d.Set("a_end_mcr_configuration", mcrAEndConfiguration)
	}

	return nil
}

// TODO: See if we can do a .HasChange on the subitem for vlans.
//
//	** This is the expected behaviour of StackSet - the item is hashed and changes are across
//	** the whole StackSet. I need to think about the structure of data to pick up the modifications better.
func ResourceMegaportVXCUpdate(d *schema.ResourceData, m interface{}) error {
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	aVlan := 0
	bVlan := 0

	if aEndConfiguration, ok := d.GetOk("a_end"); ok {
		if newVlan, aOk := aEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int); aOk {
			aVlan = newVlan
		}
	}

	if bEndConfiguration, ok := d.GetOk("b_end"); ok {
		if newVlan, bOk := bEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int); bOk {
			bVlan = newVlan
		}
	}

	if d.HasChange("vxc_name") || d.HasChange("rate_limit") || d.HasChange("a_end") || d.HasChange("b_end") {
		vxc.UpdateVXC(d.Id(), d.Get("vxc_name").(string),
			d.Get("rate_limit").(int),
			aVlan,
			bVlan)
		vxc.WaitForVXCUpdated(d.Id(), d.Get("vxc_name").(string),
			d.Get("rate_limit").(int),
			aVlan,
			bVlan)
	}

	return ResourceMegaportVXCRead(d, m)
}

func ResourceMegaportVXCDelete(d *schema.ResourceData, m interface{}) error {
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	deleteSuccess, deleteError := vxc.DeleteVXC(d.Id(), true)

	if deleteSuccess {
		log.Println("Wait for resource cleanup...")
		time.Sleep(40 * time.Second)
		return nil
	} else {
		return errors.New(fmt.Sprintf("Error deleting resource %s: %s", d.Id(), deleteError))
	}

}

func ResourceMegaportVXCCreate_generate_AEnd(d *schema.ResourceData, m interface{}) (types.VXCOrderAEndConfiguration, string, error) {
	// convert schema to map for param access
	aEndSchemaMap := d.Get("a_end").(*schema.Set).List()[0].(map[string]interface{})

	productUid, _ := aEndSchemaMap["port_id"].(string)
	isPort := productUid != ""
	aEndConfig := types.VXCOrderAEndConfiguration{}

	// vlan
	if newVlan, ok := aEndSchemaMap["requested_vlan"].(int); ok {
		aEndConfig.VLAN = newVlan
	}

	// MCR configuration.
	if mcrInterface, _ := MarshallMcrAEndConfig(d, m); mcrInterface != nil {
		aEndConfig.PartnerConfig = types.VXCOrderAEndPartnerConfig{
			Interfaces: []types.PartnerConfigInterface{*mcrInterface},
		}
	}

	// MVE configuration.
	if !isPort {
		productUid = aEndSchemaMap["mve_id"].(string)
		aEndConfig.VXCOrderMVEConfig = &types.VXCOrderMVEConfig{}

		if ivl, ok := aEndSchemaMap["inner_vlan"]; ok {
			aEndConfig.InnerVLAN = ivl.(int)
		}
		if vni, ok := aEndSchemaMap["vnic_index"]; ok {
			aEndConfig.NetworkInterfaceIndex = vni.(int)
		}
	}

	return aEndConfig, productUid, nil
}

func MarshallMcrAEndConfig(d *schema.ResourceData, m interface{}) (*types.PartnerConfigInterface, error) {
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	// Shortcut if no MCR config.
	a_end_mcr_configuration, ok := d.GetOk("a_end_mcr_configuration")
	if !ok || len(a_end_mcr_configuration.(*schema.Set).List()) < 1 {
		return nil, nil
	}

	mcrConfig := &types.PartnerConfigInterface{}

	// cast to a map
	mcr_map := a_end_mcr_configuration.(*schema.Set).List()[0].(map[string]interface{})

	// init config props
	ip_addresses_list := []string{}
	ip_routes_list := []types.IpRoute{}
	nat_ip_addresses_list := []string{}
	bfd_configuration := types.BfdConfig{}
	bgp_connection_list := []types.BgpConnectionConfig{}
	permit_exports_list := []string{}
	deny_exports_list := []string{}
	var import_permit_list_id int
	var import_deny_list_id int
	var export_permit_list_id int
	var export_deny_list_id int

	// extract ip addresses list
	if ip_addresses, ip_ok := mcr_map["ip_addresses"].([]interface{}); ip_ok {

		for _, ip_address := range ip_addresses {
			i := ip_address.(string)
			ip_addresses_list = append(ip_addresses_list, i)
		}

		mcrConfig.IpAddresses = ip_addresses_list
	}

	// extract static ip routes
	if ip_routes, ipr_ok := mcr_map["ip_route"].([]interface{}); ipr_ok {

		for _, ip_route := range ip_routes {

			i := ip_route.(map[string]interface{})

			new_ip_route := types.IpRoute{
				Prefix:      i["prefix"].(string),
				Description: i["description"].(string),
				NextHop:     i["next_hop"].(string),
			}

			ip_routes_list = append(ip_routes_list, new_ip_route)

		}

		mcrConfig.IpRoutes = ip_routes_list
	}

	// extract nat ip addresses list
	if nat_ip_addresses, nat_ok := mcr_map["nat_ip_addresses"].([]interface{}); nat_ok {

		for _, nat_ip_address := range nat_ip_addresses {
			i := nat_ip_address.(string)
			nat_ip_addresses_list = append(nat_ip_addresses_list, i)
		}

		mcrConfig.NatIpAddresses = nat_ip_addresses_list
	}

	// extract BFD settings
	if bfd_config, bfd_ok := mcr_map["bfd_configuration"].(*schema.Set); bfd_ok && len(bfd_config.List()) > 0 {

		bfd_config_map := bfd_config.List()[0].(map[string]interface{})
		bfd_configuration = types.BfdConfig{
			TxInterval: bfd_config_map["tx_interval"].(int),
			RxInterval: bfd_config_map["rx_interval"].(int),
			Multiplier: bfd_config_map["multiplier"].(int),
		}

		mcrConfig.Bfd = bfd_configuration
	}

	// extract bgp connections
	if bgp_connections, bgp_ok := mcr_map["bgp_connection"].([]interface{}); bgp_ok {

		for _, bgp_connection := range bgp_connections {

			i := bgp_connection.(map[string]interface{})

			mcrId := d.Get("a_end").(*schema.Set).List()[0].(map[string]interface{})["port_id"].(string)
			prefix_filter_lists, _ := vxc.GetPrefixFilterLists(mcrId)

			// extract permit exports list
			if permit_exports, p_ok := i["permit_export_to"].([]interface{}); p_ok {

				for _, permit_export := range permit_exports {
					x := permit_export.(string)
					permit_exports_list = append(permit_exports_list, x)
				}
			}

			// extract deny exports list
			if deny_exports, d_ok := i["deny_export_to"].([]interface{}); d_ok {

				for _, deny_export := range deny_exports {
					x := deny_export.(string)
					deny_exports_list = append(deny_exports_list, x)
				}
			}

			// extract import permit list
			if len(i["import_permit_list"].(string)) > 0 {
				import_permit_list := i["import_permit_list"]

				for i := 0; i < len(prefix_filter_lists); i++ {
					if prefix_filter_lists[i].Description == import_permit_list {
						import_permit_list_id = prefix_filter_lists[i].Id
						break
					}
				}
			}

			// extract import deny list
			if len(i["import_deny_list"].(string)) > 0 {
				import_deny_list := i["import_deny_list"]

				for i := 0; i < len(prefix_filter_lists); i++ {
					if prefix_filter_lists[i].Description == import_deny_list {
						import_deny_list_id = prefix_filter_lists[i].Id
						break
					}
				}
			}

			// extract export permit list
			if len(i["export_permit_list"].(string)) > 0 {
				export_permit_list := i["export_permit_list"]

				for i := 0; i < len(prefix_filter_lists); i++ {
					if prefix_filter_lists[i].Description == export_permit_list {
						export_permit_list_id = prefix_filter_lists[i].Id
						break
					}
				}
			}

			// extract export deny list
			if len(i["export_deny_list"].(string)) > 0 {
				export_deny_list := i["export_deny_list"]

				for i := 0; i < len(prefix_filter_lists); i++ {
					if prefix_filter_lists[i].Description == export_deny_list {
						export_deny_list_id = prefix_filter_lists[i].Id
						break
					}
				}
			}

			new_bgp_connection := types.BgpConnectionConfig{
				PeerAsn:         i["peer_asn"].(int),
				LocalIpAddress:  i["local_ip_address"].(string),
				PeerIpAddress:   i["peer_ip_address"].(string),
				Password:        i["password"].(string),
				Shutdown:        i["shutdown"].(bool),
				Description:     i["description"].(string),
				MedIn:           i["med_in"].(int),
				MedOut:          i["med_out"].(int),
				BfdEnabled:      i["bfd_enabled"].(bool),
				ExportPolicy:    i["export_policy"].(string),
				PermitExportTo:  permit_exports_list,
				DenyExportTo:    deny_exports_list,
				ImportWhitelist: import_permit_list_id,
				ImportBlacklist: import_deny_list_id,
				ExportWhitelist: export_permit_list_id,
				ExportBlacklist: export_deny_list_id,
			}

			bgp_connection_list = append(bgp_connection_list, new_bgp_connection)

		}

		mcrConfig.BgpConnections = bgp_connection_list
	}

	return mcrConfig, nil
}
