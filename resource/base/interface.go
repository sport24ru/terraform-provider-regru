package base

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// DNSRecord represents a DNS record from the API response
type DNSRecord struct {
	Subname string `json:"subname"`
	Rectype string `json:"rectype"`
	Content string `json:"content"`
	Prio    int    `json:"prio"`
	State   string `json:"state"`
	Weight  int    `json:"weight"`
	Port    int    `json:"port"`
	Flag    int    `json:"flag"`
	Tag     string `json:"tag"`
}

// DNSZoneResponse represents the API response for zone records
type DNSZoneResponse struct {
	Result string `json:"result"`
	Answer struct {
		Domains []struct {
			Dname string      `json:"dname"`
			Rrs   []DNSRecord `json:"rrs"`
		} `json:"domains"`
	} `json:"answer"`
}

// DNSRecordResource defines the interface that all DNS record resources must implement
type DNSRecordResource interface {
	// Core attributes
	GetZone(d *schema.ResourceData) string
	GetName(d *schema.ResourceData) string
	GetType() string
	GetRecords(d *schema.ResourceData) []interface{}
	
	// Optional attributes for specific record types
	GetPriority(d *schema.ResourceData) *int
	GetWeight(d *schema.ResourceData) *int
	GetPort(d *schema.ResourceData) *int
	GetFlag(d *schema.ResourceData) *int
	GetTag(d *schema.ResourceData) *string
	
	// Resource-specific operations
	Create(client interface{}, d *schema.ResourceData) error
	Read(client interface{}, d *schema.ResourceData) error
	Update(client interface{}, d *schema.ResourceData) error
	Delete(client interface{}, d *schema.ResourceData) error
	Import(client interface{}, d *schema.ResourceData) error
}

// RecordTypeStrategy defines the strategy pattern for different record types
type RecordTypeStrategy interface {
	Create(client interface{}, d *schema.ResourceData) error
	Read(client interface{}, d *schema.ResourceData) error
	Update(client interface{}, d *schema.ResourceData) error
	Delete(client interface{}, d *schema.ResourceData) error
	Import(client interface{}, d *schema.ResourceData) error
}

// CommonRecord provides default implementations for common record operations
type CommonRecord struct {
	RecordType string
}

func (c *CommonRecord) GetType() string {
	return c.RecordType
}

func (c *CommonRecord) GetZone(d *schema.ResourceData) string {
	return d.Get("zone").(string)
}

func (c *CommonRecord) GetName(d *schema.ResourceData) string {
	return d.Get("name").(string)
}

func (c *CommonRecord) GetRecords(d *schema.ResourceData) []interface{} {
	if v, ok := d.GetOk("records"); ok {
		return v.([]interface{})
	}
	return []interface{}{}
}

func (c *CommonRecord) GetPriority(d *schema.ResourceData) *int {
	if v, ok := d.GetOk("priority"); ok {
		priority := v.(int)
		return &priority
	}
	return nil
}

func (c *CommonRecord) GetWeight(d *schema.ResourceData) *int {
	if v, ok := d.GetOk("weight"); ok {
		weight := v.(int)
		return &weight
	}
	return nil
}

func (c *CommonRecord) GetPort(d *schema.ResourceData) *int {
	if v, ok := d.GetOk("port"); ok {
		port := v.(int)
		return &port
	}
	return nil
}

func (c *CommonRecord) GetFlag(d *schema.ResourceData) *int {
	if v, ok := d.GetOk("flag"); ok {
		flag := v.(int)
		return &flag
	}
	return nil
}

func (c *CommonRecord) GetTag(d *schema.ResourceData) *string {
	if v, ok := d.GetOk("tag"); ok {
		tag := v.(string)
		return &tag
	}
	return nil
} 