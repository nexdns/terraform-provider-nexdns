# A zone on the account's default nameserver group.
resource "nexdns_zone" "example" {
  name = "example.com"
}

# Delegate the domain to these at your registrar.
output "example_nameservers" {
  value = nexdns_zone.example.nameservers
}

# A zone pinned to a particular nameserver group. The valid values are the
# groups available on your account; changing this later moves the zone in
# place, keeping its identifier and its records.
variable "ns_group" {
  description = "Nameserver group slug, or null for the account default."
  type        = string
  default     = null
}

resource "nexdns_zone" "pinned" {
  name     = "example.net"
  ns_group = var.ns_group
}

# A secondary zone: transferred from your own primary server rather than edited
# here. Requires a plan that includes secondary zones.
resource "nexdns_zone" "secondary" {
  name      = "example.org"
  type      = "slave"
  master_ip = "203.0.113.53"
}
