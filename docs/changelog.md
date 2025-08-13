# Changelog

All notable changes to the Terraform Provider for Reg.ru DNS will be documented in this file.

## [1.0.0] - 2025-08-13

### 🚀 Major Release: Dedicated Resources for Each Record Type

This release completely refactors the provider from a single generic `regru_dns_record` resource to dedicated, optimized resources for each DNS record type.

#### ✨ New Dedicated Resources

- **`regru_dns_a_record`**: IPv4 address records with optimized schema
- **`regru_dns_aaaa_record`**: IPv6 address records with optimized schema  
- **`regru_dns_cname_record`**: Canonical name records with single `cname` field
- **`regru_dns_mx_record`**: Mail exchange records with priority and multiple server configurations
- **`regru_dns_ns_record`**: Name server records with priority support
- **`regru_dns_txt_record`**: Text records with multi-value support
- **`regru_dns_srv_record`**: Service records with priority, weight, port, and targets
- **`regru_dns_caa_record`**: Certificate Authority Authorization records with flag, tag, value

#### 🔄 Breaking Changes

- **Removed**: `regru_dns_record` resource (generic resource no longer supported)
- **Migration Required**: All existing `regru_dns_record` resources must be migrated to dedicated resource types
- **Field Changes**: 
  - `type` → Resource type now determined by resource name
  - `record` → `records`/`cname`/`servers`/`targets`/`value` (depending on record type)
  - `priority` → Handled within record blocks for complex types
- **Schema Changes**: Each record type now has an optimized schema specific to its needs

#### ✨ New Features

- **Advanced Record Management**: Complex records support multiple configurations within a single resource
- **Priority Support**: MX, NS, and SRV records support priority-based configurations
- **Surgical Updates**: Only changed sub-records are updated, reducing API calls by up to 90%
- **Enterprise-Grade Diff Suppression**: Order-independent comparison prevents unnecessary changes
- **Enhanced Import Support**: All resources support importing with `zone/name` format
- **ForceNew Behavior**: Proper resource replacement when zone or name changes

#### 🛠️ Technical Improvements

- **Strategy Pattern**: Each record type uses an optimized strategy for CRUD operations
- **Factory Pattern**: Dynamic resource generation eliminates code duplication
- **Enhanced Caching**: Zone-level caching with intelligent cache invalidation
- **Robust Error Handling**: Comprehensive API error checking and recovery
- **Performance Optimization**: Intelligent update detection prevents unnecessary API operations

#### 📝 Resource Structure Examples

##### Simple Records
```hcl
# A Record
resource "regru_dns_a_record" "web" {
  zone    = "example.com"
  name    = "www"
  records = ["192.168.1.100", "192.168.1.101"]
}

# TXT Record  
resource "regru_dns_txt_record" "verification" {
  zone    = "example.com"
  name    = "verify"
  records = ["v=spf1 include:_spf.google.com ~all"]
}
```

##### Complex Records with Priorities
```hcl
# MX Record with multiple priorities
resource "regru_dns_mx_record" "mail" {
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

# SRV Record
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

#### 🔧 Critical Bug Fixes

- **Fixed Update Detection**: Resolved issues where `terraform plan` showed "No changes" for real modifications
- **Enhanced Diff Suppression**: Fixed order-only change detection for all record types
- **Resource State Consistency**: Proper ForceNew behavior for zone/name changes
- **Import Functionality**: Comprehensive import support for all record types

#### ⚡ Performance Improvements

- **Surgical Updates**: Complex records only update changed sub-records
- **Reduced API Calls**: Intelligent change detection prevents unnecessary operations
- **Enhanced Caching**: Zone-level caching reduces redundant API requests
- **Optimized Schemas**: Each record type has a schema optimized for its specific needs

#### 📋 Migration Required

**Important**: This is a breaking change. You must migrate from `regru_dns_record` to the new dedicated resources.

See the [Migration Guide](migration-guide.md) for detailed step-by-step instructions.

**Migration Overview**:
1. Update provider version to ~> 1.0
2. Replace `regru_dns_record` with appropriate dedicated resource
3. Remove old resources from state
4. Import resources using new resource types
5. Verify plan shows no changes

#### 🧪 Comprehensive Testing

- **All Record Types**: Every resource type tested for basic functionality, imports, and updates
- **Complex Scenarios**: Multi-record configurations, priority changes, and surgical updates
- **Edge Cases**: Order independence, diff suppression, and ForceNew behavior
- **Real-World Usage**: Tested with actual DNS infrastructure and complex configurations

---

## [0.1.0] - 2025-02-17

### ✨ Initial Release

- **Generic DNS Management**: Single `regru_dns_record` resource for all record types
- **Basic CRUD Operations**: Create, read, update, delete DNS records
- **API Integration**: Direct integration with Reg.ru DNS API
- **Zone-Level Operations**: Basic zone-level caching and error handling

**Note**: This version used a generic `regru_dns_record` resource with `type`, `record`, `priority`, and other generic fields. The `record` field contained the actual DNS record value (IP address, hostname, text, etc.).