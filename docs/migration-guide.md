# Migration Guide: v0.x to v1.0

This guide helps you migrate from the previous version that used a single `regru_dns_record` resource to v1.0 which provides dedicated resources for each DNS record type.

## What Changed

### v0.x (Previous Version)
```hcl
# Generic resource for all record types
resource "regru_dns_record" "a-record" {
    name     = "@"
    record   = "1.1.1.1"
    type     = "A"
    zone     = "example.com"
}

resource "regru_dns_record" "txt-record" {
    name     = "@"
    record   = "google-site-verification=foo-bar"
    type     = "TXT"
    zone     = "example.com"
}

resource "regru_dns_record" "cname-record" {
    name     = "subdomain"
    record   = "example.com."
    type     = "CNAME"
    zone     = "example.com"
}
```

### v1.0 (Current Version)
```hcl
# Dedicated resources for each record type
resource "regru_dns_a_record" "web_a" {
  zone    = "example.com"
  name    = "www"
  records = ["192.168.1.100", "192.168.1.101"]
}

resource "regru_dns_mx_record" "mail_mx" {
  zone = "example.com"
  name = "@"
  
  record {
    priority = 10
    servers  = ["mail.example.com"]
  }
}
```

## Benefits of the New Structure

1. **Type Safety**: Each record type has a schema optimized for its specific needs
2. **Better Validation**: Field validation specific to each DNS record type
3. **Advanced Features**: Complex records support multiple configurations and priorities
4. **Clearer Documentation**: Dedicated documentation for each record type
5. **Better IDE Support**: Improved autocomplete and validation in IDEs

## Migration Steps

### Step 1: Backup Your Current State

```bash
# Backup your current Terraform state
cp terraform.tfstate terraform.tfstate.backup

# Export current resources for reference
terraform state list > current_resources.txt
terraform show > current_config.txt
```

### Step 2: Update Provider Configuration

Update your `required_providers` block:

```hcl
terraform {
  required_providers {
    regru = {
      source  = "sport24ru/regru"
      version = "~> 1.0"  # Update to v1.0
    }
  }
}
```

### Step 3: Replace Resources in Configuration

Replace each `regru_dns_record` with the appropriate dedicated resource:

#### A Records
```hcl
# Before
resource "regru_dns_record" "a-record" {
  name   = "@"
  record = "1.1.1.1"
  type   = "A"
  zone   = "example.com"
}

# After
resource "regru_dns_a_record" "web" {
  zone    = "example.com"
  name    = "@"
  records = ["1.1.1.1"]
}
```

#### AAAA Records
```hcl
# Before
resource "regru_dns_record" "aaaa-record" {
  name   = "ipv6"
  record = "2001:db8::1"
  type   = "AAAA"
  zone   = "example.com"
}

# After
resource "regru_dns_aaaa_record" "ipv6" {
  zone    = "example.com"
  name    = "ipv6"
  records = ["2001:db8::1"]
}
```

#### CNAME Records
```hcl
# Before
resource "regru_dns_record" "cname-record" {
  name   = "subdomain"
  record = "example.com."
  type   = "CNAME"
  zone   = "example.com"
}

# After
resource "regru_dns_cname_record" "www" {
  zone  = "example.com"
  name  = "www"
  cname = "web.example.com"  # Note: single value, not a list
}
```

#### MX Records
```hcl
# Before
resource "regru_dns_record" "mx-record" {
  name     = "@"
  record   = "mail1.example.com"
  type     = "MX"
  zone     = "example.com"
  priority = 10
}

# After - Multiple priorities in one resource
resource "regru_dns_mx_record" "mail" {
  zone = "example.com"
  name = "@"
  
  record {
    priority = 10
    servers  = ["mail1.example.com"]
  }
  
  record {
    priority = 20
    servers  = ["mail2.example.com"]
  }
}
```

#### NS Records
```hcl
# Before
resource "regru_dns_record" "ns-record" {
  name     = "subdomain"
  record   = "ns1.example.com"
  type     = "NS"
  zone     = "example.com"
  priority = 10
}

# After
resource "regru_dns_ns_record" "subdomain_ns" {
  zone = "example.com"
  name = "subdomain"
  
  record {
    priority = 10
    servers  = ["ns1.example.com", "ns2.example.com"]
  }
}
```

