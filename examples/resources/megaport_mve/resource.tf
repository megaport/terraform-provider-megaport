data "megaport_mve_images" "aruba" {
  vendor_filter = "Aruba"
  id_filter     = 23
}

resource "megaport_mve" "mve" {
  product_name         = "Megaport MVE Example"
  location_id          = 6
  contract_term_months = 1

  vnics = [
    {
      description = "Data Plane"
    },
    {
      description = "Control Plane"
    },
    {
      description = "Management Plane"
    }
  ]

  vendor_config = {
    vendor       = "aruba"
    product_size = "MEDIUM"
    image_id     = data.megaport_mve_images.aruba.mve_images.0.id
    account_name = "Aruba Test Account"
    account_key  = "12345678"
    system_tag   = "Preconfiguration-aruba-test-1"
  }
}
