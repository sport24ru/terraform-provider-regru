package strategies

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"terraform-provider-regru/resource/base"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// SRVRecordStrategy implements the strategy for SRV records
type SRVRecordStrategy struct {
	base.BaseStrategy
}

// NewSRVRecordStrategy creates a new SRV record strategy
func NewSRVRecordStrategy() *SRVRecordStrategy {
	return &SRVRecordStrategy{}
}

// SRVRecord represents a single SRV record
type SRVRecord struct {
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
	Port     int    `json:"port"`
	Target   string `json:"target"`
}

// String returns a sortable string representation of the SRV record
func (srv SRVRecord) String() string {
	return fmt.Sprintf("%d_%d_%d_%s", srv.Priority, srv.Weight, srv.Port, srv.Target)
}

// SetResourceID sets a stable resource ID for the SRV record
func (s *SRVRecordStrategy) SetResourceID(d *schema.ResourceData, zone, name, recordType string) {
	d.SetId(fmt.Sprintf("%s/%s", zone, name))
}

// parseSRVRecords converts the records from schema to SRVRecord structs
func (s *SRVRecordStrategy) parseSRVRecords(d *schema.ResourceData) ([]SRVRecord, error) {
	srvRecordBlocks := d.Get("record").([]interface{})
	var srvRecords []SRVRecord

	for _, recordBlock := range srvRecordBlocks {
		recordMap := recordBlock.(map[string]interface{})
		
		priority := recordMap["priority"].(int)
		weight := recordMap["weight"].(int)
		port := recordMap["port"].(int)
		targets := recordMap["targets"].([]interface{})

		for _, target := range targets {
			targetStr := target.(string)
			srvRecord := SRVRecord{
				Priority: priority,
				Weight:   weight,
				Port:     port,
				Target:   targetStr,
			}
			srvRecords = append(srvRecords, srvRecord)
		}
	}

	return srvRecords, nil
}

// Create creates SRV records
func (s *SRVRecordStrategy) Create(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)

	srvRecords, err := s.parseSRVRecords(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating SRV records for %s.%s: %v", name, zone, srvRecords)

	s.LogResourceOperation("Creating", "SRV", zone, name)

	// Validate records
	if len(srvRecords) == 0 {
		return fmt.Errorf("at least one SRV record must be specified")
	}

	// Sort records for consistent processing
	sort.Slice(srvRecords, func(i, j int) bool {
		return srvRecords[i].String() < srvRecords[j].String()
	})

	// Add each SRV record using the specific AddSRVRecord method
	for _, srvRecord := range srvRecords {
		log.Printf("[DEBUG] Adding SRV record: %s.%s -> %d %d %d %s", name, zone,
			srvRecord.Priority, srvRecord.Weight, srvRecord.Port, srvRecord.Target)

		response, err := c.AddSRVRecord(zone, name, srvRecord.Target, &srvRecord.Priority, &srvRecord.Weight, &srvRecord.Port)
		if err != nil {
			return fmt.Errorf("failed to create SRV record %s: %w", srvRecord.Target, err)
		}

		// Check API response for errors
		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to create SRV record %s: %w", srvRecord.Target, err)
		}
	}

	// Set resource ID and common attributes
	s.SetResourceID(d, zone, name, "SRV")

	return s.Read(meta, d)
}

