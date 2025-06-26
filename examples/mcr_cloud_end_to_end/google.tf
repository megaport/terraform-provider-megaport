resource "google_compute_network" "vpc_network_1" {
  name                    = var.google_vpc_1_name
  auto_create_subnetworks = false
  routing_mode            = "GLOBAL"
}

resource "google_compute_subnetwork" "subnet_1" {
  name          = var.google_subnet_1_name
  ip_cidr_range = var.google_vpc_1_subnet_1
  network       = google_compute_network.vpc_network_1.id
}

resource "google_compute_router" "cloud_router_1" {
  name    = var.google_cloud_router_1_name
  network = google_compute_network.vpc_network_1.id
  bgp {
    asn = 16550
  }
}

resource "google_compute_interconnect_attachment" "vlan_attach_1" {
  name                     = var.google_interconnect_attachment_1_name
  router                   = google_compute_router.cloud_router_1.id
  type                     = "PARTNER"
  region                   = var.google_region_1_name
  admin_enabled            = true
  edge_availability_domain = "AVAILABILITY_DOMAIN_1" // single attachment
}

// Gatus VM Instance

resource "google_compute_firewall" "allow_ports" {
  name    = "allow-gatus-instance-ports"
  network = google_compute_network.vpc_network_1.id

  allow {
    protocol = "tcp"
    ports    = ["22", "80"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["gatus"]
}

resource "google_compute_firewall" "allow_rfc1918" {
  name    = "allow-rfc1918"
  network = google_compute_network.vpc_network_1.id

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  source_ranges = [
    "10.0.0.0/8",
    "172.16.0.0/12",
    "192.168.0.0/16"
  ]
  target_tags = ["gatus"]
}

resource "google_compute_instance" "gatus_instance" {
  name         = "gatus-instance"
  machine_type = "e2-micro"
  zone         = var.google_instance_1_zone

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
    }
  }

  metadata = {
    ssh-keys = "${var.ssh_user}:${file(var.public_key_path)}"
  }

  network_interface {
    subnetwork = google_compute_subnetwork.subnet_1.id

    access_config {}
    network_ip = var.gatus_private_ips["google"]
  }

  tags = ["gatus"]

  metadata_startup_script = templatefile("${path.module}/templates/gatus.tpl", {
    name     = var.google_instance_1_name
    cloud    = "Google"
    interval = var.gatus_interval
    inter    = "${var.gatus_private_ips["aws"]},${var.gatus_private_ips["azure"]}"
    password = var.instance_password
  })
}
