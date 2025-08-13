package resources

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"terraform-provider-regru/resource/base"
	"terraform-provider-regru/resource/strategies"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// GenericDiffSuppressFunc provides a unified diff suppression function for nested record blocks
// It can handle different field types and comparison strategies
func GenericDiffSuppressFunc(k, old, new string, d *schema.ResourceData, config DiffSuppressConfig) bool {
	// During resource creation, old records will be empty from state,
	// but we should not suppress the diff during creation
	if d.Id() == "" {
		log.Printf("[DEBUG] GenericDiffSuppressFunc: Not suppressing diff during creation (empty resource ID)")
		return false
	}

	// Handle different field types
	switch config.FieldType {
	case "caa_record":
		return handleCAARecordDiff(k, d)
	case "nested_field":
		return handleNestedFieldDiff(k, d, config.FieldName)
	default:
		return false
	}
}

// DiffSuppressConfig defines the configuration for diff suppression
type DiffSuppressConfig struct {
	FieldType string // "caa_record" or "nested_field"
	FieldName string // Field name within record block (e.g., "servers", "targets")
}

// handleCAARecordDiff handles CAA record diff suppression
func handleCAARecordDiff(k string, d *schema.ResourceData) bool {
	// Add safety checks to prevent crashes during schema validation
	if d == nil {
		log.Printf("[DEBUG] handleCAARecordDiff: ResourceData is nil, not suppressing diff")
		return false
	}

	// Use GetChange to properly get old and new values
	oldRecordsInterface, newRecordsInterface := d.GetChange("record")

	// Check if old records is valid
	if oldRecordsInterface == nil {
		log.Printf("[DEBUG] handleCAARecordDiff: Old records is nil, not suppressing diff")
		return false
	}

	oldCAARecords, ok := oldRecordsInterface.([]interface{})
	if !ok {
		log.Printf("[DEBUG] handleCAARecordDiff: Old records is not []interface{}, not suppressing diff")
		return false
	}

	if len(oldCAARecords) == 0 {
		log.Printf("[DEBUG] handleCAARecordDiff: Not suppressing diff during creation (old records empty)")
		return false
	}

	// Check if new records is valid
	if newRecordsInterface == nil {
		log.Printf("[DEBUG] handleCAARecordDiff: New records is nil, not suppressing diff")
		return false
	}

	newCAARecords, ok := newRecordsInterface.([]interface{})
	if !ok {
		log.Printf("[DEBUG] handleCAARecordDiff: New records is not []interface{}, not suppressing diff")
		return false
	}

	// Convert old records to sortable strings with safety checks
	oldStrs := make([]string, 0, len(oldCAARecords))
	for _, recordInterface := range oldCAARecords {
		if recordInterface == nil {
			continue
		}

		recordMap, ok := recordInterface.(map[string]interface{})
		if !ok {
			log.Printf("[DEBUG] handleCAARecordDiff: Record is not map[string]interface{}, skipping")
			continue
		}

		flag, flagOk := recordMap["flag"].(int)
		tag, tagOk := recordMap["tag"].(string)
		value, valueOk := recordMap["value"].(string)

		if !flagOk || !tagOk || !valueOk {
			log.Printf("[DEBUG] handleCAARecordDiff: Invalid record data, skipping")
			continue
		}

		oldStrs = append(oldStrs, fmt.Sprintf("%d_%s_%s", flag, tag, value))
	}

	// Convert new records to sortable strings with safety checks
	newStrs := make([]string, 0, len(newCAARecords))
	for _, recordInterface := range newCAARecords {
		if recordInterface == nil {
			continue
		}

		recordMap, ok := recordInterface.(map[string]interface{})
		if !ok {
			log.Printf("[DEBUG] handleCAARecordDiff: New record is not map[string]interface{}, skipping")
			continue
		}

		flag, flagOk := recordMap["flag"].(int)
		tag, tagOk := recordMap["tag"].(string)
		value, valueOk := recordMap["value"].(string)

		if !flagOk || !tagOk || !valueOk {
			log.Printf("[DEBUG] handleCAARecordDiff: Invalid new record data, skipping")
			continue
		}

		newStrs = append(newStrs, fmt.Sprintf("%d_%s_%s", flag, tag, value))
	}

	// If we couldn't parse any records, don't suppress
	if len(oldStrs) == 0 || len(newStrs) == 0 {
		log.Printf("[DEBUG] handleCAARecordDiff: Could not parse records, not suppressing diff")
		return false
	}

	sort.Strings(oldStrs)
	sort.Strings(newStrs)

	if len(oldStrs) != len(newStrs) {
		return false
	}

	for i, oldStr := range oldStrs {
		if oldStr != newStrs[i] {
			return false
		}
	}

	log.Printf("[DEBUG] handleCAARecordDiff: Suppressing order-only diff for %s", k)
	return true
}

