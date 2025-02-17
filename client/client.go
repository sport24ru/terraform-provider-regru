package client

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// Client структура для работы с API Reg.ru
type Client struct {
	Username string
	Password string
	BaseURL  string
}

// NewClient создает новый экземпляр клиента
func NewClient(username, password string) *Client {
	return &Client{
		Username: username,
		Password: password,
		BaseURL:  "https://api.reg.ru/api/regru2",
	}
}

// doRequest выполняет HTTP POST запрос с form-данными
func (c *Client) doRequest(endpoint string, params url.Values) ([]byte, error) {
	// Добавляем логин и пароль в параметры
	params.Add("username", c.Username)
	params.Add("password", c.Password)

	// Формируем URL
	fullURL := fmt.Sprintf("%s/%s", c.BaseURL, endpoint)

	log.Printf("[DEBUG] Making request to: %s", fullURL)
	log.Printf("[DEBUG] Request params: %s", params.Encode())

	// Выполняем POST запрос
	resp, err := http.PostForm(fullURL, params)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] Response status: %s", resp.Status)

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[DEBUG] Response body: %s", string(body))

	return body, nil
}

func (c *Client) AddRecord(recordType, domainName, subdomain, value string, priority int) ([]byte, error) {
	// Параметры для запроса
	params := url.Values{}
	params.Add("domain_name", domainName)
	params.Add("subdomain", subdomain)
	params.Add("output_content_type", "plain")

	// Выбор эндпоинта и параметров в зависимости от типа записи

	var endpoint string
	switch recordType {
	case "A":
		endpoint = "zone/add_alias"
		params.Add("ipaddr", value)
	case "AAAA":
		endpoint = "zone/add_aaaa"
		params.Add("ipaddr", value)
	case "CNAME":
		endpoint = "zone/add_cname"
		params.Add("canonical_name", value)
	case "MX":
		endpoint = "zone/add_mx"
		params.Add("mail_server", value)
		params.Add("priority", fmt.Sprintf("%d", priority)) // Преобразование приоритета в строку
	case "NS":
		endpoint = "zone/add_ns"
		params.Add("dns_server", value)
		params.Add("priority", fmt.Sprintf("%d", priority))
	case "TXT":
		endpoint = "zone/add_txt"
		params.Add("text", value)
	default:
		// Если тип записи не поддерживается, используем TXT как универсальный
		endpoint = "zone/add_txt"
		params.Add("text", value)
	}

	// Выполнение запроса
	return c.doRequest(endpoint, params)
}

// RemoveRecord удаляет запись
func (c *Client) RemoveRecord(domainName, subdomain, recordType, content string, priority int) ([]byte, error) {
	params := url.Values{}
	params.Add("domain_name", domainName)
	params.Add("subdomain", subdomain)
	params.Add("record_type", recordType)
	params.Add("content", content)
	params.Add("output_content_type", "plain")

	// Добавляем приоритет для MX и NS записей
	if recordType == "MX" || recordType == "NS" {
		params.Add("priority", fmt.Sprintf("%d", priority))
	}

	return c.doRequest("zone/remove_record", params)
}

// GetRecords получает все записи для зоны
func (c *Client) GetRecords(domainName string) ([]byte, error) {
	params := url.Values{}
	params.Add("dname", domainName)

	return c.doRequest("zone/get_resource_records", params)
}
