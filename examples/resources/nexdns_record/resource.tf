resource "nexdns_zone" "example" {
  name = "example.com"
}

# Two values in one record set: same name, same type, and therefore the same
# TTL. The set's TTL is whatever was applied last, so keep them equal.
resource "nexdns_record" "apex_a" {
  zone_id = nexdns_zone.example.id
  type    = "A"
  name    = "@"
  content = "203.0.113.10"
  ttl     = 300
}

resource "nexdns_record" "apex_a_second" {
  zone_id = nexdns_zone.example.id
  type    = "A"
  name    = "@"
  content = "203.0.113.11"
  ttl     = 300
}

resource "nexdns_record" "www" {
  zone_id = nexdns_zone.example.id
  type    = "CNAME"
  name    = "www"
  content = "example.com"
  ttl     = 300
}

# The preference goes in priority, not into content.
resource "nexdns_record" "mx" {
  zone_id  = nexdns_zone.example.id
  type     = "MX"
  name     = "@"
  content  = "mail.example.com"
  priority = 10
}

# TXT content is written unquoted; the quoting is applied for you.
resource "nexdns_record" "spf" {
  zone_id = nexdns_zone.example.id
  type    = "TXT"
  name    = "@"
  content = "v=spf1 mx -all"
}

resource "nexdns_record" "sip" {
  zone_id  = nexdns_zone.example.id
  type     = "SRV"
  name     = "_sip._tcp"
  content  = "sip.example.com"
  priority = 10
  weight   = 5
  port     = 5060
}

# Only the issuer goes in content; the flags octet and the property are their
# own attributes.
resource "nexdns_record" "caa" {
  zone_id = nexdns_zone.example.id
  type    = "CAA"
  name    = "@"
  content = "letsencrypt.org"
  flags   = 0
  tag     = "issue"
}