// handleNestedFieldDiff handles nested field diff suppression (e.g., servers, targets)
func handleNestedFieldDiff(k string, d *schema.ResourceData, fieldName string) bool {
	// Add safety checks to prevent crashes during schema validation
	if d == nil {
		log.Printf("[DEBUG] handleNestedFieldDiff: ResourceData is nil, not suppressing diff")
		return false
	}

	// Extract the field name from the key (e.g., "record.0.servers" -> "servers")
	parts := strings.Split(k, ".")
	if len(parts) < 3 || parts[2] != fieldName {
		return false
	}

	// Get the current record index from the key
	recordIndexStr := parts[1]
	recordIndex, err := strconv.Atoi(recordIndexStr)
	if err != nil {
		log.Printf("[DEBUG] handleNestedFieldDiff: Invalid record index %s, not suppressing diff", recordIndexStr)
		return false
	}

	// Use GetChange to get actual old vs new values for the entire record block
	oldRecordsInterface, newRecordsInterface := d.GetChange("record")

	// Safety checks
	if oldRecordsInterface == nil || newRecordsInterface == nil {
		log.Printf("[DEBUG] handleNestedFieldDiff: Records change data is nil, not suppressing diff")
		return false
	}

	oldRecords, oldOk := oldRecordsInterface.([]interface{})
	newRecords, newOk := newRecordsInterface.([]interface{})

	if !oldOk || !newOk {
		log.Printf("[DEBUG] handleNestedFieldDiff: Records are not slices, not suppressing diff")
		return false
	}

	// Check bounds
	if recordIndex >= len(oldRecords) || recordIndex >= len(newRecords) {
		log.Printf("[DEBUG] handleNestedFieldDiff: Record index %d out of bounds, not suppressing diff", recordIndex)
		return false
	}

	// Get the specific record blocks
	oldRecord, oldRecordOk := oldRecords[recordIndex].(map[string]interface{})
	newRecord, newRecordOk := newRecords[recordIndex].(map[string]interface{})

	if !oldRecordOk || !newRecordOk {
		log.Printf("[DEBUG] handleNestedFieldDiff: Record blocks are not maps, not suppressing diff")
		return false
	}

	// Get the field lists from old and new record blocks
	oldFieldInterface, oldFieldExists := oldRecord[fieldName]
	newFieldInterface, newFieldExists := newRecord[fieldName]

	if !oldFieldExists || !newFieldExists {
		log.Printf("[DEBUG] handleNestedFieldDiff: Field %s missing in records, not suppressing diff", fieldName)
		return false
	}

	oldField, oldFieldOk := oldFieldInterface.([]interface{})
	newField, newFieldOk := newFieldInterface.([]interface{})

	if !oldFieldOk || !newFieldOk {
		log.Printf("[DEBUG] handleNestedFieldDiff: Field %s is not a slice, not suppressing diff", fieldName)
		return false
	}

	// During resource creation, oldField will be empty from state,
	// but we should not suppress the diff during creation
	if len(oldField) == 0 {
		log.Printf("[DEBUG] handleNestedFieldDiff: Not suppressing diff during creation (old %s empty)", fieldName)
		return false
	}

	// Convert to string slices with safety checks
	oldStrs := make([]string, 0, len(oldField))
	for _, v := range oldField {
		if v == nil {
			continue
		}
		if str, ok := v.(string); ok {
			oldStrs = append(oldStrs, str)
		}
	}

	newStrs := make([]string, 0, len(newField))
	for _, v := range newField {
		if v == nil {
			continue
		}
		if str, ok := v.(string); ok {
			newStrs = append(newStrs, str)
		}
	}

	// If we couldn't parse any values, don't suppress
	if len(oldStrs) == 0 || len(newStrs) == 0 {
		log.Printf("[DEBUG] handleNestedFieldDiff: Could not parse values, not suppressing diff")
		return false
	}

	// Sort both slices for comparison
	sort.Strings(oldStrs)
	sort.Strings(newStrs)

	// Compare sorted slices - suppress diff if they're the same
	if len(oldStrs) != len(newStrs) {
		return false
	}

	for i, oldStr := range oldStrs {
		if oldStr != newStrs[i] {
			return false
		}
	}

	// Fields are the same when sorted - suppress the diff
	log.Printf("[DEBUG] handleNestedFieldDiff: Suppressing order-only diff for %s", k)
	return true
}

