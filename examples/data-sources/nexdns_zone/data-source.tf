# Manage records in a zone that already exists on the account, without
# adopting the zone itself into Terraform state.
data "nexdns_zone" "existing" {
  name = "example.com"
}

resource "nexdns_record" "www" {
  zone_id = data.nexdns_zone.existing.id
  type    = "A"
  name    = "www"
  content = "203.0.113.10"
  ttl     = 300
}

output "existing_nameservers" {
  value = data.nexdns_zone.existing.nameservers
}
