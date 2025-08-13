package strategies

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"terraform-provider-regru/resource/base"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// NSRecordStrategy implements the strategy for NS records
type NSRecordStrategy struct {
	base.BaseStrategy
}

// NewNSRecordStrategy creates a new NS record strategy
func NewNSRecordStrategy() *NSRecordStrategy {
	return &NSRecordStrategy{}
}

// GetRecords returns the NS records from the resource data
func (s *NSRecordStrategy) GetRecords(d *schema.ResourceData) []interface{} {
	records := d.Get("record").([]interface{})
	var allRecords []interface{}

	for _, recordInterface := range records {
		recordMap := recordInterface.(map[string]interface{})
		servers := recordMap["servers"].([]interface{})

		for _, server := range servers {
			allRecords = append(allRecords, server)
		}
	}

	return allRecords
}

// SetResourceID sets a stable resource ID for the NS record
func (s *NSRecordStrategy) SetResourceID(d *schema.ResourceData, zone, name, recordType string) {
	d.SetId(fmt.Sprintf("%s/%s", zone, name))
}

// ValidateRecords validates NS records
func (s *NSRecordStrategy) ValidateRecords(records []interface{}) error {
	if len(records) == 0 {
		return fmt.Errorf("NS record must have at least one name server")
	}

	for _, record := range records {
		server := record.(string)
		if server == "" {
			return fmt.Errorf("NS record server cannot be empty")
		}
	}

	return nil
}

// Create creates NS records
func (s *NSRecordStrategy) Create(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for NS record creation")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)
	records := d.Get("record").([]interface{})

	s.LogResourceOperation("Creating", "NS", zone, name)

	// Create NS records for each priority group
	for _, recordInterface := range records {
		recordMap := recordInterface.(map[string]interface{})
		priority := recordMap["priority"].(int)
		servers := recordMap["servers"].([]interface{})

		for _, serverInterface := range servers {
			server := serverInterface.(string)

			// For NS records, we need to add trailing dots for domain names
			apiRecord := s.AddTrailingDot(server)
			response, err := c.AddRecord("NS", zone, name, apiRecord, &priority)
			if err != nil {
				return fmt.Errorf("failed to create NS record: %w", err)
			}

			// Check API response for errors
			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to create NS record: %w", err)
			}
		}
	}

	s.SetResourceID(d, zone, name, "NS")
	c.InvalidateZoneCache(zone)
	return nil
}

// Read reads NS records from the API
func (s *NSRecordStrategy) Read(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for NS record read")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Reading", "NS", zone, name)

	response, err := c.GetRecordsWithCache(zone)
	if err != nil {
		return fmt.Errorf("failed to get zone records: %w", err)
	}

	var zoneResponse base.DNSZoneResponse
	if err := json.Unmarshal(response, &zoneResponse); err != nil {
		return fmt.Errorf("failed to parse DNS records response: %w", err)
	}

	// Find NS records for this subdomain
	var nsRecords []map[string]interface{}
	priorityGroups := make(map[int][]string)

	for _, domain := range zoneResponse.Answer.Domains {
		if domain.Dname == zone {
			for _, rr := range domain.Rrs {
				if rr.Subname == name && rr.Rectype == "NS" {
					// Remove trailing dot from content for consistency
					server := s.NormalizeDomain(rr.Content)
					priority := rr.Prio // Use the priority from the record

					priorityGroups[priority] = append(priorityGroups[priority], server)
				}
			}
			break
		}
	}

	if len(priorityGroups) == 0 {
		// No records found, mark as deleted
		d.SetId("")
		return nil
	}

	// Convert priority groups to record blocks
	for priority, servers := range priorityGroups {
		// Sort servers for consistent ordering
		sort.Strings(servers)

		record := map[string]interface{}{
			"priority": priority,
			"servers":  servers,
		}
		nsRecords = append(nsRecords, record)
	}

	// Sort by priority for consistent ordering
	sort.Slice(nsRecords, func(i, j int) bool {
		return nsRecords[i]["priority"].(int) < nsRecords[j]["priority"].(int)
	})

	// Set the data
	d.Set("zone", zone)
	d.Set("name", name)
	d.Set("record", nsRecords)

	return nil
}

