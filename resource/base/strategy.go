package base

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// StrategyFactory creates the appropriate strategy for a given record type
type StrategyFactory struct {
	strategies map[string]RecordTypeStrategy
}

// NewStrategyFactory creates a new strategy factory
func NewStrategyFactory() *StrategyFactory {
	factory := &StrategyFactory{
		strategies: make(map[string]RecordTypeStrategy),
	}
	return factory
}

// RegisterStrategy registers a strategy for a specific record type
func (f *StrategyFactory) RegisterStrategy(recordType string, strategy RecordTypeStrategy) {
	f.strategies[recordType] = strategy
}

// GetStrategy returns the strategy for a given record type
func (f *StrategyFactory) GetStrategy(recordType string) (RecordTypeStrategy, error) {
	strategy, exists := f.strategies[recordType]
	if !exists {
		return nil, fmt.Errorf("no strategy found for record type: %s", recordType)
	}
	return strategy, nil
}

// BaseStrategy provides common functionality for all record type strategies
type BaseStrategy struct {
	CommonRecord
	CommonOperations
}

// Create provides a default create implementation
func (b *BaseStrategy) Create(client interface{}, d *schema.ResourceData) error {
	return fmt.Errorf("create operation not implemented for record type: %s", b.RecordType)
}

// Read provides a default read implementation
func (b *BaseStrategy) Read(client interface{}, d *schema.ResourceData) error {
	return fmt.Errorf("read operation not implemented for record type: %s", b.RecordType)
}

// Update provides a default update implementation
func (b *BaseStrategy) Update(client interface{}, d *schema.ResourceData) error {
	return fmt.Errorf("update operation not implemented for record type: %s", b.RecordType)
}

// Delete provides a default delete implementation
func (b *BaseStrategy) Delete(client interface{}, d *schema.ResourceData) error {
	return fmt.Errorf("delete operation not implemented for record type: %s", b.RecordType)
}

// Import provides a default import implementation
func (b *BaseStrategy) Import(client interface{}, d *schema.ResourceData) error {
	return fmt.Errorf("import operation not implemented for record type: %s", b.RecordType)
}

// GetCRUDFunctions returns the CRUD functions for a resource using the strategy pattern
func GetCRUDFunctions(recordType string) map[string]interface{} {
	factory := NewStrategyFactory()
	strategy, err := factory.GetStrategy(recordType)
	if err != nil {
		panic(fmt.Sprintf("failed to get strategy for %s: %v", recordType, err))
	}
	
	return map[string]interface{}{
		"Create": func(d *schema.ResourceData, meta interface{}) error {
			return strategy.Create(meta, d)
		},
		"Read": func(d *schema.ResourceData, meta interface{}) error {
			return strategy.Read(meta, d)
		},
		"Update": func(d *schema.ResourceData, meta interface{}) error {
			return strategy.Update(meta, d)
		},
		"Delete": func(d *schema.ResourceData, meta interface{}) error {
			return strategy.Delete(meta, d)
		},
		"Importer": &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				err := strategy.Import(meta, d)
				if err != nil {
					return nil, err
				}
				return []*schema.ResourceData{d}, nil
			},
		},
	}
} 