terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = "1.3.8"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "5.100.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "4.33.0"
    }
    google = {
      source  = "hashicorp/google"
      version = "6.39.0"
    }
  }
}
provider "megaport" {
  access_key            = "<api_key>"
  secret_key            = "<api_secret>"
  accept_purchase_terms = true
  environment           = "production"
}
provider "aws" {
  region     = var.aws_region_1
  access_key = "<access_key>"
  secret_key = "<secret_key>"
}
provider "azurerm" {
  features {}
  subscription_id = "<subscription_id>"
}
provider "google" {
  project = "<project_name>"
  region  = "<region>"
}