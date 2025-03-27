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

data "megaport_mve_images" "aviatrix" {
  vendor_filter = "Aviatrix"
  id_filter     = 70
}

# Sample Cloud Init Config for Aviatrix Edge
# #cloud-config
# write_files:
#   - path: /etc/bootstrap.cfg
#     content: |
#       controller_ip=controller.aviatrixsystems.net
#       account_name=megaport_test
#       admin_email=admin@megaport.com
#       vpc_id=megaport_edge
#       region=megaport_region
#       activation_key=AVX-EDGE-1234-5678-wmabc
#       customer_id=MP56789
#       site_id=MPORT-MVE-01

resource "megaport_mve" "aviatrix_edge" {
  product_name         = "Aviatrix-Edge"
  location_id          = 123
  contract_term_months = 12

  vendor_config = {
    vendor       = "aviatrix"
    image_id     = data.megaport_mve_images.aviatrix.mve_images.0.id
    product_size = "SMALL"
    mve_label    = "avx-edge-01"
    cloud_init   = "I2Nsb3VkLWNvbmZpZwp3cml0ZV9maWxlczoKICAtIHBhdGg6IC9ldGMvYm9vdHN0cmFwLmNmZwogICAgY29udGVudDogfAogICAgICBjb250cm9sbGVyX2lwPWNvbnRyb2xsZXIuYXZpYXRyaXhzeXN0ZW1zLm5ldAogICAgICBhY2NvdW50X25hbWU9bWVnYXBvcnRfdGVzdAogICAgICBhZG1pbl9lbWFpbD1hZG1pbkBtZWdhcG9ydC5jb20KICAgICAgdnBjX2lkPW1lZ2Fwb3J0X2VkZ2UKICAgICAgcmVnaW9uPW1lZ2Fwb3J0X3JlZ2lvbgogICAgICBhY3RpdmF0aW9uX2tleT1BVlgtRURHRS0xMjM0LTU2Nzgtd21hYmMKICAgICAgY3VzdG9tZXJfaWQ9TVA1Njc4OQogICAgICBzaXRlX2lkPU1QT1JULU1WRS0wMQ==" # Base64 Encoded Cloud Init for Aviatrix Edge
  }
}