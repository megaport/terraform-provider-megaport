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
	"github.com/megaport/megaportgo/vxc"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func MegaportGcpConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportGcpConnectionCreate,
		Read:   resourceMegaportGcpConnectionRead,
		Update: resourceMegaportGcpConnectionUpdate,
		Delete: resourceMegaportGcpConnectionDelete,
		Schema: schema_megaport.ResourceGcpConnectionVXCSchema(),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceMegaportGcpConnectionCreate(d *schema.ResourceData, m interface{}) error {
	cspSettings := d.Get("csp_settings").(*schema.Set).List()[0].(map[string]interface{})
	vlan := 0

	attachToId := cspSettings["attached_to"].(string)
	name := d.Get("vxc_name").(string)
	rateLimit := d.Get("rate_limit").(int)
	pairingKey := cspSettings["pairing_key"].(string)

	if aEndConfiguration, ok := d.GetOk("a_end"); ok {
		if newVlan, aOk := aEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int); aOk {
			vlan = newVlan
		}
	}

	vxcId, buyErr := vxc.BuyGoogleInterconnect(attachToId, name, rateLimit, vlan, pairingKey)

	if buyErr != nil {
		return buyErr
	}

	d.SetId(vxcId)
	vxc.WaitForVXCProvisioning(vxcId)
	return resourceMegaportGcpConnectionRead(d, m)
}

func resourceMegaportGcpConnectionRead(d *schema.ResourceData, m interface{}) error {
	return ResourceMegaportVXCRead(d, m)
}

func resourceMegaportGcpConnectionUpdate(d *schema.ResourceData, m interface{}) error {
	aVlan := 0

	if aEndConfiguration, ok := d.GetOk("a_end"); ok {
		if newVlan, aOk := aEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int); aOk {
			aVlan = newVlan
		}
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

	return resourceMegaportGcpConnectionRead(d, m)
}

func resourceMegaportGcpConnectionDelete(d *schema.ResourceData, m interface{}) error {
	return ResourceMegaportVXCDelete(d, m)
}
