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
	"fmt"
	"net/http"

	"github.com/megaport/megaportgo/config"
	"github.com/megaport/megaportgo/service/authentication"
	"github.com/megaport/megaportgo/service/location"
	"github.com/megaport/megaportgo/service/mcr"
	"github.com/megaport/megaportgo/service/mve"
	"github.com/megaport/megaportgo/service/partner"
	"github.com/megaport/megaportgo/service/port"
	"github.com/megaport/megaportgo/service/product"
	"github.com/megaport/megaportgo/service/vxc"
)

// Set via goreleaser.
var buildVersion = "devel"

func SetBuildVersion(v string) {
	buildVersion = v
}

type MegaportClient struct {
	*MegaportServices
	DeletePorts bool
	Url         string
}

type MegaportServices struct {
	Authentication *authentication.Authentication
	Location       *location.Location
	Mcr            *mcr.MCR
	Mve            *mve.MVE
	Partner        *partner.Partner
	Port           *port.Port
	Product        *product.Product
	Vxc            *vxc.VXC
}

// Wrap http.RoundTripper to append the user-agent header.
type terraformRoundTripper struct {
	T http.RoundTripper
}

func (t *terraformRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", fmt.Sprintf("Go-Megaport-Library/0.1 terraform-provider-megaport/%s", buildVersion))
	return t.T.RoundTrip(req)
}

func (m *MegaportClient) ConfigureServices(accessKey, secretKey string) error {
	logger := NewMegaportLogger()
	cfg := config.Config{
		Log:      logger,
		Endpoint: m.Url,
		Client:   &http.Client{Transport: &terraformRoundTripper{http.DefaultTransport}},
	}

	auth := authentication.New(&cfg)
	token, err := auth.LoginOauth(accessKey, secretKey)
	if err != nil {
		logger.Error("Unable to Authenticate user")
		return err
	}

	cfg.SessionToken = token

	m.MegaportServices = &MegaportServices{
		Authentication: auth,
		Location:       location.New(&cfg),
		Mcr:            mcr.New(&cfg),
		Mve:            mve.New(&cfg),
		Partner:        partner.New(&cfg),
		Port:           port.New(&cfg),
		Product:        product.New(&cfg),
		Vxc:            vxc.New(&cfg),
	}

	return nil
}