// Read reads SRV records from the API
func (s *SRVRecordStrategy) Read(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)
	expectedPriority := s.GetPriority(d)

	s.LogResourceOperation("Reading", "SRV", zone, name)

	// Get zone data from API (with caching)
	response, err := c.GetRecordsWithCache(zone)
	if err != nil {
		return fmt.Errorf("failed to get zone records: %w", err)
	}

	log.Printf("[DEBUG] API Response: %s", string(response))

	// Parse the response
	var zoneResponse base.DNSZoneResponse
	if err := json.Unmarshal(response, &zoneResponse); err != nil {
		return fmt.Errorf("failed to parse DNS records response: %w", err)
	}

	log.Printf("[DEBUG] Parsed response - domains: %d", len(zoneResponse.Answer.Domains))

	// Parse response and find SRV records
	var foundSRVRecords []SRVRecord
	for _, domain := range zoneResponse.Answer.Domains {
		log.Printf("[DEBUG] Processing domain: %s, records: %d", domain.Dname, len(domain.Rrs))
		for _, record := range domain.Rrs {
			log.Printf("[DEBUG] Record: type=%s, subname=%s, content=%s, prio=%d, weight=%d, port=%d",
				record.Rectype, record.Subname, record.Content, record.Prio, record.Weight, record.Port)

			if record.Rectype == "SRV" && record.Subname == name {
				// For SRV records, match by priority if specified
				if expectedPriority != nil && record.Prio != *expectedPriority {
					log.Printf("[DEBUG] Priority mismatch: expected %d, got %d", *expectedPriority, record.Prio)
					continue
				}

				// SRV content format: "weight port target" or just target
				// The API should provide weight and port in separate fields
				target := record.Content
				if strings.Contains(target, " ") {
					// Parse "weight port target" format if API returns combined format
					parts := strings.Fields(target)
					if len(parts) >= 3 {
						target = parts[2] // Extract just the target
					}
				}

				srvRecord := SRVRecord{
					Priority: record.Prio,
					Weight:   record.Weight,
					Port:     record.Port,
					Target:   s.NormalizeDomain(target),
				}

				foundSRVRecords = append(foundSRVRecords, srvRecord)
			}
		}
	}

	if len(foundSRVRecords) == 0 {
		log.Printf("[DEBUG] No SRV records found for %s.%s", name, zone)
		// No records found, mark as deleted
		d.SetId("")
		return nil
	}

	// Sort records for consistent state
	sort.Slice(foundSRVRecords, func(i, j int) bool {
		return foundSRVRecords[i].String() < foundSRVRecords[j].String()
	})

	log.Printf("[DEBUG] Sorted SRV records: %v", foundSRVRecords)

	// Set the data
	d.Set("zone", zone)
	d.Set("name", name)

	// Group SRV records by priority, weight, and port
	recordGroups := make(map[string][]string)
	for _, srvRecord := range foundSRVRecords {
		key := fmt.Sprintf("%d_%d_%d", srvRecord.Priority, srvRecord.Weight, srvRecord.Port)
		recordGroups[key] = append(recordGroups[key], srvRecord.Target)
	}

	// Convert to the new record block structure
	var recordBlocks []map[string]interface{}
	for key, targets := range recordGroups {
		parts := strings.Split(key, "_")
		priority, _ := strconv.Atoi(parts[0])
		weight, _ := strconv.Atoi(parts[1])
		port, _ := strconv.Atoi(parts[2])
		
		// Sort targets for consistent state
		sort.Strings(targets)
		
		recordBlock := map[string]interface{}{
			"priority": priority,
			"weight":   weight,
			"port":     port,
			"targets":  targets,
		}
		recordBlocks = append(recordBlocks, recordBlock)
	}

	// Sort record blocks by priority, weight, port for consistent state
	sort.Slice(recordBlocks, func(i, j int) bool {
		prioI := recordBlocks[i]["priority"].(int)
		prioJ := recordBlocks[j]["priority"].(int)
		if prioI != prioJ {
			return prioI < prioJ
		}
		
		weightI := recordBlocks[i]["weight"].(int)
		weightJ := recordBlocks[j]["weight"].(int)
		if weightI != weightJ {
			return weightI < weightJ
		}
		
		portI := recordBlocks[i]["port"].(int)
		portJ := recordBlocks[j]["port"].(int)
		return portI < portJ
	})

	d.Set("record", recordBlocks)

	log.Printf("[DEBUG] Successfully read %d SRV records", len(foundSRVRecords))
	return nil
}

