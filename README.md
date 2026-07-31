# NexDNS Terraform Provider

[![Terraform Registry](https://img.shields.io/badge/terraform-registry-7B42BC?logo=terraform)](https://registry.terraform.io/providers/nexdns/nexdns/latest)
[![Latest release](https://img.shields.io/github/v/release/nexdns/terraform-provider-nexdns?sort=semver&label=release)](https://github.com/nexdns/terraform-provider-nexdns/releases)
[![License](https://img.shields.io/github/license/nexdns/terraform-provider-nexdns)](LICENSE)

Manage DNS zones, records and DNSSEC signing on [NexDNS](https://nexdns.tech)
from Terraform. DNS lives in the same plan as the infrastructure it points at, so
a hostname is never left pointing at an address that no longer exists, and the
zone in production is the zone in the repository.

Built for teams that already describe their infrastructure as code: the address a
server gets on creation is the address its `A` record gets in the same apply, and
`terraform destroy` takes the records with it.

## Requirements

- Terraform 1.0 or later
- A NexDNS account on a plan that includes API access, and an API token created
  under [API keys](https://nexdns.tech/settings/api-keys). DNSSEC and secondary
  zones are part of the plan too – the [pricing page](https://nexdns.tech/pricing)
  lists which plans include what
- Go 1.25 or later, to build from source rather than install from the Registry

## Installation

The provider is published on the Terraform Registry. Declare it and run
`terraform init`:

```hcl
terraform {
  required_version = ">= 1.0"

  required_providers {
    nexdns = {
      source  = "nexdns/nexdns"
      version = "~> 1.0"
    }
  }
}
```

## Authentication

Leave the token in the environment and the provider block stays empty:

```shell
export NEXDNS_API_TOKEN="nxd_xxxxxxxxxxxxxxxxxxxx"
terraform plan
```

```hcl
provider "nexdns" {}
```

Or pass it explicitly, when it comes from a variable or a secret manager:

```hcl
provider "nexdns" {
  api_token = var.nexdns_api_token
}

variable "nexdns_api_token" {
  type      = string
  sensitive = true
}
```

| Setting | Attribute | Environment variable | Default |
|---------|-----------|----------------------|---------|
| API token | `api_token` | `NEXDNS_API_TOKEN` | none, required |
| API endpoint | `api_url` | `NEXDNS_API_URL` | `https://api.nexdns.tech/v1` |

The attribute wins over the environment variable when both are set.

Scope the token to what the configuration touches: `zones.read` and `zones.write`
for `nexdns_zone` and `nexdns_dnssec`, `records.read` and `records.write` for
`nexdns_record`.

## Quickstart

A zone, the records in it, and a signed delegation – the whole thing in one
apply:

```hcl
terraform {
  required_version = ">= 1.0"

  required_providers {
    nexdns = {
      source  = "nexdns/nexdns"
      version = "~> 1.0"
    }
  }
}

provider "nexdns" {} # reads NEXDNS_API_TOKEN

resource "nexdns_zone" "example" {
  name = "example.com"
}

resource "nexdns_record" "apex" {
  zone_id = nexdns_zone.example.id
  type    = "A"
  name    = "@"
  content = "203.0.113.10"
  ttl     = 300
}

resource "nexdns_record" "www" {
  zone_id = nexdns_zone.example.id
  type    = "CNAME"
  name    = "www"
  content = "example.com"
  ttl     = 300
}

resource "nexdns_record" "mx" {
  zone_id  = nexdns_zone.example.id
  type     = "MX"
  name     = "@"
  content  = "mail.example.com"
  priority = 10
}

resource "nexdns_dnssec" "example" {
  zone_id = nexdns_zone.example.id
}

# Delegate the domain to these at your registrar, then publish the DS records
# there to complete the DNSSEC chain of trust.
output "nameservers" {
  value = nexdns_zone.example.nameservers
}

output "ds_records" {
  value = nexdns_dnssec.example.ds_records
}
```

## What the provider covers

| Kind | Name | Purpose |
|------|------|---------|
| Resource | `nexdns_zone` | A DNS zone, primary or secondary |
| Resource | `nexdns_record` | One value in a record set |
| Resource | `nexdns_dnssec` | DNSSEC signing for a zone |
| Data source | `nexdns_zone` | Look up a zone that already exists on the account |

Full attribute reference for each:
[Registry documentation](https://registry.terraform.io/providers/nexdns/nexdns/latest/docs),
or [`docs/`](docs) in this repository, which is what the Registry renders.

### nexdns_zone

```hcl
resource "nexdns_zone" "example" {
  name = "example.com"
}
```

`ns_group` selects the nameserver group that serves the zone; omitted, the
account's default group is used. The valid values are the groups available on
your account, listed in the control panel and returned by the API.

**`ns_group` is the only attribute that changes in place.** Moving a zone to
another group keeps the zone, its identifier and all of its records, and
refreshes `nameservers`; update the delegation at your registrar afterwards.
Every other attribute – `name`, `type`, `master_ip` – forces replacement, and
replacing a zone deletes it together with every record in it, including records
Terraform does not manage. Read those plans before approving them.

A secondary zone is transferred from your own primary server rather than edited
here:

```hcl
resource "nexdns_zone" "secondary" {
  name      = "example.net"
  type      = "slave"
  master_ip = "203.0.113.53"
}
```

### nexdns_record

Supported types: `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `SRV`, `CAA`, `PTR`,
`ALIAS`, `DNAME`, `DS`, `TLSA`.

`content` carries the record's primary value and nothing else. Everything else
the type needs has its own attribute:

| Type | `content` | Other attributes |
|------|-----------|------------------|
| `A`, `AAAA` | the address | – |
| `CNAME`, `PTR`, `ALIAS`, `DNAME`, `NS` | the target hostname | – |
| `MX` | the mail host | `priority` |
| `SRV` | the target host | `priority`, `weight`, `port` |
| `TXT` | the text, unquoted | – |
| `CAA` | the issuer | `flags`, `tag` |
| `DS` | the digest | `keytag`, `algorithm`, `digest_type` |
| `TLSA` | the certificate association data | `usage`, `selector`, `matching_type` |

So an `MX` record is `content = "mail.example.com"` with `priority = 10`, never
`content = "10 mail.example.com"`.

The zone's own `NS` records at the apex come from its nameserver group and are
not managed here; `NS` records for a delegated subdomain are.

Given the zone above, a set of values reads well with `for_each`:

```hcl
resource "nexdns_record" "edge" {
  for_each = toset(["203.0.113.10", "203.0.113.11"])

  zone_id = nexdns_zone.example.id
  type    = "A"
  name    = "@"
  content = each.value
  ttl     = 300
}
```

#### A TTL belongs to the record set

Every value sharing a name and a type is one record set, and the TTL is a
property of that set rather than of the single value. Setting `ttl` on one
`nexdns_record` retimes all of them.

Give every record in a set the same `ttl` – the `for_each` above does this
naturally. Two resources at the same name and type with different TTLs will
overwrite each other on every apply, and the plan will never come out empty.

Left unset, an existing set keeps its TTL and a new one gets the platform
default; either way the effective value is read back into state.

#### Ids change with content

A record's id is derived from its content, so editing `content` gives the record
a new id. This is an update, not a replacement: the new id comes back in the same
apply. It matters when you address a record by id – after an edit, the old one is
gone.

### nexdns_dnssec

```hcl
resource "nexdns_dnssec" "example" {
  zone_id = nexdns_zone.example.id
}
```

Creating the resource enables signing, destroying it disables signing. Keys are
held by the platform, so there is nothing to update: the resource has no editable
attribute, and changing `zone_id` replaces it.

Signing is only half of the work. Publish the exported `ds_records` at the
domain's registrar to complete the chain of trust – Terraform cannot do that
step. In the other direction, remove the DS records at the registrar *before*
destroying this resource: a DS pointing at a zone that is no longer signed makes
validating resolvers refuse the whole domain.

### nexdns_zone (data source)

```hcl
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
```

Manages records in a zone without adopting the zone itself. The lookup matches
the stored name exactly, so write the domain the way the platform holds it:
lower-case, no trailing dot, punycode for an internationalised domain.

## Import

All three resources support `terraform import`:

```shell
# Zone – by its identifier
terraform import nexdns_zone.example zK7mPq2R

# Record – by <zone_id>/<record_id>, since a record id is unique inside its zone
terraform import nexdns_record.www zK7mPq2R/4d0f7a1b9c2e5386

# DNSSEC – by the identifier of the zone it signs
terraform import nexdns_dnssec.example zK7mPq2R
```

## Names as the platform stores them

Zone and record names are stored lower-cased, without a trailing dot, and in
punycode for an internationalised name. The provider keeps the spelling you
wrote when it means the same name, so `Example.com`, `example.com.` and a
national-script domain do not turn into permanent drift.

A zone or record deleted outside Terraform drops out of state on the next
refresh, and the following plan recreates it – rather than leaving the
configuration unplannable.

## Development

```shell
go build ./...
go vet ./...
go test ./...
```

To try a local build against real configuration, point Terraform at the compiled
binary instead of the Registry. Install it, then add a `dev_overrides` block to
`~/.terraformrc`:

```shell
go install .
```

```hcl
provider_installation {
  dev_overrides {
    "nexdns/nexdns" = "/home/you/go/bin"
  }

  direct {}
}
```

With an override in place Terraform skips `terraform init` for this provider and
warns on every command, which is the intended signal that you are not running a
released build.

The Registry documentation in [`docs/`](docs) is generated from the schema
descriptions in `internal/provider/` and the examples in [`examples/`](examples).
Edit those, never `docs/` by hand, and regenerate:

```shell
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate
```

CI builds, vets, tests, and fails if the regenerated documentation differs from
what is committed.

## Contributing

Issues and pull requests are welcome at
[github.com/nexdns/terraform-provider-nexdns](https://github.com/nexdns/terraform-provider-nexdns/issues).
Please keep `go vet` and `gofmt` clean, add or update the schema descriptions
when you change an attribute, regenerate `docs/`, and record user-visible changes
in [`CHANGELOG.md`](CHANGELOG.md) under `## [Unreleased]`.

## Related projects

- [NexDNS CLI](https://github.com/nexdns/cli) – zone and record management from
  the shell, plus declarative DNS-as-Code in YAML
- [certbot-dns-nexdns](https://github.com/nexdns/certbot-dns-nexdns) – DNS-01
  challenges for wildcard certificates
- [octodns-nexdns](https://github.com/nexdns/octodns-nexdns) – multi-provider
  DNS-as-Code with OctoDNS

## Links

- **Provider documentation**: [registry.terraform.io/providers/nexdns/nexdns](https://registry.terraform.io/providers/nexdns/nexdns/latest/docs)
- **REST API reference**: [nexdns.tech/docs/api](https://nexdns.tech/docs/api)
- **Website**: [nexdns.tech](https://nexdns.tech)
- **Pricing**: [nexdns.tech/pricing](https://nexdns.tech/pricing)

## License

[Apache-2.0](LICENSE)