// MXServersDiffSuppressFunc compares MX server lists as sets, ignoring order differences
func MXServersDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	config := DiffSuppressConfig{
		FieldType: "nested_field",
		FieldName: "servers",
	}
	return GenericDiffSuppressFunc(k, old, new, d, config)
}

// SRVTargetsDiffSuppressFunc compares SRV target lists as sets, ignoring order differences
func SRVTargetsDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	config := DiffSuppressConfig{
		FieldType: "nested_field",
		FieldName: "targets",
	}
	return GenericDiffSuppressFunc(k, old, new, d, config)
}

// CAARecordsDiffSuppressFunc compares CAA records as sets, ignoring order differences
func CAARecordsDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	config := DiffSuppressConfig{
		FieldType: "caa_record",
		FieldName: "",
	}
	return GenericDiffSuppressFunc(k, old, new, d, config)
}

// NSServersDiffSuppressFunc compares NS server lists as sets, ignoring order differences
func NSServersDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	config := DiffSuppressConfig{
		FieldType: "nested_field",
		FieldName: "servers",
	}
	return GenericDiffSuppressFunc(k, old, new, d, config)
}

// ResourceConfig defines the configuration for creating a DNS record resource
type ResourceConfig struct {
	RecordType      string
	Description     string
	ExtraFields     map[string]*schema.Schema
	StrategyFactory func() interface{} // Returns the strategy for this record type
	UsesGenericCRUD bool               // Whether to use generic CRUD functions
}

// CreateDNSRecordResource creates a Terraform resource for DNS records
func CreateDNSRecordResource(config ResourceConfig) *schema.Resource {
	// Base schema that all DNS records share
	baseSchema := map[string]*schema.Schema{
		"zone": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The DNS zone (domain) for this record",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The name for this record (use @ for root domain)",
		},
	}

	// Add records field for simple record types
	if config.UsesGenericCRUD {
		baseSchema["records"] = &schema.Schema{
			Type:             schema.TypeList,
			Required:         true,
			MinItems:         1,
			Description:      config.Description,
			Elem:             &schema.Schema{Type: schema.TypeString},
			DiffSuppressFunc: base.RecordsListDiffSuppressFunc,
		}
	}

	// Add any extra fields specific to this record type
	for fieldName, fieldSchema := range config.ExtraFields {
		baseSchema[fieldName] = fieldSchema
	}

	// Create CRUD functions
	var createFunc, readFunc, updateFunc, deleteFunc func(d *schema.ResourceData, meta interface{}) error
	var importFunc func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error)

	if config.UsesGenericCRUD {
		// Use generic CRUD functions for simple record types
		createFunc = createGenericCRUDFunc(config.StrategyFactory, "Create")
		readFunc = createGenericCRUDFunc(config.StrategyFactory, "Read")
		updateFunc = createGenericCRUDFunc(config.StrategyFactory, "Update")
		deleteFunc = createGenericCRUDFunc(config.StrategyFactory, "Delete")
		importFunc = createGenericImportFunc(config.StrategyFactory)
	} else {
		// For complex record types, delegate to specific functions
		createFunc = createSpecificCRUDFunc(config.RecordType, config.StrategyFactory, "Create")
		readFunc = createSpecificCRUDFunc(config.RecordType, config.StrategyFactory, "Read")
		updateFunc = createSpecificCRUDFunc(config.RecordType, config.StrategyFactory, "Update")
		deleteFunc = createSpecificCRUDFunc(config.RecordType, config.StrategyFactory, "Delete")
		importFunc = createSpecificImportFunc(config.RecordType, config.StrategyFactory)
	}

	return &schema.Resource{
		Schema:   baseSchema,
		Create:   createFunc,
		Read:     readFunc,
		Update:   updateFunc,
		Delete:   deleteFunc,
		Importer: &schema.ResourceImporter{State: importFunc},
	}
}

