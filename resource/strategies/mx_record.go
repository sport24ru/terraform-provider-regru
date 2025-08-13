package strategies

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"terraform-provider-regru/resource/base"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// MXRecordStrategy implements the strategy for MX records
type MXRecordStrategy struct {
	base.BaseStrategy
}

// GetRecords returns the MX records from the resource data
func (s *MXRecordStrategy) GetRecords(d *schema.ResourceData) []interface{} {
	mxRecords := d.Get("record").([]interface{})
	var allRecords []interface{}

	for _, mxRecord := range mxRecords {
		mxRecordMap := mxRecord.(map[string]interface{})
		servers := mxRecordMap["servers"].([]interface{})
		allRecords = append(allRecords, servers...)
	}

	return allRecords
}

// GetPriority returns the priority from the resource data
func (s *MXRecordStrategy) GetPriority(d *schema.ResourceData) *int {
	// For the new structure, we'll use the first priority found
	// This is a simplification - in practice, you might want to handle multiple priorities differently
	mxRecords := d.Get("record").([]interface{})
	if len(mxRecords) > 0 {
		firstRecord := mxRecords[0].(map[string]interface{})
		if priority, ok := firstRecord["priority"]; ok {
			priorityInt := priority.(int)
			return &priorityInt
		}
	}
	return nil
}

// SetResourceID sets a stable resource ID for the MX record
func (s *MXRecordStrategy) SetResourceID(d *schema.ResourceData, zone, name, recordType string) {
	d.SetId(fmt.Sprintf("%s/%s", zone, name))
}

// ValidateRecords validates MX records
func (s *MXRecordStrategy) ValidateRecords(records []interface{}) error {
	if len(records) == 0 {
		return fmt.Errorf("at least one MX record is required")
	}

	for _, record := range records {
		if recordStr, ok := record.(string); ok {
			if recordStr == "" {
				return fmt.Errorf("MX record cannot be empty")
			}
		} else {
			return fmt.Errorf("MX record must be a string")
		}
	}

	return nil
}

// Create creates MX records
func (s *MXRecordStrategy) Create(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for MX record creation")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)
	mxRecords := d.Get("record").([]interface{})

	s.LogResourceOperation("Creating", "MX", zone, name)

	// Create each MX record set
	for _, mxRecord := range mxRecords {
		mxRecordMap := mxRecord.(map[string]interface{})
		priority := mxRecordMap["priority"].(int)
		servers := mxRecordMap["servers"].([]interface{})

		// Convert to string slice and sort alphabetically for consistent ordering
		serverStrings := make([]string, len(servers))
		for i, server := range servers {
			serverStrings[i] = server.(string)
		}
		sort.Strings(serverStrings)
		log.Printf("[DEBUG] Creating MX record set with priority %d: %v", priority, serverStrings)

		for _, serverStr := range serverStrings {
			log.Printf("[DEBUG] Creating MX record: %s %s %s (priority: %d)", zone, name, serverStr, priority)

			// For MX records, we need to add trailing dots for domain names
			apiRecord := s.AddTrailingDot(serverStr)
			response, err := c.AddRecord("MX", zone, name, apiRecord, &priority)
			if err != nil {
				return fmt.Errorf("failed to create MX record %s: %w", serverStr, err)
			}

			// Check API response for errors
			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to create MX record %s: %w", serverStr, err)
			}
		}
	}

	s.SetResourceID(d, zone, name, "MX")
	c.InvalidateZoneCache(zone)
	return nil
}

// Read reads MX records from the API
func (s *MXRecordStrategy) Read(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for MX record read")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Reading", "MX", zone, name)

	response, err := c.GetRecordsWithCache(zone)
	if err != nil {
		return fmt.Errorf("failed to get zone records: %w", err)
	}

	var zoneResponse base.DNSZoneResponse
	if err := json.Unmarshal(response, &zoneResponse); err != nil {
		return fmt.Errorf("failed to parse DNS records response: %w", err)
	}

	// Group MX records by priority
	priorityGroups := make(map[int][]string)
	for _, domain := range zoneResponse.Answer.Domains {
		if domain.Dname == zone {
			for _, rr := range domain.Rrs {
				if rr.Subname == name && rr.Rectype == "MX" {
					// Remove trailing dot from content for consistency
					content := s.NormalizeDomain(rr.Content)
					priorityGroups[rr.Prio] = append(priorityGroups[rr.Prio], content)
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

	// Convert to the new mx_records structure
	var mxRecords []map[string]interface{}
	for priority, records := range priorityGroups {
		// Sort records alphabetically for consistent state
		sort.Strings(records)
		log.Printf("[DEBUG] MX records with priority %d: %v", priority, records)

		mxRecord := map[string]interface{}{
			"priority": priority,
			"servers":  records,
		}
		mxRecords = append(mxRecords, mxRecord)
	}

	// Sort by priority for consistent ordering
	sort.Slice(mxRecords, func(i, j int) bool {
		return mxRecords[i]["priority"].(int) < mxRecords[j]["priority"].(int)
	})

	// Set the data
	d.Set("zone", zone)
	d.Set("name", name)
	d.Set("record", mxRecords)

	return nil
}

// Update updates MX records using surgical approach - only change what actually changed
func (s *MXRecordStrategy) Update(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for MX record update")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Updating", "MX", zone, name)

	// Get old and new record configurations
	oldRecordsInterface, newRecordsInterface := d.GetChange("record")
	oldRecords, oldOk := oldRecordsInterface.([]interface{})
	newRecords, newOk := newRecordsInterface.([]interface{})

	if !oldOk || !newOk {
		log.Printf("[DEBUG] Could not parse old/new records, falling back to delete-all + create-all")
		return s.recreateAllRecords(client, d)
	}

	// Parse old and new records into comparable structures
	oldMXRecords := s.parseRecordsFromState(oldRecords)
	newMXRecords := s.parseRecordsFromState(newRecords)

	// Calculate what needs to be removed and what needs to be added
	toRemove := s.findRecordsToRemove(oldMXRecords, newMXRecords)
	toAdd := s.findRecordsToAdd(oldMXRecords, newMXRecords)

	log.Printf("[DEBUG] MX Update: %d records to remove, %d records to add", len(toRemove), len(toAdd))

	// Remove records that are no longer needed
	for _, record := range toRemove {
		log.Printf("[DEBUG] Removing MX record: %s (priority: %d)", record.Server, record.Priority)
		apiRecord := s.AddTrailingDot(record.Server)
		response, err := c.RemoveRecord(zone, name, "MX", apiRecord, &record.Priority)
		if err != nil {
			if err := s.HandleAPIError(err, "remove"); err != nil {
				return fmt.Errorf("failed to remove MX record %s: %w", record.Server, err)
			}
		}

		// Check API response for errors
		if response != nil {
			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to remove MX record %s: %w", record.Server, err)
			}
		}
	}

	// Add new records
	for _, record := range toAdd {
		log.Printf("[DEBUG] Adding MX record: %s (priority: %d)", record.Server, record.Priority)
		apiRecord := s.AddTrailingDot(record.Server)
		response, err := c.AddRecord("MX", zone, name, apiRecord, &record.Priority)
		if err != nil {
			return fmt.Errorf("failed to add MX record %s: %w", record.Server, err)
		}

		// Check API response for errors
		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to add MX record %s: %w", record.Server, err)
		}
	}

	c.InvalidateZoneCache(zone)
	return nil
}

