# regru_dns_txt_record

Manages TXT records for a DNS zone on Reg.ru. TXT records store arbitrary text data and are commonly used for domain verification, SPF records, and other purposes.

## Example Usage

```hcl
# SPF record
resource "regru_dns_txt_record" "spf" {
  zone    = "example.com"
  name    = "@"
  records = ["v=spf1 include:_spf.google.com ~all"]
}

# Multiple TXT records
resource "regru_dns_txt_record" "verification" {
  zone    = "example.com"
  name    = "verify"
  records = [
    "google-site-verification=abc123",
    "facebook-domain-verification=xyz789",
    "v=DMARC1; p=reject; rua=mailto:admin@example.com"
  ]
}

# Domain key record
resource "regru_dns_txt_record" "dkim" {
  zone    = "example.com"
  name    = "selector1._domainkey"
  records = ["v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUAA4..."]
}
```

## Argument Reference

- `zone` (Required) - The DNS zone (domain) for this record. Changes force resource replacement.
- `name` (Required) - The name for this record. Use `@` for the root domain. Changes force resource replacement.
- `records` (Required) - List of text values for this TXT record.

## Attributes Reference

- `id` - The resource ID in the format `zone/name`.

## Import

TXT records can be imported using the format `zone/name`:

```bash
terraform import regru_dns_txt_record.spf example.com/@
terraform import regru_dns_txt_record.verification example.com/verify
```

## Notes

- **Multiple Values**: A single TXT record resource can contain multiple text values.
- **Order Independence**: The order of records in the `records` list doesn't affect functionality.
- **Special Characters**: TXT records support special characters and long strings.
- **Common Use Cases**: SPF, DKIM, DMARC, domain verification, and custom metadata.
- **Quotes**: The provider automatically handles proper quoting of TXT record values.
