# regru_dns_aaaa_record

Manages an AAAA record for a DNS zone on Reg.ru. AAAA records map domain names to IPv6 addresses.

## Example Usage

```hcl
# Single AAAA record
resource "regru_dns_aaaa_record" "ipv6_server" {
  zone    = "example.com"
  name    = "ipv6"
  records = ["2001:db8::1"]
}

# Multiple AAAA records for load balancing
resource "regru_dns_aaaa_record" "ipv6_servers" {
  zone    = "example.com"
  name    = "ipv6"
  records = ["2001:db8::1", "2001:db8::2", "2001:db8::3"]
}

# Root domain AAAA record
resource "regru_dns_aaaa_record" "root_ipv6" {
  zone    = "example.com"
  name    = "@"
  records = ["2001:db8::100"]
}
```

## Argument Reference

- `zone` (Required) - The DNS zone (domain) for this record. Changes force resource replacement.
- `name` (Required) - The name for this record. Use `@` for the root domain. Changes force resource replacement.
- `records` (Required) - List of IPv6 addresses for this AAAA record.

## Attributes Reference

- `id` - The resource ID in the format `zone/name`.

## Import

AAAA records can be imported using the format `zone/name`:

```bash
terraform import regru_dns_aaaa_record.ipv6_server example.com/ipv6
terraform import regru_dns_aaaa_record.root_ipv6 example.com/@
```

## Notes

- **IPv6 Only**: This resource is for IPv6 addresses only. Use `regru_dns_a_record` for IPv4 addresses.
- **Multiple Records**: Multiple IPv6 addresses provide simple load balancing and redundancy.
- **Order Independence**: The order of records in the `records` list doesn't affect functionality.
- **IPv6 Format**: Supports standard IPv6 notation including compressed format (e.g., `2001:db8::1`).
- **Root Domain**: Use `@` as the name for root domain records.
