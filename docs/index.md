# Reg.ru DNS Provider

A production-ready Terraform provider for managing DNS records on Reg.ru with dedicated resources for each DNS record type.

## Overview

This provider enables you to manage DNS records on Reg.ru through Terraform with dedicated resources for each record type. Unlike generic DNS providers, each record type has an optimized schema and specialized logic for the best user experience.

## Quick Start

```hcl
terraform {
  required_providers {
    regru = {
      source  = "sport24ru/regru"
      version = "~> 1.0"
    }
  }
}

provider "regru" {
  username = var.regru_username
  password = var.regru_password
}

# Create an A record
resource "regru_dns_a_record" "web_servers" {
  zone    = "example.com"
  name    = "www"
  records = ["192.168.1.100", "192.168.1.101"]
}
```

## Resources

### DNS Record Resources

- [regru_dns_a_record](resources/dns_a_record.md) - IPv4 address records
- [regru_dns_aaaa_record](resources/dns_aaaa_record.md) - IPv6 address records
- [regru_dns_cname_record](resources/dns_cname_record.md) - Canonical name records
- [regru_dns_mx_record](resources/dns_mx_record.md) - Mail exchange records
- [regru_dns_ns_record](resources/dns_ns_record.md) - Name server records
- [regru_dns_txt_record](resources/dns_txt_record.md) - Text records
- [regru_dns_srv_record](resources/dns_srv_record.md) - Service records
- [regru_dns_caa_record](resources/dns_caa_record.md) - Certificate Authority Authorization records

## Provider Configuration

| Argument | Description | Type | Required |
|----------|-------------|------|----------|
| `username` | Reg.ru username | `string` | Yes |
| `password` | Reg.ru alternative password | `string` | Yes |

**Important**: You must use an "alternative password" from your Reg.ru API settings, not your regular account password.

## Migration from v0.x

If you're upgrading from a previous version that used `regru_dns_record`, see our [Migration Guide](migration-guide.md) for step-by-step instructions.

## Import Support

All resources support importing with the format `zone/name`:

```bash
terraform import regru_dns_a_record.example example.com/www
```

## Features

- **Dedicated Resources**: Each DNS record type has its own optimized resource
- **Advanced Schemas**: Complex records support multiple configurations and priorities
- **Order Independence**: Record lists are compared as sets, ignoring order
- **Surgical Updates**: Only changed sub-records are updated, not entire record sets
- **Complete Import Support**: Import existing DNS infrastructure easily
- **Production Ready**: Enterprise-grade error handling and performance optimization