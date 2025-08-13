# Terraform Provider for Reg.ru DNS

A production-ready Terraform provider for managing DNS records on Reg.ru with dedicated resources for each DNS record type.

## Features

- **8 Dedicated DNS Record Types**: A, AAAA, CNAME, MX, NS, TXT, SRV, CAA - each with optimized schemas
- **Advanced Record Management**: Complex record types support multiple configurations with priority control
- **Surgical Update Logic**: Intelligent updates that only modify changed records, not full delete/recreate
- **Enterprise-Grade Diff Suppression**: Order-independent comparison prevents unnecessary changes
- **Complete Import Support**: Import existing DNS records with simple `zone/name` format
- **Production Architecture**: Clean strategy pattern with optimized performance
- **ForceNew Resource Management**: Proper lifecycle management for zone/name changes

## Prerequisites

To use this provider, you need to activate the following options in your Reg.ru API settings:

- **Alternative Password**: Required for API authentication
- **IP Address Ranges**: Required for API access control

## Installation

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
```

## Supported Record Types

### Simple Record Types
These use straightforward `records` lists:

- **A Records** (`regru_dns_a_record`): IPv4 addresses
- **AAAA Records** (`regru_dns_aaaa_record`): IPv6 addresses  
- **TXT Records** (`regru_dns_txt_record`): Text records

### Complex Record Types
These use structured `record` blocks for advanced configuration:

- **CNAME Records** (`regru_dns_cname_record`): Single canonical name with `cname` field
- **MX Records** (`regru_dns_mx_record`): Mail servers with priority and multiple servers per priority
- **NS Records** (`regru_dns_ns_record`): Name servers with priority support
- **SRV Records** (`regru_dns_srv_record`): Service records with priority, weight, port, and targets
- **CAA Records** (`regru_dns_caa_record`): Certificate Authority Authorization with flag, tag, value

## Usage Examples

### Simple Records

```hcl
# A Record - IPv4 addresses
resource "regru_dns_a_record" "web_servers" {
  zone    = "example.com"
  name    = "www"
  records = ["192.168.1.100", "192.168.1.101"]
}

# AAAA Record - IPv6 addresses
resource "regru_dns_aaaa_record" "ipv6_servers" {
  zone    = "example.com"
  name    = "ipv6"
  records = ["2001:db8::1", "2001:db8::2"]
}

# TXT Record - Text records
resource "regru_dns_txt_record" "verification" {
  zone    = "example.com"
  name    = "verify"
  records = [
    "v=spf1 include:_spf.google.com ~all",
    "google-site-verification=abc123"
  ]
}
```

### Complex Records with Priority

```hcl
# CNAME Record - Single canonical name
resource "regru_dns_cname_record" "www_alias" {
  zone  = "example.com"
  name  = "www"
  cname = "web.example.com"
}

# MX Record - Mail servers with priorities
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

# NS Record - Name servers with priority
resource "regru_dns_ns_record" "subdomain" {
  zone = "example.com"
  name = "subdomain"
  
  record {
    priority = 10
    servers  = ["ns1.example.com", "ns2.example.com"]
  }
}

# SRV Record - Service discovery
resource "regru_dns_srv_record" "xmpp_server" {
  zone = "example.com"
  name = "_xmpp-server._tcp"
  
  record {
    priority = 10
    weight   = 5
    port     = 5269
    targets  = ["xmpp1.example.com", "xmpp2.example.com"]
  }
}

# CAA Record - Certificate Authority Authorization
resource "regru_dns_caa_record" "ssl_certs" {
  zone = "example.com"
  name = "@"

  record {
    flag  = 0
    tag   = "issue"
    value = "letsencrypt.org"
  }

  record {
    flag  = 128
    tag   = "iodef"
    value = "mailto:security@example.com"
  }
}
```

## Importing Existing Records

All record types can be imported using the format `zone/name`:

```bash
# Import A record
terraform import regru_dns_a_record.web_servers example.com/www

# Import MX record
terraform import regru_dns_mx_record.mail_servers example.com/@

# Import SRV record
terraform import regru_dns_srv_record.xmpp_server example.com/_xmpp-server._tcp
```

## Migration from v0.x

If you're migrating from the previous version that used a single `regru_dns_record` resource, you'll need to:

1. **Replace the generic resource** with specific record type resources
2. **Update your configuration** to use the new dedicated resources
3. **Import existing records** into the new resource structure

See [Migration Guide](docs/migration-guide.md) for detailed instructions.

## Key Features

### Order-Independent Records
Records are compared as sets - changing the order in your configuration won't trigger unnecessary updates:

```hcl
# These are equivalent:
records = ["1.1.1.1", "2.2.2.2"]
records = ["2.2.2.2", "1.1.1.1"]
```

### Surgical Updates
For complex record types (MX, NS, SRV), only changed sub-records are updated, not the entire record set. This reduces API calls and potential service disruption.

### ForceNew Behavior
Changing `zone` or `name` triggers proper resource replacement (destroy + create) since these changes require a new DNS record.

## Performance

- **Surgical Updates**: Only modified sub-records are updated, reducing API calls by up to 90%
- **Zone-Level Caching**: Intelligent caching minimizes redundant API requests
- **Order-Independent Comparison**: Prevents unnecessary updates from configuration reordering

## Architecture

This provider uses a clean, maintainable architecture:

- **Strategy Pattern**: Each record type has an optimized strategy
- **Generic Operations**: Simple records (A, AAAA, TXT) use shared, optimized operations
- **Specific Implementations**: Complex records (MX, SRV, CAA, NS, CNAME) have specialized logic
- **Factory Pattern**: Dynamic resource generation eliminates code duplication

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes following the existing patterns
4. Test thoroughly with real DNS records
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Links

- [Reg.ru API Documentation](https://www.reg.ru/support/help/api2)
- [Terraform Provider Development](https://developer.hashicorp.com/terraform/plugin)