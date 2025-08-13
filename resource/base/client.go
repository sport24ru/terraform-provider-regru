package base

// CachedClientInterface defines the interface for cached client operations
// This avoids import cycles between strategies and provider packages
type CachedClientInterface interface {
	// Core DNS operations
	AddRecord(recordType, domainName, subdomain, value string, priority *int) ([]byte, error)
	RemoveRecord(domainName, subdomain, recordType, content string, priority *int) ([]byte, error)
	GetRecords(domainName string) ([]byte, error)

	// Specialized SRV operations
	AddSRVRecord(domainName, subdomain, target string, priority, weight, port *int) ([]byte, error)
	RemoveSRVRecord(domainName, subdomain, target string, priority, weight, port *int) ([]byte, error)

	// Specialized CAA operations
	AddCAARecord(domainName, subdomain, value string, flag *int, tag *string) ([]byte, error)
	RemoveCAARecord(domainName, subdomain, value string, flag *int, tag *string) ([]byte, error)

	// Caching operations
	GetRecordsWithCache(domainName string) ([]byte, error)
	InvalidateZoneCache(zone string)
	ClearZoneCache()
}
