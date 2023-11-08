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
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/megaportgo/service/mve"
	"github.com/megaport/megaportgo/types"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportMVE() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportMVECreate,
		Read:   resourceMegaportMVERead,
		Update: resourceMegaportMVEUpdate,
		Delete: resourceMegaportMVEDelete,
		Schema: schema_megaport.ResourceMegaportMVESchema(),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceMegaportMVECreate(d *schema.ResourceData, m interface{}) error {
	mve := m.(*terraform_utility.MegaportClient).Mve

	name := d.Get("mve_name").(string)
	term := d.Get("term").(int)
	locationId := d.Get("location_id").(int)

	// Grab the vendor config and then force-set some keys from the config.
	vendorConfig := d.Get("vendor_config").(map[string]interface{})
	origVendorConfig := make(map[string]interface{})
	for k, v := range vendorConfig {
		origVendorConfig[k] = v
	}

	vendorConfig["vendor"] = d.Get("vendor").(string)
	vendorConfig["productSize"] = d.Get("size").(string)
	vendorConfig["imageId"] = d.Get("image_id").(int)

	// Extract any vNIC configuration.
	vnics := []*types.MVENetworkInterface{}
	if vn, ok := d.Get("vnic").([]interface{}); ok {
		for _, v := range vn {
			if m, ok := v.(map[string]interface{}); ok {
				if desc, ok := m["description"].(string); ok {
					vnics = append(vnics, &types.MVENetworkInterface{Description: desc})
				}
			}
		}
	}

	// Create a default vNIC if none defined.
	if len(vnics) == 0 {
		vnics = append(vnics, &types.MVENetworkInterface{Description: "Data Plane"})
	}

	uid, err := mve.BuyMVE(locationId, name, term, vendorConfig, vnics)
	if err != nil {
		return err
	}

	d.SetId(uid)
	if _, err := mve.WaitForMVEProvisioning(uid); err != nil {
		return err
	}

	details, err := fetchMVEDetails(mve, d)
	if err != nil {
		return err
	}
	populateBaseResourceData(details, d)

	// Set original vendor config back to avoid Terraform trying to modify it later.
	d.Set("vendor_config", origVendorConfig)

	return nil
}

func fetchMVEDetails(mve *mve.MVE, d *schema.ResourceData) (*types.MVE, error) {
	details, err := mve.GetMVEDetails(d.Id())
	if err != nil {
		return nil, err
	}

	return details, nil
}

func populateBaseResourceData(details *types.MVE, d *schema.ResourceData) {
	d.Set("uid", details.UID)
	d.Set("mve_name", details.Name)
	d.Set("type", details.Type)
	d.Set("provisioning_status", details.ProvisioningStatus)
	d.Set("create_date", details.CreateDate)
	d.Set("created_by", details.CreatedBy)
	d.Set("live_date", details.LiveDate)
	d.Set("market", details.Market)
	d.Set("location_id", details.LocationID)
	d.Set("marketplace_visibility", details.MarketplaceVisibility)
	d.Set("company_name", details.CompanyName)
	d.Set("term", details.ContractTermMonths)
	d.Set("locked", details.Locked)
	d.Set("admin_locked", details.AdminLocked)
	d.Set("vendor", details.Vendor)
	d.Set("size", details.Size)

	if vmr, ok := details.Resources["virtual_machine"]; ok {
		if vmList, ok := vmr.([]interface{}); ok {
			vm := vmList[0].(map[string]interface{})
			if image, ok := vm["image"].(map[string]interface{}); ok {
				d.Set("image_vendor", image["vendor"])
				d.Set("image_product", image["product"])
				d.Set("image_version", image["version"])
			}
		}
	}

	// Populate vNIC data.
	vnics := make([]map[string]interface{}, len(details.NetworkInterfaces))
	for i, vn := range details.NetworkInterfaces {
		vnics[i] = map[string]interface{}{
			"index":       int(i),
			"description": vn.Description,
			"vlan":        vn.VLAN,
		}
	}

	d.Set("vnic", vnics)
}

func resourceMegaportMVERead(d *schema.ResourceData, m interface{}) error {
	mve := m.(*terraform_utility.MegaportClient).Mve

	details, err := fetchMVEDetails(mve, d)
	if err != nil {
		return err
	}
	populateBaseResourceData(details, d)

	return nil
}

func resourceMegaportMVEUpdate(d *schema.ResourceData, m interface{}) error {
	mve := m.(*terraform_utility.MegaportClient).Mve

	// Initially, check if there are changes that we don't support.
	if d.HasChange("vendor_config") {
		return fmt.Errorf("changes to vendor config currently unsupported")
	}
	if d.HasChange("vnic") {
		return fmt.Errorf("changes to vNIC config currently unsupported")
	}

	if d.HasChange("mve_name") {
		if _, err := mve.ModifyMVE(d.Id(), d.Get("mve_name").(string)); err != nil {
			return err
		}
	}

	details, err := fetchMVEDetails(mve, d)
	if err != nil {
		return err
	}
	populateBaseResourceData(details, d)

	return nil
}

func resourceMegaportMVEDelete(d *schema.ResourceData, m interface{}) error {
	mve := m.(*terraform_utility.MegaportClient).Mve

	if _, err := mve.DeleteMVE(d.Id()); err != nil {
		return fmt.Errorf("Error deleting resource %s: %s", d.Id(), err)
	}

	return nil
}