// createGenericCRUDFunc creates a generic CRUD function for simple record types
func createGenericCRUDFunc(strategyFactory func() interface{}, operation string) func(d *schema.ResourceData, meta interface{}) error {
	return func(d *schema.ResourceData, meta interface{}) error {
		strategy := strategyFactory()

		switch operation {
		case "Create":
			if s, ok := strategy.(interface {
				Create(interface{}, *schema.ResourceData) error
			}); ok {
				return s.Create(meta, d)
			}
		case "Read":
			if s, ok := strategy.(interface {
				Read(interface{}, *schema.ResourceData) error
			}); ok {
				return s.Read(meta, d)
			}
		case "Update":
			if s, ok := strategy.(interface {
				Update(interface{}, *schema.ResourceData) error
			}); ok {
				return s.Update(meta, d)
			}
		case "Delete":
			if s, ok := strategy.(interface {
				Delete(interface{}, *schema.ResourceData) error
			}); ok {
				return s.Delete(meta, d)
			}
		}

		return fmt.Errorf("operation %s not supported", operation)
	}
}

// createGenericImportFunc creates a generic import function for simple record types
func createGenericImportFunc(strategyFactory func() interface{}) func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	return func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
		strategy := strategyFactory()

		if s, ok := strategy.(interface {
			Import(interface{}, *schema.ResourceData) error
		}); ok {
			err := s.Import(meta, d)
			if err != nil {
				return nil, err
			}
			return []*schema.ResourceData{d}, nil
		}

		return nil, fmt.Errorf("import not supported for this record type")
	}
}

// createSpecificCRUDFunc creates CRUD functions for complex record types
func createSpecificCRUDFunc(recordType string, strategyFactory func() interface{}, operation string) func(d *schema.ResourceData, meta interface{}) error {
	return func(d *schema.ResourceData, meta interface{}) error {
		strategy := strategyFactory()

		switch operation {
		case "Create":
			if s, ok := strategy.(interface {
				Create(interface{}, *schema.ResourceData) error
			}); ok {
				return s.Create(meta, d)
			}
		case "Read":
			if s, ok := strategy.(interface {
				Read(interface{}, *schema.ResourceData) error
			}); ok {
				return s.Read(meta, d)
			}
		case "Update":
			if s, ok := strategy.(interface {
				Update(interface{}, *schema.ResourceData) error
			}); ok {
				return s.Update(meta, d)
			}
		case "Delete":
			if s, ok := strategy.(interface {
				Delete(interface{}, *schema.ResourceData) error
			}); ok {
				return s.Delete(meta, d)
			}
		}

		return fmt.Errorf("operation %s not supported for %s records", operation, recordType)
	}
}

// createSpecificImportFunc creates import functions for complex record types
func createSpecificImportFunc(recordType string, strategyFactory func() interface{}) func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	return func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
		strategy := strategyFactory()

		if s, ok := strategy.(interface {
			Import(interface{}, *schema.ResourceData) error
		}); ok {
			err := s.Import(meta, d)
			if err != nil {
				return nil, err
			}
			return []*schema.ResourceData{d}, nil
		}

		return nil, fmt.Errorf("import not supported for %s records", recordType)
	}
}

// Convenience functions for creating specific DNS record resources

// ResourceDNSARecord creates the A record resource
func ResourceDNSARecord() *schema.Resource {
	return CreateDNSRecordResource(ResourceConfig{
		RecordType:      "A",
		Description:     "List of IPv4 addresses for this A record",
		StrategyFactory: func() interface{} { return strategies.NewARecordStrategy() },
		UsesGenericCRUD: true,
	})
}

// ResourceDNSAAAARecord creates the AAAA record resource
func ResourceDNSAAAARecord() *schema.Resource {
	return CreateDNSRecordResource(ResourceConfig{
		RecordType:      "AAAA",
		Description:     "List of IPv6 addresses for this AAAA record",
		StrategyFactory: func() interface{} { return strategies.NewAAAARecordStrategy() },
		UsesGenericCRUD: true,
	})
}

