# regru_dns_a_record

Manages an A record for a DNS zone on Reg.ru. A records map domain names to IPv4 addresses.

## Example Usage

```hcl
# Single A record
resource "regru_dns_a_record" "web_server" {
  zone    = "example.com"
  name    = "www"
  records = ["192.168.1.100"]
}

# Multiple A records for load balancing
resource "regru_dns_a_record" "web_servers" {
  zone    = "example.com"
  name    = "www"
  records = ["192.168.1.100", "192.168.1.101", "192.168.1.102"]
}

# Root domain A record
resource "regru_dns_a_record" "root" {
  zone    = "example.com"
  name    = "@"
  records = ["192.168.1.100"]
}
```

## Argument Reference

- `zone` (Required) - The DNS zone (domain) for this record. Changes force resource replacement.
- `name` (Required) - The name for this record. Use `@` for the root domain. Changes force resource replacement.
- `records` (Required) - List of IPv4 addresses for this A record.

## Attributes Reference

- `id` - The resource ID in the format `zone/name`.

## Import

A records can be imported using the format `zone/name`:

```bash
terraform import regru_dns_a_record.web_server example.com/www
terraform import regru_dns_a_record.root example.com/@
```

## Notes

- **Order Independence**: The order of records in the `records` list doesn't matter. Terraform will not detect changes if only the order changes.
- **IPv4 Only**: This resource is for IPv4 addresses only. Use `regru_dns_aaaa_record` for IPv6 addresses.
- **Multiple Records**: Multiple IPv4 addresses provide simple load balancing and redundancy.
- **Root Domain**: Use `@` as the name for root domain records.
