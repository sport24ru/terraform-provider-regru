# regru_dns_cname_record

Manages a CNAME record for a DNS zone on Reg.ru. CNAME records create an alias from one domain name to another.

## Example Usage

```hcl
# Basic CNAME record
resource "regru_dns_cname_record" "www_alias" {
  zone  = "example.com"
  name  = "www"
  cname = "web.example.com"
}

# Subdomain alias
resource "regru_dns_cname_record" "blog_alias" {
  zone  = "example.com"
  name  = "blog"
  cname = "myblog.wordpress.com"
}
```

## Argument Reference

- `zone` (Required) - The DNS zone (domain) for this record. Changes force resource replacement.
- `name` (Required) - The name for this record. Cannot be `@` (root domain). Changes force resource replacement.
- `cname` (Required) - The canonical name (target) for this CNAME record.

## Attributes Reference

- `id` - The resource ID in the format `zone/name`.

## Import

CNAME records can be imported using the format `zone/name`:

```bash
terraform import regru_dns_cname_record.www_alias example.com/www
```

## Notes

- **Single Target**: CNAME records can only point to one target, hence the `cname` field is a single string, not a list.
- **No Root Domain**: CNAME records cannot be created for the root domain (`@`). Use A records instead.
- **DNS Conflicts**: CNAME records cannot coexist with other record types for the same subdomain.
- **Trailing Dots**: The provider automatically handles trailing dots in CNAME targets.
- **RFC Compliance**: This resource enforces DNS RFC requirements for CNAME records.
