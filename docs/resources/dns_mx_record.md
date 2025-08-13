# regru_dns_mx_record

Manages MX (Mail Exchange) records for a DNS zone on Reg.ru. MX records specify mail servers responsible for accepting email for a domain.

## Example Usage

```hcl
# Simple MX record
resource "regru_dns_mx_record" "mail" {
  zone = "example.com"
  name = "@"
  
  record {
    priority = 10
    servers  = ["mail.example.com"]
  }
}

# Multiple priorities for redundancy
resource "regru_dns_mx_record" "mail_servers" {
  zone = "example.com"
  name = "@"
  
  record {
    priority = 10
    servers  = ["mail1.example.com", "mail2.example.com"]
  }
  
  record {
    priority = 20
    servers  = ["backup.example.com"]
  }
}

# Subdomain mail
resource "regru_dns_mx_record" "subdomain_mail" {
  zone = "example.com"
  name = "dept"
  
  record {
    priority = 10
    servers  = ["dept-mail.example.com"]
  }
}
```

## Argument Reference

- `zone` (Required) - The DNS zone (domain) for this record. Changes force resource replacement.
- `name` (Required) - The name for this record. Use `@` for the root domain. Changes force resource replacement.
- `record` (Required) - One or more record blocks defining MX configurations.

### record Block

- `priority` (Required) - The priority of this MX record. Lower values have higher priority.
- `servers` (Required) - List of mail server hostnames for this priority level.

## Attributes Reference

- `id` - The resource ID in the format `zone/name`.

## Import

MX records can be imported using the format `zone/name`:

```bash
terraform import regru_dns_mx_record.mail_servers example.com/@
terraform import regru_dns_mx_record.subdomain_mail example.com/dept
```

## Notes

- **Priority-Based**: Lower priority values (e.g., 10) have higher precedence than higher values (e.g., 20).
- **Multiple Servers per Priority**: You can specify multiple servers at the same priority level for load balancing.
- **Surgical Updates**: Only changed servers are updated, not the entire record set, for optimal performance.
- **Order Independence**: The order of servers within a priority level doesn't affect functionality.
- **Consolidated Management**: Multiple priority levels are managed within a single resource for better organization.
