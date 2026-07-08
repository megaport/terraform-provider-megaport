provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
  wait_time             = 20 # Minutes to wait for resources to provision (default 10)
}