package resource

import (
	"fmt"
	"log"
	"terraform-provider-regru/client"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceDnsRecord возвращает ресурс DNS записи
func ResourceDnsRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceDnsRecordCreate,
		Read:   resourceDnsRecordRead,
		Delete: resourceDnsRecordDelete,

		Schema: map[string]*schema.Schema{
			"zone": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Zone name",
				ForceNew:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Record name",
				ForceNew:    true,
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Record type (e.g., A, CNAME)",
				ForceNew:    true,
			},
			"record": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Record value",
				ForceNew:    true,
			},
			"priority": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Priority for MX and NS records (default is 10)",
				Default:     10, // Значение по умолчанию, если приоритет не указан
				ForceNew:    true,
			},
		},
	}
}

func resourceDnsRecordCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(*client.Client)

	// Получение данных из Terraform
	zone := d.Get("zone").(string)
	name := d.Get("name").(string)
	record := d.Get("record").(string)
	recordType := d.Get("type").(string)
	priority := 0

	// Если тип записи требует приоритета (например, MX), считываем его
	if recordType == "MX" || recordType == "NS" {
		if v, ok := d.GetOk("priority"); ok {
			priority = v.(int)
		}
	}

	// Вызов AddRecord с новыми параметрами
	_, err := c.AddRecord(recordType, zone, name, record, priority)
	if err != nil {
		return fmt.Errorf("failed to create DNS record: %w", err)
	}

	// Установка ID ресурса
	d.SetId(fmt.Sprintf("%s/%s/%s", zone, name, recordType))
	return nil
}

func resourceDnsRecordRead(d *schema.ResourceData, m interface{}) error {
	// Реализация чтения данных
	return nil
}

func resourceDnsRecordDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(*client.Client)
	zone := d.Get("zone").(string)
	name := d.Get("name").(string)
	recordType := d.Get("type").(string)
	record := d.Get("record").(string)

	log.Printf("[DEBUG] Deleting DNS record: zone=%s, name=%s, type=%s, record=%s", zone, name, recordType, record)

	// Добавляем проверку для MX и NS записей, чтобы учесть приоритет
	priority := 0
	if recordType == "MX" || recordType == "NS" {
		if v, ok := d.GetOk("priority"); ok {
			priority = v.(int)
		}
	}

	// Для MX и NS добавляем точку в конце записи, если она отсутствует
	if (recordType == "MX" || recordType == "NS") && record[len(record)-1] != '.' {
		record = record + "."
	}

	// Попытка удалить запись
	_, err := c.RemoveRecord(zone, name, recordType, record, priority)
	if err != nil {
		// Если ошибка "RR_NOT_FOUND", то игнорируем её, т.к. запись может быть уже удалена
		if err.Error() == "RR_NOT_FOUND" {
			log.Printf("[INFO] DNS record not found, skipping deletion.")
			return nil
		}
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}

	// Убираем ID, так как запись удалена
	d.SetId("")
	return nil
}
