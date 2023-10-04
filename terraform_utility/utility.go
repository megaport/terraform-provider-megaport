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

package terraform_utility

import (
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/megaportgo/config"
	"github.com/megaport/megaportgo/service/authentication"
	"github.com/megaport/megaportgo/service/location"
	"github.com/megaport/megaportgo/service/mcr"
	"github.com/megaport/megaportgo/service/partner"
	"github.com/megaport/megaportgo/service/port"
	"github.com/megaport/megaportgo/service/product"
	"github.com/megaport/megaportgo/service/vxc"
)

type MegaportClient struct {
	*MegaportServices
	DeletePorts bool
	Url         string
}

type MegaportServices struct {
	Authentication *authentication.Authentication
	Location       *location.Location
	Mcr            *mcr.MCR
	Partner        *partner.Partner
	Port           *port.Port
	Product        *product.Product
	Vxc            *vxc.VXC
}

type AuthBridge struct {
	// Deprecated
	username, password, oneTimePassword string
	// Correct
	accessKey, secretKey string
}

// Wrap http.RoundTripper to append the user-agent header.
type terraformRoundTripper struct {
	T http.RoundTripper
}

func (t *terraformRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", "Go-Megaport-Library/0.1 terraform-provider-megaport/0.3.0")

	return t.T.RoundTrip(req)
}

func ParseAuthConfig(d *schema.ResourceData) *AuthBridge {
	ab := &AuthBridge{}

	// Rip out the auth config.
	if username, ok := d.GetOk("username"); ok {
		ab.username = username.(string)
	}
	if password, ok := d.GetOk("password"); ok {
		ab.password = password.(string)
	}
	if oneTimePassword, ok := d.GetOk("one_time_password"); ok {
		ab.oneTimePassword = oneTimePassword.(string)
	}
	if accessKey, ok := d.GetOk("access_key"); ok {
		ab.accessKey = accessKey.(string)
	}
	if secretKey, ok := d.GetOk("secret_key"); ok {
		ab.secretKey = secretKey.(string)
	}

	return ab
}

func (ab *AuthBridge) Valid() (bool, error) {
	if ab.username != "" && ab.password != "" {
		return true, nil
	} else if ab.accessKey != "" && ab.secretKey != "" {
		return true, nil
	}

	return false, errors.New("invalid combination of credentials passed to provider")
}

func (m *MegaportClient) ConfigureServices(ab *AuthBridge) error {
	logger := NewMegaportLogger()
	cfg := config.Config{
		Log:      logger,
		Endpoint: m.Url,
		Client:   &http.Client{Transport: &terraformRoundTripper{http.DefaultTransport}},
	}

	auth := authentication.New(&cfg)

	var (
		token    string
		loginErr error
	)
	if ab.username != "" {
		token, loginErr = auth.LoginUsername(ab.username, ab.password, ab.oneTimePassword)
	} else {
		token, loginErr = auth.LoginOauth(ab.accessKey, ab.secretKey)
	}

	if loginErr != nil {
		logger.Error("Unable to Authenticate user")
		return loginErr
	}

	cfg.SessionToken = token

	m.MegaportServices = &MegaportServices{
		Authentication: auth,
		Location:       location.New(&cfg),
		Mcr:            mcr.New(&cfg),
		Partner:        partner.New(&cfg),
		Port:           port.New(&cfg),
		Product:        product.New(&cfg),
		Vxc:            vxc.New(&cfg),
	}

	return nil
}