// Update updates SRV records
func (s *SRVRecordStrategy) Update(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Updating", "SRV", zone, name)

	if d.HasChange("record") {
		// Get old and new configurations
		oldSRVRecords, err := s.getOldSRVRecords(d)
		if err != nil {
			return err
		}

		newSRVRecords, err := s.parseSRVRecords(d)
		if err != nil {
			return err
		}

		// Sort both sets for comparison
		sort.Slice(oldSRVRecords, func(i, j int) bool {
			return oldSRVRecords[i].String() < oldSRVRecords[j].String()
		})
		sort.Slice(newSRVRecords, func(i, j int) bool {
			return newSRVRecords[i].String() < newSRVRecords[j].String()
		})

		// Find records to remove
		recordsToRemove := []SRVRecord{}
		for _, oldRecord := range oldSRVRecords {
			found := false
			for _, newRecord := range newSRVRecords {
				if oldRecord.String() == newRecord.String() {
					found = true
					break
				}
			}
			if !found {
				recordsToRemove = append(recordsToRemove, oldRecord)
			}
		}

		// Find records to add
		recordsToAdd := []SRVRecord{}
		for _, newRecord := range newSRVRecords {
			found := false
			for _, oldRecord := range oldSRVRecords {
				if newRecord.String() == oldRecord.String() {
					found = true
					break
				}
			}
			if !found {
				recordsToAdd = append(recordsToAdd, newRecord)
			}
		}

		// Remove old records
		for _, record := range recordsToRemove {
			log.Printf("[DEBUG] Removing SRV record: %s -> %d %d %d %s", name,
				record.Priority, record.Weight, record.Port, record.Target)
			response, err := c.RemoveSRVRecord(zone, name, record.Target, &record.Priority, &record.Weight, &record.Port)
			if err != nil {
				return fmt.Errorf("failed to remove SRV record %s: %w", record.Target, err)
			}

			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to remove SRV record %s: %w", record.Target, err)
			}
		}

		// Add new records
		for _, record := range recordsToAdd {
			log.Printf("[DEBUG] Adding SRV record: %s -> %d %d %d %s", name,
				record.Priority, record.Weight, record.Port, record.Target)
			response, err := c.AddSRVRecord(zone, name, record.Target, &record.Priority, &record.Weight, &record.Port)
			if err != nil {
				return fmt.Errorf("failed to add SRV record %s: %w", record.Target, err)
			}

			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to add SRV record %s: %w", record.Target, err)
			}
		}

		// Invalidate cache after updates
		c.InvalidateZoneCache(zone)
	}

	return s.Read(meta, d)
}

// getOldSRVRecords reconstructs old SRV records from the change data
func (s *SRVRecordStrategy) getOldSRVRecords(d *schema.ResourceData) ([]SRVRecord, error) {
	old, _ := d.GetChange("record")
	oldRecordBlocks := old.([]interface{})

	var srvRecords []SRVRecord
	for _, recordBlock := range oldRecordBlocks {
		recordMap := recordBlock.(map[string]interface{})
		
		priority := recordMap["priority"].(int)
		weight := recordMap["weight"].(int)
		port := recordMap["port"].(int)
		targets := recordMap["targets"].([]interface{})

		for _, target := range targets {
			targetStr := target.(string)
			srvRecord := SRVRecord{
				Priority: priority,
				Weight:   weight,
				Port:     port,
				Target:   targetStr,
			}
			srvRecords = append(srvRecords, srvRecord)
		}
	}

	return srvRecords, nil
}

// Delete deletes SRV records
func (s *SRVRecordStrategy) Delete(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)

	srvRecords, err := s.parseSRVRecords(d)
	if err != nil {
		return err
	}

	s.LogResourceOperation("Deleting", "SRV", zone, name)

	// Remove each SRV record
	for _, srvRecord := range srvRecords {
		log.Printf("[DEBUG] Removing SRV record: %s -> %d %d %d %s", name,
			srvRecord.Priority, srvRecord.Weight, srvRecord.Port, srvRecord.Target)
		response, err := c.RemoveSRVRecord(zone, name, srvRecord.Target, &srvRecord.Priority, &srvRecord.Weight, &srvRecord.Port)
		if err != nil {
			return fmt.Errorf("failed to delete SRV record %s: %w", srvRecord.Target, err)
		}

		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to delete SRV record %s: %w", srvRecord.Target, err)
		}
	}

	// Invalidate cache after deletion
	c.InvalidateZoneCache(zone)

	return nil
}

// Import imports an existing SRV record
func (s *SRVRecordStrategy) Import(meta interface{}, d *schema.ResourceData) error {
	// Parse the import ID using the common format
	zone, name, err := s.ParseResourceID(d.Id())
	if err != nil {
		return err
	}

	d.Set("zone", zone)
	d.Set("name", name)

	return s.Read(meta, d)
}
