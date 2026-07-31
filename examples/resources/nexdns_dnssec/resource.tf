resource "nexdns_zone" "example" {
  name = "example.com"
}

resource "nexdns_dnssec" "example" {
  zone_id = nexdns_zone.example.id
}

# Publish these at the domain's registrar to complete the chain of trust.
# Nothing in Terraform can do that step for you.
output "ds_records" {
  value = nexdns_dnssec.example.ds_records
}
