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
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
)

func MegaportAWSConnection() *schema.Resource {
	return &schema.Resource{
		Read:   dataMegaportAwsConnectionRead,
		Schema: schema_megaport.DataVXCSchema(),
	}
}

func dataMegaportAwsConnectionRead(d *schema.ResourceData, m interface{}) error {
	readErr := DataMegaportVXCRead(d, m)

	if readErr != nil {
		return readErr
	}

	d.Set("vxc_internal_type", "aws")

	return nil
}