#### TXT Records
```hcl
# Before
resource "regru_dns_record" "txt-record" {
  name   = "@"
  record = "google-site-verification=foo-bar"
  type   = "TXT"
  zone   = "example.com"
}

# After
resource "regru_dns_txt_record" "verification" {
  zone    = "example.com"
  name    = "@"
  records = ["google-site-verification=foo-bar"]
}
```

#### SRV Records
```hcl
# Before
resource "regru_dns_record" "srv-record" {
  name     = "_xmpp-server._tcp"
  record   = "xmpp.example.com"
  type     = "SRV"
  zone     = "example.com"
  priority = 10
}

# After
resource "regru_dns_srv_record" "xmpp" {
  zone = "example.com"
  name = "_xmpp-server._tcp"
  
  record {
    priority = 10
    weight   = 5
    port     = 5269
    targets  = ["xmpp.example.com"]
  }
}
```

#### CAA Records
```hcl
# Before
resource "regru_dns_record" "caa-record" {
  name   = "@"
  record = "0 issue letsencrypt.org"
  type   = "CAA"
  zone   = "example.com"
}

# After
resource "regru_dns_caa_record" "ssl_ca" {
  zone = "example.com"
  name = "@"
  
  record {
    flag  = 0
    tag   = "issue"
    value = "letsencrypt.org"
  }
}
```

### Step 4: Remove Old Resources from State

```bash
# List all old regru_dns_record resources
terraform state list | grep regru_dns_record

# Remove each old resource from state (they will remain in DNS)
terraform state rm regru_dns_record.web
terraform state rm regru_dns_record.mail_primary
# ... repeat for all old resources
```

### Step 5: Import Resources to New Structure

```bash
# Import each resource using the new dedicated resource types
terraform import regru_dns_a_record.web example.com/www
terraform import regru_dns_mx_record.mail example.com/@
# ... repeat for all resources
```

### Step 6: Verify and Apply

```bash
# Initialize with new provider version
terraform init -upgrade

# Verify the plan shows no changes
terraform plan

# If plan is clean, you're done!
# If there are differences, review and adjust configuration
```

## Common Migration Patterns

### Consolidating Multiple Records

If you had multiple `regru_dns_record` resources for the same name but different priorities (common with MX records), consolidate them into a single resource with multiple `record` blocks:

```hcl
# Before (multiple resources)
resource "regru_dns_record" "mx_10" { ... priority = 10 }
resource "regru_dns_record" "mx_20" { ... priority = 20 }

# After (single resource)
resource "regru_dns_mx_record" "mail" {
  record { priority = 10 ... }
  record { priority = 20 ... }
}
```

### Field Name Changes

| Old Field | New Field | Notes |
|-----------|-----------|-------|
| `record` | `records` | For A, AAAA, TXT |
| `record` | `cname` | For CNAME (single value) |
| `record` | `servers` | In MX/NS record blocks |
| `record` | `targets` | In SRV record blocks |
| `record` | `value` | In CAA record blocks |
| `type` | N/A | Record type now determined by resource type |
| `priority` | N/A | Priority now handled within record blocks for complex types (MX, NS, SRV) |

## Troubleshooting

### Import Fails
- Verify the record exists in DNS: `dig example.com`
- Check the import format: `zone/name` (e.g., `example.com/www`)
- For root domain records, use `@`: `example.com/@`

### Plan Shows Unexpected Changes
- Check field mappings (especially `content` → `records`/`cname`/etc.)
- Verify complex records use `record` blocks correctly
- Ensure CNAME uses single `cname` field, not `records` list

### State Issues
- Keep your backup: `terraform.tfstate.backup`
- You can restore: `cp terraform.tfstate.backup terraform.tfstate`
- Re-run migration steps carefully

## Getting Help

If you encounter issues during migration:

1. Check the [resource documentation](resources/) for the specific record type
2. Verify your Reg.ru API credentials are working
3. Test with a simple A record first before migrating complex records
4. Keep backups of your state and configuration files
