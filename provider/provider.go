package provider

import (
	"log"
	"sync"
	"time"

	"terraform-provider-regru/client"
	"terraform-provider-regru/resource/resources"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Global cache manager that persists across all resource operations
var (
	globalZoneCache  = NewZoneCache()
	globalCacheMutex sync.RWMutex
)

// ZoneCache provides caching for zone records to prevent multiple API calls
type ZoneCache struct {
	cache map[string]*ZoneCacheEntry
	mutex sync.RWMutex
}

// ZoneCacheEntry represents cached zone data
type ZoneCacheEntry struct {
	Data      []byte
	Timestamp time.Time
	TTL       time.Duration
}

// NewZoneCache creates a new zone cache
func NewZoneCache() *ZoneCache {
	return &ZoneCache{
		cache: make(map[string]*ZoneCacheEntry),
	}
}

// Get retrieves cached zone data if it's still valid
func (zc *ZoneCache) Get(zone string) ([]byte, bool) {
	zc.mutex.RLock()
	defer zc.mutex.RUnlock()

	log.Printf("[DEBUG] ZoneCache.Get called for zone: %s", zone)
	log.Printf("[DEBUG] Current cache contents: %v", zc.cache)

	entry, exists := zc.cache[zone]
	if !exists {
		log.Printf("[DEBUG] ZoneCache.Get: zone %s not found in cache", zone)
		return nil, false
	}

	log.Printf("[DEBUG] ZoneCache.Get: zone %s found in cache, timestamp: %v, TTL: %v", zone, entry.Timestamp, entry.TTL)

	if time.Since(entry.Timestamp) > entry.TTL {
		log.Printf("[DEBUG] ZoneCache.Get: zone %s cache expired, removing", zone)
		delete(zc.cache, zone)
		return nil, false
	}

	log.Printf("[DEBUG] ZoneCache.Get: zone %s cache valid, returning data", zone)
	return entry.Data, true
}

// Set stores zone data in cache
func (zc *ZoneCache) Set(zone string, data []byte) {
	zc.mutex.Lock()
	defer zc.mutex.Unlock()

	log.Printf("[DEBUG] ZoneCache.Set called for zone: %s", zone)
	log.Printf("[DEBUG] ZoneCache.Set: storing data of length %d bytes", len(data))

	zc.cache[zone] = &ZoneCacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       30 * time.Second, // Cache for 30 seconds
	}

	log.Printf("[DEBUG] ZoneCache.Set: zone %s stored in cache", zone)
	log.Printf("[DEBUG] ZoneCache.Set: current cache contents: %v", zc.cache)
}

// Invalidate removes a specific zone from cache
func (zc *ZoneCache) Invalidate(zone string) {
	zc.mutex.Lock()
	defer zc.mutex.Unlock()
	delete(zc.cache, zone)
}

// Clear clears all cached data
func (zc *ZoneCache) Clear() {
	zc.mutex.Lock()
	defer zc.mutex.Unlock()
	zc.cache = make(map[string]*ZoneCacheEntry)
}

// CachedClient wraps the original client with caching capabilities
type CachedClient struct {
	*client.Client
}

// GetRecordsWithCache gets zone records with caching using global cache
func (cc *CachedClient) GetRecordsWithCache(zone string) ([]byte, error) {
	log.Printf("[DEBUG] GetRecordsWithCache called for zone: %s", zone)

	// Try to get from global cache first
	globalCacheMutex.RLock()
	log.Printf("[DEBUG] Acquired global cache read lock for zone: %s", zone)

	if cached, exists := globalZoneCache.Get(zone); exists {
		log.Printf("[DEBUG] GLOBAL CACHE HIT for zone %s, returning cached data", zone)
		globalCacheMutex.RUnlock()
		return cached, nil
	}

	log.Printf("[DEBUG] GLOBAL CACHE MISS for zone %s, cache does not exist", zone)
	globalCacheMutex.RUnlock()

	log.Printf("[DEBUG] Making API call for zone: %s", zone)

	// If not in cache, fetch from API
	data, err := cc.GetRecords(zone)
	if err != nil {
		log.Printf("[DEBUG] API call failed for zone %s: %v", zone, err)
		return nil, err
	}

	log.Printf("[DEBUG] API call successful for zone %s, storing in global cache", zone)

	// Store in global cache
	globalCacheMutex.Lock()
	log.Printf("[DEBUG] Acquired global cache write lock for zone: %s", zone)
	globalZoneCache.Set(zone, data)
	log.Printf("[DEBUG] GLOBAL CACHE SET for zone %s", zone)
	globalCacheMutex.Unlock()

	log.Printf("[DEBUG] Returning data for zone: %s", zone)
	return data, nil
}

// InvalidateZoneCache invalidates global cache for a specific zone
func (cc *CachedClient) InvalidateZoneCache(zone string) {
	globalCacheMutex.Lock()
	globalZoneCache.Invalidate(zone)
	log.Printf("[DEBUG] GLOBAL CACHE INVALIDATED for zone %s", zone)
	globalCacheMutex.Unlock()
}

// ClearZoneCache clears all global zone caches
func (cc *CachedClient) ClearZoneCache() {
	globalCacheMutex.Lock()
	globalZoneCache.Clear()
	log.Printf("[DEBUG] GLOBAL CACHE CLEARED")
	globalCacheMutex.Unlock()
}

// Provider returns a terraform.ResourceProvider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Reg.ru username",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Reg.ru password",
				Sensitive:   true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"regru_dns_a_record":     resources.ResourceDNSARecord(),
			"regru_dns_aaaa_record":  resources.ResourceDNSAAAARecord(),
			"regru_dns_cname_record": resources.ResourceDNSCNAMERecord(),
			"regru_dns_mx_record":    resources.ResourceDNSMXRecord(),
			"regru_dns_ns_record":    resources.ResourceDNSNSRecord(),
			"regru_dns_txt_record":   resources.ResourceDNSTXTRecord(),
			"regru_dns_srv_record":   resources.ResourceDNSSRVRecord(),
			"regru_dns_caa_record":   resources.ResourceDNSCAARecord(),
		},
		ConfigureFunc: providerConfigure,
	}
}

// providerConfigure configures the provider with a cached client
func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	username := d.Get("username").(string)
	password := d.Get("password").(string)

	// Create the base client
	baseClient := client.NewClient(username, password)

	// Create cached client with global caching
	cachedClient := &CachedClient{
		Client: baseClient,
	}

	return cachedClient, nil
}
