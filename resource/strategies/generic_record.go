package strategies

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"terraform-provider-regru/resource/base"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// RecordPreprocessor defines a function to preprocess record values
type RecordPreprocessor func(string) string

// RecordValidator defines a function to validate record values
type RecordValidator func([]interface{}) error

// GenericRecordStrategy implements a reusable strategy for simple DNS record types
type GenericRecordStrategy struct {
	base.BaseStrategy
	recordType   string
	preprocessor RecordPreprocessor
	validator    RecordValidator
}

// NewGenericRecordStrategy creates a new generic record strategy
func NewGenericRecordStrategy(recordType string, preprocessor RecordPreprocessor, validator RecordValidator) *GenericRecordStrategy {
	if preprocessor == nil {
		preprocessor = func(s string) string { return s } // No-op preprocessor
	}
	if validator == nil {
		validator = func(records []interface{}) error {
			if len(records) == 0 {
				return fmt.Errorf("at least one %s record must be specified", recordType)
			}
			return nil
		}
	}

	return &GenericRecordStrategy{
		BaseStrategy: base.BaseStrategy{
			CommonRecord: base.CommonRecord{RecordType: recordType},
		},
		recordType:   recordType,
		preprocessor: preprocessor,
		validator:    validator,
	}
}

// Create creates DNS records using the generic pattern
func (s *GenericRecordStrategy) Create(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)
	records := s.GetRecords(d)

	log.Printf("[DEBUG] Creating %s records for %s.%s: %v", s.recordType, name, zone, records)

	// Convert to string slice and apply preprocessing
	recordStrings := make([]string, len(records))
	for i, record := range records {
		recordStr := record.(string)
		recordStrings[i] = s.preprocessor(recordStr)
	}
	sort.Strings(recordStrings)
	log.Printf("[DEBUG] Sorted %s records for creation: %v", s.recordType, recordStrings)

	s.LogResourceOperation("Creating", s.recordType, zone, name)

	// Validate records
	if err := s.validator(records); err != nil {
		return err
	}

	// Add each record
	for _, recordStr := range recordStrings {
		log.Printf("[DEBUG] Adding %s record: %s.%s -> %s", s.recordType, name, zone, recordStr)
		response, err := c.AddRecord(s.recordType, zone, name, recordStr, nil)
		if err != nil {
			return fmt.Errorf("failed to create %s record %s: %w", s.recordType, recordStr, err)
		}

		// Check API response for errors
		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to create %s record %s: %w", s.recordType, recordStr, err)
		}
	}

	// Set resource ID
	s.SetResourceID(d, zone, name, s.recordType)

	return s.Read(meta, d)
}

// Read reads DNS records using the generic pattern
func (s *GenericRecordStrategy) Read(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Reading", s.recordType, zone, name)

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

	// Find records of our type
	var foundRecords []string
	for _, domain := range zoneResponse.Answer.Domains {
		log.Printf("[DEBUG] Processing domain: %s, records: %d", domain.Dname, len(domain.Rrs))
		for _, record := range domain.Rrs {
			log.Printf("[DEBUG] Record: type=%s, subname=%s, content=%s",
				record.Rectype, record.Subname, record.Content)

			if record.Rectype == s.recordType && record.Subname == name {
				// Apply preprocessing to normalize the content
				normalizedContent := s.preprocessor(record.Content)
				foundRecords = append(foundRecords, normalizedContent)
			}
		}
	}

	if len(foundRecords) == 0 {
		log.Printf("[DEBUG] No %s records found for %s.%s", s.recordType, name, zone)
		// No records found, mark as deleted
		d.SetId("")
		return nil
	}

	// Sort records for consistent state
	sort.Strings(foundRecords)
	log.Printf("[DEBUG] Sorted %s records: %v", s.recordType, foundRecords)

	// Set the data
	d.Set("zone", zone)
	d.Set("name", name)

	// Convert to interface slice for Terraform
	recordsInterface := make([]interface{}, len(foundRecords))
	for i, record := range foundRecords {
		recordsInterface[i] = record
	}
	d.Set("records", recordsInterface)

	log.Printf("[DEBUG] Successfully read %d %s records", len(foundRecords), s.recordType)
	return nil
}

