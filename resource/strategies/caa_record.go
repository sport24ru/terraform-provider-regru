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

// CAARecordStrategy implements the strategy for CAA records
type CAARecordStrategy struct {
	base.BaseStrategy
}

// NewCAARecordStrategy creates a new CAA record strategy
func NewCAARecordStrategy() *CAARecordStrategy {
	return &CAARecordStrategy{}
}

// CAARecord represents a single CAA record
type CAARecord struct {
	Flag  int    `json:"flag"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

// String returns a sortable string representation of the CAA record
func (caa CAARecord) String() string {
	return fmt.Sprintf("%d_%s_%s", caa.Flag, caa.Tag, caa.Value)
}

// parseCAARecords converts the record from schema to CAARecord structs
func (s *CAARecordStrategy) parseCAARecords(d *schema.ResourceData) ([]CAARecord, error) {
	recordList := d.Get("record").([]interface{})

	var caaRecords []CAARecord
	for _, recordInterface := range recordList {
		recordMap := recordInterface.(map[string]interface{})

		flag := recordMap["flag"].(int)
		tag := recordMap["tag"].(string)
		value := recordMap["value"].(string)

		caaRecord := CAARecord{
			Flag:  flag,
			Tag:   tag,
			Value: value,
		}

		caaRecords = append(caaRecords, caaRecord)
	}

	return caaRecords, nil
}

// Create creates CAA records
func (s *CAARecordStrategy) Create(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)

	caaRecords, err := s.parseCAARecords(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating CAA records for %s.%s: %v", name, zone, caaRecords)

	s.LogResourceOperation("Creating", "CAA", zone, name)

	// Validate records
	if len(caaRecords) == 0 {
		return fmt.Errorf("at least one CAA record must be specified")
	}

	// Sort records for consistent processing
	sort.Slice(caaRecords, func(i, j int) bool {
		return caaRecords[i].String() < caaRecords[j].String()
	})

	// Add each CAA record using the specific AddCAARecord method
	for _, caaRecord := range caaRecords {
		log.Printf("[DEBUG] Adding CAA record: %s.%s -> %d %s %s", name, zone,
			caaRecord.Flag, caaRecord.Tag, caaRecord.Value)

		response, err := c.AddCAARecord(zone, name, caaRecord.Value, &caaRecord.Flag, &caaRecord.Tag)
		if err != nil {
			return fmt.Errorf("failed to create CAA record %s: %w", caaRecord.Value, err)
		}

		// Check API response for errors
		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to create CAA record %s: %w", caaRecord.Value, err)
		}
	}

	// Set resource ID
	d.SetId(fmt.Sprintf("%s/%s/%s", zone, name, "CAA"))

	return s.Read(meta, d)
}

// Read reads CAA records from the API
func (s *CAARecordStrategy) Read(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Reading", "CAA", zone, name)

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

	// Parse response and find CAA records
	var foundCAARecords []CAARecord
	for _, domain := range zoneResponse.Answer.Domains {
		log.Printf("[DEBUG] Processing domain: %s, records: %d", domain.Dname, len(domain.Rrs))
		for _, record := range domain.Rrs {
			log.Printf("[DEBUG] Record: type=%s, subname=%s, content=%s, flag=%d, tag=%s",
				record.Rectype, record.Subname, record.Content, record.Flag, record.Tag)

			if record.Rectype == "CAA" && record.Subname == name {
				var flag int
				var tag string
				var value string

				// The API might provide flag and tag in separate fields or combined in content
				if record.Flag != 0 || record.Tag != "" {
					// Use separate fields if available
					flag = record.Flag
					tag = record.Tag
					value = record.Content
				} else {
					// Parse CAA content format: "flag tag \"value\""
					parts := strings.Fields(record.Content)
					if len(parts) >= 3 {
						// Parse flag
						if flagVal, err := strconv.Atoi(parts[0]); err == nil {
							flag = flagVal
						}

						// Parse tag
						tag = parts[1]

						// Parse value (remove quotes if present)
						value = strings.Join(parts[2:], " ")
						if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
							value = strings.Trim(value, "\"")
						}
					} else {
						// Fallback: use content as value
						value = record.Content
					}
				}

				caaRecord := CAARecord{
					Flag:  flag,
					Tag:   tag,
					Value: s.NormalizeDomain(value),
				}

				foundCAARecords = append(foundCAARecords, caaRecord)
			}
		}
	}

	if len(foundCAARecords) == 0 {
		log.Printf("[DEBUG] No CAA records found for %s.%s", name, zone)
		// No records found, mark as deleted
		d.SetId("")
		return nil
	}

	// Sort records for consistent state
	sort.Slice(foundCAARecords, func(i, j int) bool {
		return foundCAARecords[i].String() < foundCAARecords[j].String()
	})

	log.Printf("[DEBUG] Sorted CAA records: %v", foundCAARecords)

	// Set the data
	d.Set("zone", zone)
	d.Set("name", name)

	// Convert to interface slice for Terraform record schema
	recordInterface := make([]interface{}, len(foundCAARecords))
	for i, caaRecord := range foundCAARecords {
		recordInterface[i] = map[string]interface{}{
			"flag":  caaRecord.Flag,
			"tag":   caaRecord.Tag,
			"value": caaRecord.Value,
		}
	}
	d.Set("record", recordInterface)

	log.Printf("[DEBUG] Successfully read %d CAA records", len(foundCAARecords))
	return nil
}

// Update updates CAA records
func (s *CAARecordStrategy) Update(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Updating", "CAA", zone, name)

	if d.HasChange("record") {
		// Get old and new configurations
		oldCAARecords, err := s.getOldCAARecords(d)
		if err != nil {
			return err
		}

		newCAARecords, err := s.parseCAARecords(d)
		if err != nil {
			return err
		}

		// Sort both sets for comparison
		sort.Slice(oldCAARecords, func(i, j int) bool {
			return oldCAARecords[i].String() < oldCAARecords[j].String()
		})
		sort.Slice(newCAARecords, func(i, j int) bool {
			return newCAARecords[i].String() < newCAARecords[j].String()
		})

		// Find records to remove
		recordsToRemove := []CAARecord{}
		for _, oldRecord := range oldCAARecords {
			found := false
			for _, newRecord := range newCAARecords {
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
		recordsToAdd := []CAARecord{}
		for _, newRecord := range newCAARecords {
			found := false
			for _, oldRecord := range oldCAARecords {
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
			log.Printf("[DEBUG] Removing CAA record: %s -> %d %s %s", name,
				record.Flag, record.Tag, record.Value)
			response, err := c.RemoveCAARecord(zone, name, record.Value, &record.Flag, &record.Tag)
			if err != nil {
				return fmt.Errorf("failed to remove CAA record %s: %w", record.Value, err)
			}

			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to remove CAA record %s: %w", record.Value, err)
			}
		}

		// Add new records
		for _, record := range recordsToAdd {
			log.Printf("[DEBUG] Adding CAA record: %s -> %d %s %s", name,
				record.Flag, record.Tag, record.Value)
			response, err := c.AddCAARecord(zone, name, record.Value, &record.Flag, &record.Tag)
			if err != nil {
				return fmt.Errorf("failed to add CAA record %s: %w", record.Value, err)
			}

			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to add CAA record %s: %w", record.Value, err)
			}
		}

		// Invalidate cache after updates
		c.InvalidateZoneCache(zone)
	}

	return s.Read(meta, d)
}

// getOldCAARecords reconstructs old CAA records from the change data
func (s *CAARecordStrategy) getOldCAARecords(d *schema.ResourceData) ([]CAARecord, error) {
	old, _ := d.GetChange("record")
	oldRecordList := old.([]interface{})

	var caaRecords []CAARecord
	for _, recordInterface := range oldRecordList {
		recordMap := recordInterface.(map[string]interface{})

		flag := recordMap["flag"].(int)
		tag := recordMap["tag"].(string)
		value := recordMap["value"].(string)

		caaRecord := CAARecord{
			Flag:  flag,
			Tag:   tag,
			Value: value,
		}

		caaRecords = append(caaRecords, caaRecord)
	}

	return caaRecords, nil
}

// Delete deletes CAA records
func (s *CAARecordStrategy) Delete(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)

	caaRecords, err := s.parseCAARecords(d)
	if err != nil {
		return err
	}

	s.LogResourceOperation("Deleting", "CAA", zone, name)

	// Remove each CAA record
	for _, caaRecord := range caaRecords {
		log.Printf("[DEBUG] Removing CAA record: %s -> %d %s %s", name,
			caaRecord.Flag, caaRecord.Tag, caaRecord.Value)
		response, err := c.RemoveCAARecord(zone, name, caaRecord.Value, &caaRecord.Flag, &caaRecord.Tag)
		if err != nil {
			return fmt.Errorf("failed to delete CAA record %s: %w", caaRecord.Value, err)
		}

		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to delete CAA record %s: %w", caaRecord.Value, err)
		}
	}

	// Invalidate cache after deletion
	c.InvalidateZoneCache(zone)

	return nil
}

// Import imports an existing CAA record
func (s *CAARecordStrategy) Import(meta interface{}, d *schema.ResourceData) error {
	// Parse the import ID using the common format
	zone, name, err := s.ParseResourceID(d.Id())
	if err != nil {
		return err
	}

	d.Set("zone", zone)
	d.Set("name", name)

	return s.Read(meta, d)
}
