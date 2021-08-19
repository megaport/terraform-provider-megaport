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

func (m *MegaportClient) ConfigureServices(username string, password string, oneTimePassword string) error {
	logger := NewMegaportLogger()
	cfg := config.Config{
		Log:      logger,
		Endpoint: m.Url,
	}

	auth := authentication.New(&cfg, username, password, oneTimePassword)
	token, loginErr := auth.Login()

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