// Update updates DNS records using the generic pattern
func (s *GenericRecordStrategy) Update(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Updating", s.recordType, zone, name)

	if d.HasChange("records") {
		old, new := d.GetChange("records")
		oldRecords := old.([]interface{})
		newRecords := new.([]interface{})

		// Apply preprocessing and sort both sets
		oldRecordsStr := make([]string, len(oldRecords))
		for i, record := range oldRecords {
			oldRecordsStr[i] = s.preprocessor(record.(string))
		}
		sort.Strings(oldRecordsStr)

		newRecordsStr := make([]string, len(newRecords))
		for i, record := range newRecords {
			newRecordsStr[i] = s.preprocessor(record.(string))
		}
		sort.Strings(newRecordsStr)

		// Find records to remove
		recordsToRemove := []string{}
		for _, oldRecord := range oldRecordsStr {
			found := false
			for _, newRecord := range newRecordsStr {
				if oldRecord == newRecord {
					found = true
					break
				}
			}
			if !found {
				recordsToRemove = append(recordsToRemove, oldRecord)
			}
		}

		// Find records to add
		recordsToAdd := []string{}
		for _, newRecord := range newRecordsStr {
			found := false
			for _, oldRecord := range oldRecordsStr {
				if newRecord == oldRecord {
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
			log.Printf("[DEBUG] Removing %s record: %s -> %s", s.recordType, name, record)
			response, err := c.RemoveRecord(zone, name, s.recordType, record, nil)
			if err != nil {
				return fmt.Errorf("failed to remove %s record %s: %w", s.recordType, record, err)
			}

			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to remove %s record %s: %w", s.recordType, record, err)
			}
		}

		// Add new records
		for _, record := range recordsToAdd {
			log.Printf("[DEBUG] Adding %s record: %s -> %s", s.recordType, name, record)
			response, err := c.AddRecord(s.recordType, zone, name, record, nil)
			if err != nil {
				return fmt.Errorf("failed to add %s record %s: %w", s.recordType, record, err)
			}

			if err := base.CheckAPIResponseForErrors(response); err != nil {
				return fmt.Errorf("failed to add %s record %s: %w", s.recordType, record, err)
			}
		}

		// Invalidate cache after updates
		c.InvalidateZoneCache(zone)
	}

	return s.Read(meta, d)
}

// Delete deletes DNS records using the generic pattern
func (s *GenericRecordStrategy) Delete(meta interface{}, d *schema.ResourceData) error {
	c := meta.(base.CachedClientInterface)
	zone := s.GetZone(d)
	name := s.GetName(d)
	records := s.GetRecords(d)

	s.LogResourceOperation("Deleting", s.recordType, zone, name)

	// Remove each record
	for _, record := range records {
		recordStr := s.preprocessor(record.(string))
		log.Printf("[DEBUG] Removing %s record: %s -> %s", s.recordType, name, recordStr)
		response, err := c.RemoveRecord(zone, name, s.recordType, recordStr, nil)
		if err != nil {
			return fmt.Errorf("failed to delete %s record %s: %w", s.recordType, recordStr, err)
		}

		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to delete %s record %s: %w", s.recordType, recordStr, err)
		}
	}

	// Invalidate cache after deletion
	c.InvalidateZoneCache(zone)

	return nil
}

// Import imports an existing DNS record using the generic pattern
func (s *GenericRecordStrategy) Import(meta interface{}, d *schema.ResourceData) error {
	zone, name, err := s.ParseResourceID(d.Id())
	if err != nil {
		return err
	}

	d.Set("zone", zone)
	d.Set("name", name)

	return s.Read(meta, d)
}

// Helper functions to create common preprocessors and validators

// NoOpPreprocessor returns the input unchanged
func NoOpPreprocessor(input string) string {
	return input
}

// NormalizeDomainPreprocessor removes trailing dots from domains
func NormalizeDomainPreprocessor(input string) string {
	ops := &base.CommonOperations{}
	return ops.NormalizeDomain(input)
}

// AddTrailingDotPreprocessor adds trailing dots to domains
func AddTrailingDotPreprocessor(input string) string {
	ops := &base.CommonOperations{}
	return ops.AddTrailingDot(input)
}

// DefaultRecordValidator validates that at least one record is provided
func DefaultRecordValidator(recordType string) RecordValidator {
	return func(records []interface{}) error {
		if len(records) == 0 {
			return fmt.Errorf("at least one %s record must be specified", recordType)
		}
		return nil
	}
}