// ResourceDNSTXTRecord creates the TXT record resource
func ResourceDNSTXTRecord() *schema.Resource {
	return CreateDNSRecordResource(ResourceConfig{
		RecordType:      "TXT",
		Description:     "List of text values for this TXT record",
		StrategyFactory: func() interface{} { return strategies.NewTXTRecordStrategy() },
		UsesGenericCRUD: true,
	})
}

// ResourceDNSNSRecord creates the NS record resource
func ResourceDNSNSRecord() *schema.Resource {
	return CreateDNSRecordResource(ResourceConfig{
		RecordType: "NS",
		ExtraFields: map[string]*schema.Schema{
			"record": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "List of NS record sets with priority and servers",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"priority": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The priority for this NS record set (lower number = higher priority)",
						},
						"servers": {
							Type:             schema.TypeList,
							Required:         true,
							MinItems:         1,
							Description:      "List of name server hostnames for this NS record set",
							Elem:             &schema.Schema{Type: schema.TypeString},
							DiffSuppressFunc: NSServersDiffSuppressFunc,
						},
					},
				},
			},
		},
		StrategyFactory: func() interface{} { return strategies.NewNSRecordStrategy() },
		UsesGenericCRUD: false,
	})
}

// ResourceDNSCNAMERecord creates the CNAME record resource
func ResourceDNSCNAMERecord() *schema.Resource {
	return CreateDNSRecordResource(ResourceConfig{
		RecordType: "CNAME",
		ExtraFields: map[string]*schema.Schema{
			"cname": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The canonical name (target domain) for this CNAME record",
			},
		},
		StrategyFactory: func() interface{} { return strategies.NewCNAMERecordStrategy() },
		UsesGenericCRUD: false,
	})
}

// ResourceDNSMXRecord creates the MX record resource
func ResourceDNSMXRecord() *schema.Resource {
	return CreateDNSRecordResource(ResourceConfig{
		RecordType: "MX",
		ExtraFields: map[string]*schema.Schema{
			"record": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "List of MX record sets with priority and servers",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"priority": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The priority for this MX record set (lower number = higher priority)",
						},
						"servers": {
							Type:             schema.TypeList,
							Required:         true,
							MinItems:         1,
							Description:      "List of mail server hostnames for this MX record set",
							Elem:             &schema.Schema{Type: schema.TypeString},
							DiffSuppressFunc: MXServersDiffSuppressFunc,
						},
					},
				},
			},
		},
		StrategyFactory: func() interface{} { return strategies.NewMXRecordStrategy() },
		UsesGenericCRUD: false,
	})
}

// ResourceDNSSRVRecord creates the SRV record resource
func ResourceDNSSRVRecord() *schema.Resource {
	return CreateDNSRecordResource(ResourceConfig{
		RecordType: "SRV",
		ExtraFields: map[string]*schema.Schema{
			"record": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "List of SRV record sets with priority, weight, port, and targets",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"priority": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The priority for this SRV record set (lower number = higher priority)",
						},
						"weight": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The weight for this SRV record set (used for load balancing within the same priority)",
						},
						"port": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The port number for this SRV record set",
						},
						"targets": {
							Type:             schema.TypeList,
							Required:         true,
							MinItems:         1,
							Description:      "List of target hostnames for this SRV record set",
							Elem:             &schema.Schema{Type: schema.TypeString},
							DiffSuppressFunc: SRVTargetsDiffSuppressFunc,
						},
					},
				},
			},
		},
		StrategyFactory: func() interface{} { return strategies.NewSRVRecordStrategy() },
		UsesGenericCRUD: false,
	})
}

// ResourceDNSCAARecord creates the CAA record resource
func ResourceDNSCAARecord() *schema.Resource {
	return CreateDNSRecordResource(ResourceConfig{
		RecordType: "CAA",
		ExtraFields: map[string]*schema.Schema{
			"record": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "List of CAA records with flag, tag, and value",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"flag": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Flag for CAA records (0 for non-critical, 128 for critical)",
						},
						"tag": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Tag for CAA records (issue, issuewild, iodef)",
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The CAA record value (e.g., domain name or email)",
						},
					},
				},
				DiffSuppressFunc: CAARecordsDiffSuppressFunc,
			},
		},
		StrategyFactory: func() interface{} { return strategies.NewCAARecordStrategy() },
		UsesGenericCRUD: false,
	})
}
