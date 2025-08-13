# regru_dns_srv_record

Manages SRV (Service) records for a DNS zone on Reg.ru. SRV records specify the location of services for a domain, commonly used for service discovery.

## Example Usage

```hcl
# Basic SRV record for XMPP service
resource "regru_dns_srv_record" "xmpp_server" {
  zone = "example.com"
  name = "_xmpp-server._tcp"
  
  record {
    priority = 10
    weight   = 5
    port     = 5269
    targets  = ["xmpp.example.com"]
  }
}

# Multiple SRV records with different priorities
resource "regru_dns_srv_record" "sip_service" {
  zone = "example.com"
  name = "_sip._tcp"
  
  record {
    priority = 10
    weight   = 60
    port     = 5060
    targets  = ["sip1.example.com", "sip2.example.com"]
  }
  
  record {
    priority = 20
    weight   = 40
    port     = 5060
    targets  = ["sip-backup.example.com"]
  }
}

# LDAP service discovery
resource "regru_dns_srv_record" "ldap_service" {
  zone = "example.com"
  name = "_ldap._tcp"
  
  record {
    priority = 10
    weight   = 100
    port     = 389
    targets  = ["ldap1.example.com", "ldap2.example.com"]
  }
}
```

## Argument Reference

- `zone` (Required) - The DNS zone (domain) for this record. Changes force resource replacement.
- `name` (Required) - The service name in the format `_service._protocol`. Changes force resource replacement.
- `record` (Required) - One or more record blocks defining SRV configurations.

### record Block

- `priority` (Required) - The priority of this SRV record. Lower values have higher precedence.
- `weight` (Required) - The weight for load balancing between records with the same priority.
- `port` (Required) - The port number on which the service is available.
- `targets` (Required) - List of hostnames providing this service.

## Attributes Reference

- `id` - The resource ID in the format `zone/name`.

## Import

SRV records can be imported using the format `zone/name`:

```bash
terraform import regru_dns_srv_record.xmpp_server example.com/_xmpp-server._tcp
terraform import regru_dns_srv_record.sip_service example.com/_sip._tcp
```

## Notes

- **Service Discovery**: SRV records are used for automatic service discovery by applications.
- **Priority System**: Lower priority values (e.g., 10) have higher precedence than higher values (e.g., 20).
- **Weight Load Balancing**: Among records with the same priority, weight determines the proportion of traffic.
- **Protocol Support**: Common protocols include `_tcp`, `_udp`, `_tls`, and `_sctp`.
- **Service Names**: Common services include `_xmpp-server`, `_sip`, `_ldap`, `_kerberos`, `_http`, `_https`.
- **Surgical Updates**: Only changed targets are updated, not the entire record set, for optimal performance.
- **Order Independence**: The order of targets within a priority level doesn't affect functionality.
- **Port Specification**: The port field allows services to run on non-standard ports.
- **Multiple Targets**: Multiple targets at the same priority provide redundancy and load balancing.
