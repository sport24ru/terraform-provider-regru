package provider

import (
	"terraform-provider-regru/client"
	"terraform-provider-regru/resource"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider возвращает провайдер Reg.ru
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "API username for Reg.ru",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "API password for Reg.ru",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"regru_dns_record": resource.ResourceDnsRecord(),
		},
		ConfigureFunc: func(d *schema.ResourceData) (interface{}, error) {
			username := d.Get("username").(string)
			password := d.Get("password").(string)
			return client.NewClient(username, password), nil
		},
	}
}
