package base

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// CommonOperations provides shared functionality for all DNS record types
type CommonOperations struct{}

// SetResourceID sets a stable resource ID for the DNS record
func (c *CommonOperations) SetResourceID(d *schema.ResourceData, zone, name, recordType string) {
	d.SetId(fmt.Sprintf("%s/%s", zone, name))
}

// ParseResourceID parses a resource ID into its components
func (c *CommonOperations) ParseResourceID(id string) (zone, name string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid resource ID format: %s", id)
	}
	return parts[0], parts[1], nil
}

// SetCommonAttributes sets the common attributes for a DNS record
func (c *CommonOperations) SetCommonAttributes(d *schema.ResourceData, zone, name, recordType string, records []interface{}) {
	d.Set("zone", zone)
	d.Set("name", name)
	d.Set("type", recordType)
	d.Set("records", records)
}

// AddTrailingDot adds a trailing dot to domain names if not present
func (c *CommonOperations) AddTrailingDot(domain string) string {
	if !strings.HasSuffix(domain, ".") {
		return domain + "."
	}
	return domain
}

// NormalizeDomain removes trailing dots from domain names for consistent comparison
func (c *CommonOperations) NormalizeDomain(domain string) string {
	return strings.TrimSuffix(domain, ".")
}

// ValidateRecords validates that records list is not empty
func (c *CommonOperations) ValidateRecords(records []interface{}) error {
	if len(records) == 0 {
		return fmt.Errorf("at least one record must be specified")
	}
	return nil
}

// LogResourceOperation logs resource operations for debugging
func (c *CommonOperations) LogResourceOperation(operation, recordType, zone, name string) {
	log.Printf("[INFO] %s %s record: %s.%s", operation, recordType, name, zone)
}

// HandleAPIError handles API errors with proper context
func (c *CommonOperations) HandleAPIError(err error, operation string) error {
	if err != nil {
		return fmt.Errorf("API error during %s operation: %w", operation, err)
	}
	return nil
}

// GetStringSlice converts interface slice to string slice
func (c *CommonOperations) GetStringSlice(records []interface{}) []string {
	result := make([]string, len(records))
	for i, record := range records {
		result[i] = record.(string)
	}
	return result
}

// SetStringSlice converts string slice to interface slice
func (c *CommonOperations) SetStringSlice(strings []string) []interface{} {
	result := make([]interface{}, len(strings))
	for i, str := range strings {
		result[i] = str
	}
	return result
}

// OrderRecordsByConfiguration orders found records according to the configuration order
func (c *CommonOperations) OrderRecordsByConfiguration(foundRecords []string, configRecords []interface{}) []string {
	if len(configRecords) == 0 || len(foundRecords) == 0 {
		return foundRecords
	}

	// Create a map of found records for quick lookup
	foundMap := make(map[string]bool)
	for _, r := range foundRecords {
		foundMap[r] = true
	}

	// Create ordered result based on configuration order
	orderedRecords := make([]string, 0, len(configRecords))
	for _, configRecord := range configRecords {
		configStr := configRecord.(string)
		if foundMap[configStr] {
			orderedRecords = append(orderedRecords, configStr)
		}
	}

	// Add any remaining found records that weren't in config order
	for _, found := range foundRecords {
		foundInOrdered := false
		for _, ordered := range orderedRecords {
			if ordered == found {
				foundInOrdered = true
				break
			}
		}
		if !foundInOrdered {
			orderedRecords = append(orderedRecords, found)
		}
	}

	return orderedRecords
}

// InvalidateZoneCache invalidates the zone cache for a specific zone
func (c *CommonOperations) InvalidateZoneCache(client interface{}, zone string) {
	if cachedClient, ok := client.(interface {
		InvalidateZoneCache(zone string)
	}); ok {
		cachedClient.InvalidateZoneCache(zone)
	}
}

// ClearZoneCache clears all zone caches
func (c *CommonOperations) ClearZoneCache(client interface{}) {
	if cachedClient, ok := client.(interface {
		ClearZoneCache()
	}); ok {
		cachedClient.ClearZoneCache()
	}
}

