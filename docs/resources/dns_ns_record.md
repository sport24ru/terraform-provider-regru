# regru_dns_ns_record

Manages NS (Name Server) records for a DNS zone on Reg.ru. NS records specify authoritative name servers for a domain or subdomain.

## Example Usage

```hcl
# Simple NS record for subdomain
resource "regru_dns_ns_record" "subdomain" {
  zone = "example.com"
  name = "subdomain"
  
  record {
    priority = 10
    servers  = ["ns1.example.com", "ns2.example.com"]
  }
}

# Multiple NS configurations with different priorities
resource "regru_dns_ns_record" "delegated_zone" {
  zone = "example.com"
  name = "delegated"
  
  record {
    priority = 10
    servers  = ["ns1.delegated.com", "ns2.delegated.com"]
  }
  
  record {
    priority = 20
    servers  = ["ns3.delegated.com"]
  }
}

# Subdomain with multiple name servers
resource "regru_dns_ns_record" "app_subdomain" {
  zone = "example.com"
  name = "app"
  
  record {
    priority = 10
    servers  = ["ns1.app.example.com", "ns2.app.example.com", "ns3.app.example.com"]
  }
}
```

## Argument Reference

- `zone` (Required) - The DNS zone (domain) for this record. Changes force resource replacement.
- `name` (Required) - The name for this record. Cannot be `@` (root domain). Changes force resource replacement.
- `record` (Required) - One or more record blocks defining NS configurations.

### record Block

- `priority` (Required) - The priority of this NS record. Lower values have higher precedence.
- `servers` (Required) - List of name server hostnames for this priority level.

## Attributes Reference

- `id` - The resource ID in the format `zone/name`.

## Import

NS records can be imported using the format `zone/name`:

```bash
terraform import regru_dns_ns_record.subdomain example.com/subdomain
terraform import regru_dns_ns_record.delegated_zone example.com/delegated
```

## Notes

- **No Root Domain**: NS records cannot be created for the root domain (`@`) as these are managed by Reg.ru.
- **Priority Support**: Lower priority values (e.g., 10) have higher precedence than higher values (e.g., 20).
- **Multiple Servers per Priority**: You can specify multiple name servers at the same priority level for redundancy.
- **Surgical Updates**: Only changed servers are updated, not the entire record set, for optimal performance.
- **Order Independence**: The order of servers within a priority level doesn't affect functionality.
- **Zone Delegation**: Commonly used for delegating subdomains to different name servers.
- **Consolidated Management**: Multiple priority levels are managed within a single resource for better organization.
