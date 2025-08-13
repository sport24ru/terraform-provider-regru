package strategies

import (
	"encoding/json"
	"fmt"
	"terraform-provider-regru/resource/base"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// CNAMERecordStrategy implements the strategy for CNAME records
type CNAMERecordStrategy struct {
	base.BaseStrategy
}

// NewCNAMERecordStrategy creates a new CNAME record strategy
func NewCNAMERecordStrategy() *CNAMERecordStrategy {
	return &CNAMERecordStrategy{}
}

// GetRecords returns the CNAME record from the resource data
func (s *CNAMERecordStrategy) GetRecords(d *schema.ResourceData) []interface{} {
	cname := d.Get("cname").(string)
	return []interface{}{cname}
}

// SetResourceID sets a stable resource ID for the CNAME record
func (s *CNAMERecordStrategy) SetResourceID(d *schema.ResourceData, zone, name, recordType string) {
	d.SetId(fmt.Sprintf("%s/%s", zone, name))
}

// ValidateRecords validates CNAME records
func (s *CNAMERecordStrategy) ValidateRecords(records []interface{}) error {
	if len(records) != 1 {
		return fmt.Errorf("CNAME record must have exactly one target")
	}

	record := records[0].(string)
	if record == "" {
		return fmt.Errorf("CNAME record cannot be empty")
	}

	return nil
}

// Create creates CNAME records
func (s *CNAMERecordStrategy) Create(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for CNAME record creation")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)
	cname := d.Get("cname").(string)

	s.LogResourceOperation("Creating", "CNAME", zone, name)

	// For CNAME records, we need to add trailing dots for domain names
	apiRecord := s.AddTrailingDot(cname)
	response, err := c.AddRecord("CNAME", zone, name, apiRecord, nil)
	if err != nil {
		return fmt.Errorf("failed to create CNAME record: %w", err)
	}

	// Check API response for errors
	if err := base.CheckAPIResponseForErrors(response); err != nil {
		return fmt.Errorf("failed to create CNAME record: %w", err)
	}

	s.SetResourceID(d, zone, name, "CNAME")
	c.InvalidateZoneCache(zone)
	return nil
}

// Read reads CNAME records from the API
func (s *CNAMERecordStrategy) Read(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for CNAME record read")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Reading", "CNAME", zone, name)

	response, err := c.GetRecordsWithCache(zone)
	if err != nil {
		return fmt.Errorf("failed to get zone records: %w", err)
	}

	var zoneResponse base.DNSZoneResponse
	if err := json.Unmarshal(response, &zoneResponse); err != nil {
		return fmt.Errorf("failed to parse DNS records response: %w", err)
	}

	// Find CNAME records for this subdomain
	var foundCNAME string
	for _, domain := range zoneResponse.Answer.Domains {
		if domain.Dname == zone {
			for _, rr := range domain.Rrs {
				if rr.Subname == name && rr.Rectype == "CNAME" {
					// Remove trailing dot from content for consistency
					foundCNAME = s.NormalizeDomain(rr.Content)
					break
				}
			}
			break
		}
	}

	if foundCNAME == "" {
		// No record found, mark as deleted
		d.SetId("")
		return nil
	}

	// Set the data
	d.Set("zone", zone)
	d.Set("name", name)
	d.Set("cname", foundCNAME)

	return nil
}

// Update updates CNAME records
func (s *CNAMERecordStrategy) Update(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for CNAME record update")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Updating", "CNAME", zone, name)

	// Get old and new CNAME values
	oldCNAME, newCNAME := d.GetChange("cname")
	oldCNAMEStr := oldCNAME.(string)
	newCNAMEStr := newCNAME.(string)

	// Delete the old record first (required due to DNS CNAME constraints)
	if oldCNAMEStr != "" {
		apiOldRecord := s.AddTrailingDot(oldCNAMEStr)
		response, err := c.RemoveRecord(zone, name, "CNAME", apiOldRecord, nil)
		if err != nil {
			return fmt.Errorf("failed to delete old CNAME record: %w", err)
		}

		// Check API response for errors
		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to delete old CNAME record: %w", err)
		}
	}

	// Add the new record
	if newCNAMEStr != "" {
		apiNewRecord := s.AddTrailingDot(newCNAMEStr)
		response, err := c.AddRecord("CNAME", zone, name, apiNewRecord, nil)
		if err != nil {
			return fmt.Errorf("failed to create new CNAME record: %w", err)
		}

		// Check API response for errors
		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to create new CNAME record: %w", err)
		}
	}

	// Invalidate cache after update
	c.InvalidateZoneCache(zone)
	return nil
}

// Delete deletes CNAME records
func (s *CNAMERecordStrategy) Delete(client interface{}, d *schema.ResourceData) error {
	// Type assert to get the cached client using shared interface
	c, ok := client.(base.CachedClientInterface)
	if !ok {
		return fmt.Errorf("invalid client type for CNAME record deletion")
	}

	zone := s.GetZone(d)
	name := s.GetName(d)

	s.LogResourceOperation("Deleting", "CNAME", zone, name)

	// Get the CNAME value to remove
	cname := d.Get("cname").(string)
	if cname == "" {
		// If no CNAME value, try to get it from the old state
		oldCNAME, _ := d.GetChange("cname")
		cname = oldCNAME.(string)
	}

	if cname != "" {
		// For CNAME records, we need to add trailing dots for domain names
		apiRecord := s.AddTrailingDot(cname)
		response, err := c.RemoveRecord(zone, name, "CNAME", apiRecord, nil)
		if err != nil {
			return fmt.Errorf("failed to delete CNAME record: %w", err)
		}

		// Check API response for errors
		if err := base.CheckAPIResponseForErrors(response); err != nil {
			return fmt.Errorf("failed to delete CNAME record: %w", err)
		}
	}

	// Invalidate cache after deletion
	c.InvalidateZoneCache(zone)
	return nil
}

// Import imports an existing CNAME record
func (s *CNAMERecordStrategy) Import(client interface{}, d *schema.ResourceData) error {
	// Parse the import ID using the common format
	zone, name, err := s.ParseResourceID(d.Id())
	if err != nil {
		return err
	}

	d.Set("zone", zone)
	d.Set("name", name)

	return s.Read(client, d)
}
