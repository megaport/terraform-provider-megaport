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
	"reflect"
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

	// assemble a end
	aEndConfiguration, aEndPortId, _ := ResourceMegaportVXCCreate_generate_AEnd(d, m)

	// convert schema ends to map for access
	bEndSchemaMap := d.Get("b_end").(*schema.Set).List()[0].(map[string]interface{})

	// assemble b end
	bEndConfiguration := types.VXCOrderBEndConfiguration{
		ProductUID: bEndSchemaMap["port_id"].(string),
		VLAN:       bEndSchemaMap["requested_vlan"].(int),
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

	aEndConfiguration := []interface{}{map[string]interface{}{
		"port_id":        vxcDetails.AEndConfiguration.UID,
		"owner_uid":      vxcDetails.AEndConfiguration.OwnerUID,
		"name":           vxcDetails.AEndConfiguration.Name,
		"location":       vxcDetails.AEndConfiguration.Location,
		"assigned_vlan":  vxcDetails.AEndConfiguration.VLAN,
		"requested_vlan": requested_a_vlan,
	}}

	if aEndErr := d.Set("a_end", aEndConfiguration); aEndErr != nil {
		return aEndErr
	}

	bEndConfiguration := []interface{}{map[string]interface{}{
		"port_id":        vxcDetails.BEndConfiguration.UID,
		"owner_uid":      vxcDetails.BEndConfiguration.OwnerUID,
		"name":           vxcDetails.BEndConfiguration.Name,
		"location":       vxcDetails.BEndConfiguration.Location,
		"assigned_vlan":  vxcDetails.AEndConfiguration.VLAN,
		"requested_vlan": requested_b_vlan,
	}}

	if bEndErr := d.Set("b_end", bEndConfiguration); bEndErr != nil {
		return bEndErr
	}

	if mcrAEndConfiguration, err := vxc.UnmarshallMcrAEndConfig(vxcDetails); err == nil && mcrAEndConfiguration != nil {
		d.Set("a_end_mcr_configuration", mcrAEndConfiguration)
	}

	return nil
}

// TODO: See if we can do a .HasChange on the subitem for vlans.
//       ** This is the expected behaviour of StackSet - the item is hashed and changes are across
//       ** the whole StackSet. I need to think about the structure of data to pick up the modifications better.
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
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	// convert schema to map for param access
	aEndSchemaMap := d.Get("a_end").(*schema.Set).List()[0].(map[string]interface{})

	// vlan
	aEndVlan := 0
	if newVlan, v_ok := aEndSchemaMap["requested_vlan"].(int); v_ok {
		aEndVlan = newVlan
	}

	// MCR configuration
	interfaces := []types.PartnerConfigInterface{}
	mcrInterface, _ := vxc.MarshallMcrAEndConfig(d)

	// Add interface if not empty
	if reflect.DeepEqual(mcrInterface, types.PartnerConfigInterface{}) {
		aEndConfiguration := types.VXCOrderAEndConfiguration{
			VLAN: aEndVlan,
		}
		return aEndConfiguration, aEndSchemaMap["port_id"].(string), nil
	} else {
		interfaces = append(interfaces, mcrInterface)
		aEndConfiguration := types.VXCOrderAEndConfiguration{
			VLAN: aEndVlan,
			PartnerConfig: types.VXCOrderAEndPartnerConfig{
				Interfaces: interfaces,
			},
		}
		return aEndConfiguration, aEndSchemaMap["port_id"].(string), nil
	}

}
