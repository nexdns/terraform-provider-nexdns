terraform {
  required_version = ">= 1.0"

  required_providers {
    nexdns = {
      source  = "nexdns/nexdns"
      version = "~> 1.0"
    }
  }
}

# The token may also come from the NEXDNS_API_TOKEN environment variable, which
# keeps it out of the configuration entirely.
provider "nexdns" {
  api_token = var.nexdns_api_token
}

variable "nexdns_api_token" {
  description = "NexDNS API token with the zones.read/zones.write and records.read/records.write scopes."
  type        = string
  sensitive   = true
}