// Update updates NS records using surgical approach - only change what actually changed
func (s *NSRecordStrategy) Update(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for NS record update")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Updating", "NS", zone, name)

	// Get old and new record configurations
	oldRecordsInterface, newRecordsInterface := d.GetChange("record")
	oldRecords, oldOk := oldRecordsInterface.([]interface{})
	newRecords, newOk := newRecordsInterface.([]interface{})

	if !oldOk || !newOk {
		log.Printf("[DEBUG] Could not parse old/new records, falling back to delete-all + create-all")
		return s.recreateAllRecords(client, d)
	}

	// Parse old and new records into comparable structures
	oldNSRecords := s.parseRecordsFromState(oldRecords)
	newNSRecords := s.parseRecordsFromState(newRecords)

	// Calculate what needs to be removed and what needs to be added
	toRemove := s.findRecordsToRemove(oldNSRecords, newNSRecords)
	toAdd := s.findRecordsToAdd(oldNSRecords, newNSRecords)

	log.Printf("[DEBUG] NS Update: %d records to remove, %d records to add", len(toRemove), len(toAdd))

	// Remove records that are no longer needed
	for _, record := range toRemove {
		log.Printf("[DEBUG] Removing NS record: %s (priority: %d)", record.Server, record.Priority)
		apiRecord := s.AddTrailingDot(record.Server)
		response, err := c.RemoveRecord(zone, name, "NS", apiRecord, &record.Priority)
		if err != nil {
			return fmt.Errorf("failed to remove NS record %s: %w", record.Server, err)
		}

		// Check API response for errors
		if response != nil {
			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to remove NS record %s: %w", record.Server, err)
			}
		}
	}

	// Add new records
	for _, record := range toAdd {
		log.Printf("[DEBUG] Adding NS record: %s (priority: %d)", record.Server, record.Priority)
		apiRecord := s.AddTrailingDot(record.Server)
		response, err := c.AddRecord("NS", zone, name, apiRecord, &record.Priority)
		if err != nil {
			return fmt.Errorf("failed to add NS record %s: %w", record.Server, err)
		}

		// Check API response for errors
		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to add NS record %s: %w", record.Server, err)
		}
	}

	c.InvalidateZoneCache(zone)
	return nil
}

// recreateAllRecords is the fallback method (original behavior)
func (s *NSRecordStrategy) recreateAllRecords(client interface{}, d *schema.ResourceData) error {
	// For NS records, we'll delete and recreate since the structure might have changed
	// First delete existing records
	if err := s.Delete(client, d); err != nil {
		return err
	}

	// Then create new records
	return s.Create(client, d)
}

// NSRecord represents a single NS record for comparison
type NSRecord struct {
	Priority int
	Server   string
}

// parseRecordsFromState converts record blocks to NSRecord structs for easy comparison
func (s *NSRecordStrategy) parseRecordsFromState(records []interface{}) []NSRecord {
	var nsRecords []NSRecord

	for _, recordInterface := range records {
		recordMap, ok := recordInterface.(map[string]interface{})
		if !ok {
			continue
		}

		priority, priorityOk := recordMap["priority"].(int)
		if !priorityOk {
			continue
		}

		serversInterface, serversOk := recordMap["servers"].([]interface{})
		if !serversOk {
			continue
		}

		// Convert each server in this priority group to individual NSRecord
		for _, serverInterface := range serversInterface {
			if server, serverOk := serverInterface.(string); serverOk {
				nsRecords = append(nsRecords, NSRecord{
					Priority: priority,
					Server:   server,
				})
			}
		}
	}

	return nsRecords
}

// findRecordsToRemove finds records that exist in old but not in new
func (s *NSRecordStrategy) findRecordsToRemove(oldRecords, newRecords []NSRecord) []NSRecord {
	var toRemove []NSRecord

	for _, oldRecord := range oldRecords {
		found := false
		for _, newRecord := range newRecords {
			if oldRecord.Priority == newRecord.Priority && oldRecord.Server == newRecord.Server {
				found = true
				break
			}
		}
		if !found {
			toRemove = append(toRemove, oldRecord)
		}
	}

	return toRemove
}

// findRecordsToAdd finds records that exist in new but not in old
func (s *NSRecordStrategy) findRecordsToAdd(oldRecords, newRecords []NSRecord) []NSRecord {
	var toAdd []NSRecord

	for _, newRecord := range newRecords {
		found := false
		for _, oldRecord := range oldRecords {
			if newRecord.Priority == oldRecord.Priority && newRecord.Server == oldRecord.Server {
				found = true
				break
			}
		}
		if !found {
			toAdd = append(toAdd, newRecord)
		}
	}

	return toAdd
}

// Delete deletes NS records
func (s *NSRecordStrategy) Delete(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for NS record deletion")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Deleting", "NS", zone, name)

	// Get the old NS records to remove
	oldRecords, _ := d.GetChange("record")
	if oldRecords != nil {
		oldRecordsList := oldRecords.([]interface{})
		for _, recordInterface := range oldRecordsList {
			recordMap := recordInterface.(map[string]interface{})
			priority := recordMap["priority"].(int)
			servers := recordMap["servers"].([]interface{})

			for _, serverInterface := range servers {
				server := serverInterface.(string)

				// For NS records, we need to add trailing dots for domain names
				apiRecord := s.AddTrailingDot(server)
				response, err := c.RemoveRecord(zone, name, "NS", apiRecord, &priority)
				if err != nil {
					return fmt.Errorf("failed to delete NS record: %w", err)
				}

				// Check API response for errors
				if err := base.CheckAPIResponseForErrors(response); err != nil {
					return fmt.Errorf("failed to delete NS record: %w", err)
				}
			}
		}
	}

	// Invalidate cache after deletion
	c.InvalidateZoneCache(zone)
	return nil
}

// Import imports an existing NS record
func (s *NSRecordStrategy) Import(client interface{}, d *schema.ResourceData) error {
	// Parse the import ID using the common format
	zone, name, err := s.ParseResourceID(d.Id())
	if err != nil {
		return err
	}

	d.Set("zone", zone)
	d.Set("name", name)

	return s.Read(client, d)
}