// recreateAllRecords is the fallback method (original behavior)
func (s *MXRecordStrategy) recreateAllRecords(client interface{}, d *schema.ResourceData) error {
	// For simplicity, we'll delete all existing records and recreate them
	// This ensures consistency with the new structure
	if err := s.Delete(client, d); err != nil {
		return fmt.Errorf("failed to remove old MX records: %w", err)
	}

	// Create new records
	if err := s.Create(client, d); err != nil {
		return fmt.Errorf("failed to create new MX records: %w", err)
	}

	return nil
}

// MXRecord represents a single MX record for comparison
type MXRecord struct {
	Priority int
	Server   string
}

// parseRecordsFromState converts record blocks to MXRecord structs for easy comparison
func (s *MXRecordStrategy) parseRecordsFromState(records []interface{}) []MXRecord {
	var mxRecords []MXRecord

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

		// Convert each server in this priority group to individual MXRecord
		for _, serverInterface := range serversInterface {
			if server, serverOk := serverInterface.(string); serverOk {
				mxRecords = append(mxRecords, MXRecord{
					Priority: priority,
					Server:   server,
				})
			}
		}
	}

	return mxRecords
}

// findRecordsToRemove finds records that exist in old but not in new
func (s *MXRecordStrategy) findRecordsToRemove(oldRecords, newRecords []MXRecord) []MXRecord {
	var toRemove []MXRecord

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
func (s *MXRecordStrategy) findRecordsToAdd(oldRecords, newRecords []MXRecord) []MXRecord {
	var toAdd []MXRecord

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

// Delete deletes MX records
func (s *MXRecordStrategy) Delete(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for MX record deletion")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Deleting", "MX", zone, name)

	// Get all MX records from the current state to remove them
	response, err := c.GetRecordsWithCache(zone)
	if err != nil {
		return fmt.Errorf("failed to get zone records for deletion: %w", err)
	}

	var zoneResponse base.DNSZoneResponse
	if err := json.Unmarshal(response, &zoneResponse); err != nil {
		return fmt.Errorf("failed to parse DNS records response for deletion: %w", err)
	}

	// Remove all MX records for this subdomain
	for _, domain := range zoneResponse.Answer.Domains {
		if domain.Dname == zone {
			for _, rr := range domain.Rrs {
				if rr.Subname == name && rr.Rectype == "MX" {
					log.Printf("[DEBUG] Removing MX record: %s (priority: %d)", rr.Content, rr.Prio)

					// For MX records, we need to add trailing dots when removing
					apiRecord := s.AddTrailingDot(rr.Content)
					response, err := c.RemoveRecord(zone, name, "MX", apiRecord, &rr.Prio)
					if err != nil {
						if err := s.HandleAPIError(err, "remove"); err != nil {
							return err
						}
					}

					// Check API response for errors
					if err := base.CheckAPIResponseForErrors(response); err != nil {
						return fmt.Errorf("failed to remove MX record %s: %w", rr.Content, err)
					}
				}
			}
			break
		}
	}

	d.SetId("")
	c.InvalidateZoneCache(zone)
	return nil
}

// Import imports an existing MX record
func (s *MXRecordStrategy) Import(client interface{}, d *schema.ResourceData) error {
	// Parse the import ID using the common format
	zone, name, err := s.ParseResourceID(d.Id())
	if err != nil {
		return err
	}

	// Set the basic fields
	d.Set("zone", zone)
	d.Set("name", name)

	// Read the current state to populate records and priority
	err = s.Read(client, d)
	if err != nil {
		return err
	}

	// Note: During import, we can't access the configuration file to get the desired order
	// The records will be set in the order returned by the API
	// The configuration order will be respected during subsequent operations
	log.Printf("[DEBUG] Import: Records imported in API order, configuration order will be applied on next plan/apply")

	return nil
}
