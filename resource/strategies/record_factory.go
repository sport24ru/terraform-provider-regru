package strategies

import (
	"terraform-provider-regru/resource/base"
)

// Record type strategy constructors using the generic strategy

// NewARecordStrategy creates a new A record strategy
func NewARecordStrategy() *GenericRecordStrategy {
	return NewGenericRecordStrategy(
		"A",
		NoOpPreprocessor, // A records don't need preprocessing
		DefaultRecordValidator("A"),
	)
}

// NewAAAARecordStrategy creates a new AAAA record strategy
func NewAAAARecordStrategy() *GenericRecordStrategy {
	return NewGenericRecordStrategy(
		"AAAA",
		NoOpPreprocessor, // AAAA records don't need preprocessing
		DefaultRecordValidator("AAAA"),
	)
}

// NewTXTRecordStrategy creates a new TXT record strategy
func NewTXTRecordStrategy() *GenericRecordStrategy {
	return NewGenericRecordStrategy(
		"TXT",
		NoOpPreprocessor, // TXT records don't need preprocessing
		DefaultRecordValidator("TXT"),
	)
}

// NewNSRecordStrategy creates a new NS record strategy
// This is now implemented in ns_record.go with custom logic

// NewCNAMERecordStrategy creates a new CNAME record strategy
// This is now implemented in cname_record.go with custom logic

// Complex record type strategy constructors (these use their own specific strategies)

// NewMXRecordStrategy creates a new MX record strategy
func NewMXRecordStrategy() *MXRecordStrategy {
	return &MXRecordStrategy{}
}

// NewSRVRecordStrategy creates a new SRV record strategy (already defined in srv_record.go)

// NewCAARecordStrategy creates a new CAA record strategy (already defined in caa_record.go)

// Interface compliance check - ensure generic strategy implements the interface
var _ base.RecordTypeStrategy = (*GenericRecordStrategy)(nil)
