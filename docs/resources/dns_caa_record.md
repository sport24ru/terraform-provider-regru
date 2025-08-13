# regru_dns_caa_record

Manages CAA (Certificate Authority Authorization) records for a DNS zone on Reg.ru. CAA records specify which Certificate Authorities (CAs) are allowed to issue certificates for a domain.

## Example Usage

```hcl
# Basic CAA record allowing Let's Encrypt
resource "regru_dns_caa_record" "ssl_certs" {
  zone = "example.com"
  name = "@"
  
  record {
    flag  = 0
    tag   = "issue"
    value = "letsencrypt.org"
  }
}

# Comprehensive CAA policy with multiple rules
resource "regru_dns_caa_record" "comprehensive_policy" {
  zone = "example.com"
  name = "@"
  
  record {
    flag  = 0
    tag   = "issue"
    value = "digicert.com"
  }
  
  record {
    flag  = 0
    tag   = "issuewild"
    value = "letsencrypt.org"
  }
  
  record {
    flag  = 128
    tag   = "iodef"
    value = "mailto:security@example.com"
  }
}

# Subdomain with restrictive CAA policy
resource "regru_dns_caa_record" "restrictive_subdomain" {
  zone = "example.com"
  name = "secure"
  
  record {
    flag  = 0
    tag   = "issue"
    value = "digicert.com"
  }
  
  record {
    flag  = 128
    tag   = "issue"
    value = ";"  # Deny all other CAs
  }
}
```

## Argument Reference

- `zone` (Required) - The DNS zone (domain) for this record. Changes force resource replacement.
- `name` (Required) - The name for this record. Use `@` for the root domain. Changes force resource replacement.
- `record` (Required) - One or more record blocks defining CAA policies.

### record Block

- `flag` (Required) - The critical flag. 0 = non-critical, 128 = critical.
- `tag` (Required) - The CAA tag. Common values: `issue`, `issuewild`, `iodef`.
- `value` (Required) - The CAA value. For `issue`/`issuewild`: CA domain name. For `iodef`: email or URL.

## Attributes Reference

- `id` - The resource ID in the format `zone/name`.

## Import

CAA records can be imported using the format `zone/name`:

```bash
terraform import regru_dns_caa_record.ssl_certs example.com/@
terraform import regru_dns_caa_record.restrictive_subdomain example.com/secure
```

## Notes

- **Certificate Control**: CAA records control which CAs can issue SSL/TLS certificates for your domain.
- **Flag Values**: 
  - `0` = Non-critical (CAs may ignore if they don't support the tag)
  - `128` = Critical (CAs must understand and respect the tag)
- **Common Tags**:
  - `issue`: Specifies which CAs can issue certificates
  - `issuewild`: Specifies which CAs can issue wildcard certificates
  - `iodef`: Specifies how to report policy violations
- **Value Examples**:
  - CA domains: `letsencrypt.org`, `digicert.com`, `globalsign.com`
  - Deny all: `;` (semicolon)
  - IODEF: `mailto:admin@example.com` or `https://example.com/report`
- **Policy Enforcement**: CAs check CAA records before issuing certificates.
- **Multiple Policies**: You can specify multiple CAA records for different policies.
- **Subdomain Policies**: CAA records can be set at subdomain level for granular control.
- **Order Independence**: The order of records doesn't affect functionality.
- **Security Best Practice**: Use CAA records to prevent unauthorized certificate issuance.