// APIErrorResponse represents an error response from the Reg.ru API
type APIErrorResponse struct {
	Answer struct {
		Domains []struct {
			Dname       string `json:"dname"`
			Result      string `json:"result"`
			ErrorCode   string `json:"error_code"`
			ErrorText   string `json:"error_text"`
			ErrorParams struct {
				ConflictingRecords []struct {
					Data    string `json:"data"`
					Rectype string `json:"rectype"`
					Subname string `json:"subdomain"`
				} `json:"conflicting_records"`
				RecordToAdd struct {
					Data    string `json:"data"`
					Rectype string `json:"rectype"`
					Subname string `json:"subdomain"`
				} `json:"record_to_add"`
			} `json:"error_params"`
		} `json:"domains"`
	} `json:"answer"`
	Result string `json:"result"`
}

// CheckAPIResponseForErrors checks if the API response contains errors
func CheckAPIResponseForErrors(response []byte) error {
	var apiResponse APIErrorResponse
	if err := json.Unmarshal(response, &apiResponse); err != nil {
		// If we can't parse the response, assume it's not an error
		return nil
	}

	// Check if the top-level result indicates an error
	if apiResponse.Result == "error" {
		var errorMessages []string

		for _, domain := range apiResponse.Answer.Domains {
			if domain.Result == "error" {
				errorMsg := fmt.Sprintf("Domain %s: %s", domain.Dname, domain.ErrorText)
				if domain.ErrorCode != "" {
					errorMsg += fmt.Sprintf(" (Error Code: %s)", domain.ErrorCode)
				}
				errorMessages = append(errorMessages, errorMsg)
			}
		}

		if len(errorMessages) > 0 {
			return fmt.Errorf("API operation failed: %s", strings.Join(errorMessages, "; "))
		}
	}

	// Check individual domain results
	for _, domain := range apiResponse.Answer.Domains {
		if domain.Result == "error" {
			errorMsg := fmt.Sprintf("Domain %s: %s", domain.Dname, domain.ErrorText)
			if domain.ErrorCode != "" {
				errorMsg += fmt.Sprintf(" (Error Code: %s)", domain.ErrorCode)
			}
			return fmt.Errorf("API operation failed: %s", errorMsg)
		}
	}

	return nil
}

// RecordsListDiffSuppressFunc provides a universal diff suppression function for record lists
// It compares records as sets, ignoring order differences
func RecordsListDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	// Safety check
	if d == nil {
		return false
	}

	// During resource creation, don't suppress diffs
	if d.Id() == "" {
		return false
	}

	// Only process list elements (records.0, records.1, etc.), not other keys
	if !strings.HasPrefix(k, "records.") {
		return false
	}

	// Use GetChange to get the actual old and new values
	oldInterface, newInterface := d.GetChange("records")

	// Safety checks
	if oldInterface == nil || newInterface == nil {
		return false
	}

	oldRecords, oldOk := oldInterface.([]interface{})
	newRecords, newOk := newInterface.([]interface{})

	if !oldOk || !newOk {
		return false
	}

	// Convert to string slices
	oldStrs := make([]string, 0, len(oldRecords))
	for _, v := range oldRecords {
		if v != nil {
			if str, ok := v.(string); ok {
				oldStrs = append(oldStrs, str)
			}
		}
	}

	newStrs := make([]string, 0, len(newRecords))
	for _, v := range newRecords {
		if v != nil {
			if str, ok := v.(string); ok {
				newStrs = append(newStrs, str)
			}
		}
	}

	log.Printf("[DEBUG] RecordsListDiffSuppressFunc: Comparing for %s: oldStrs=%v vs newStrs=%v", k, oldStrs, newStrs)

	// If different lengths, definitely different
	if len(oldStrs) != len(newStrs) {
		log.Printf("[DEBUG] RecordsListDiffSuppressFunc: Different lengths, not suppressing")
		return false
	}

	// Create sets for comparison
	oldSet := make(map[string]int)
	newSet := make(map[string]int)

	for _, str := range oldStrs {
		oldSet[str]++
	}
	for _, str := range newStrs {
		newSet[str]++
	}

	// Compare sets - if they're identical, this is just an order change
	if len(oldSet) != len(newSet) {
		log.Printf("[DEBUG] RecordsListDiffSuppressFunc: Different unique values, not suppressing")
		return false
	}

	for str, count := range oldSet {
		if newSet[str] != count {
			log.Printf("[DEBUG] RecordsListDiffSuppressFunc: Different counts for %s (%d vs %d), not suppressing", str, count, newSet[str])
			return false
		}
	}

	// Records are the same when treated as sets - suppress the diff
	log.Printf("[DEBUG] RecordsListDiffSuppressFunc: Suppressing order-only diff for %s (sets are identical)", k)
	return true
}
