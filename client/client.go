package client

import (
	"encoding/json"
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

// APIError represents the error response structure
type APIError struct {
	ErrorCode   string            `json:"error_code"`
	ErrorText   string            `json:"error_text"`
	ErrorParams map[string]string `json:"error_params"`
	Result      string            `json:"result"`
}

// APIResponse represents the full API response structure
type APIResponse struct {
	Answer struct {
		Domains []struct {
			ErrorCode   string            `json:"error_code"`
			ErrorText   string            `json:"error_text"`
			ErrorParams map[string]string `json:"error_params"`
			Result      string            `json:"result"`
		} `json:"domains"`
	} `json:"answer"`
	Result string `json:"result"`
}

// NewClient создает новый экземпляр клиента
func NewClient(username, password string) *Client {
	return &Client{
		Username: username,
		Password: password,
		BaseURL:  "https://api.reg.ru/api/regru2",
	}
}

// formatHumanReadableError creates user-friendly error messages for common API errors
func formatHumanReadableError(errorCode, errorText string, errorParams map[string]string) error {
	// Handle specific error codes with user-friendly messages
	switch errorCode {
	case "ACCESS_DENIED_FROM_IP":
		return fmt.Errorf("Access denied: Your IP address is not authorized to access the Reg.ru API. Please contact Reg.ru support to whitelist your IP address or check your account settings.")
	case "IP_EXCEEDED_ALLOWED_CONNECTION_RATE":
		return fmt.Errorf("Rate limit exceeded: Your IP address has exceeded the allowed connection rate to the Reg.ru API. Please wait a few minutes before making additional requests or contact Reg.ru support if this persists.")
	case "INVALID_USERNAME_OR_PASSWORD":
		return fmt.Errorf("Authentication failed: Invalid username or password. Please check your Reg.ru API credentials.")
	case "DOMAIN_NOT_FOUND":
		return fmt.Errorf("Domain not found: The specified domain does not exist in your account or you don't have access to it.")
	case "RECORD_NOT_FOUND":
		return fmt.Errorf("DNS record not found: The specified DNS record does not exist.")
	case "INVALID_RECORD_TYPE":
		return fmt.Errorf("Invalid record type: The specified DNS record type is not supported or invalid.")
	case "DUPLICATE_RECORD":
		return fmt.Errorf("Duplicate record: A DNS record with the same name and type already exists.")
	case "INVALID_IP_ADDRESS":
		return fmt.Errorf("Invalid IP address: The provided IP address format is incorrect.")
	case "RATE_LIMIT_EXCEEDED":
		return fmt.Errorf("Rate limit exceeded: Too many API requests. Please wait before making additional requests.")
	default:
		// For unknown error codes, provide a detailed error message
		errorMsg := fmt.Sprintf("API Error: %s (Code: %s)", errorText, errorCode)
		if len(errorParams) > 0 {
			errorMsg += fmt.Sprintf(" - Additional info: %v", errorParams)
		}
		return fmt.Errorf(errorMsg)
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

	// Проверяем JSON на наличие ошибки
	// First, try to parse as a direct error response (like ACCESS_DENIED_FROM_IP)
	var directError APIError
	if err := json.Unmarshal(body, &directError); err == nil {
		if directError.Result == "error" {
			return nil, formatHumanReadableError(directError.ErrorCode, directError.ErrorText, directError.ErrorParams)
		}
	}

	// If not a direct error, try to parse as APIResponse
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err == nil {
		// Check if the overall result is error
		if apiResp.Result == "error" {
			// Try to get more specific error information from the response
			if len(apiResp.Answer.Domains) > 0 {
				domain := apiResp.Answer.Domains[0]
				if domain.ErrorCode != "" {
					return nil, formatHumanReadableError(domain.ErrorCode, domain.ErrorText, domain.ErrorParams)
				}
			}
			return nil, fmt.Errorf("API error: overall result is error")
		}

		// Check if any domain has an error
		for _, domain := range apiResp.Answer.Domains {
			if domain.Result == "error" {
				return nil, formatHumanReadableError(domain.ErrorCode, domain.ErrorText, domain.ErrorParams)
			}
		}
	}

	return body, nil
}

func (c *Client) AddRecord(recordType, domainName, subdomain, value string, priority *int) ([]byte, error) {
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
		if priority != nil {
			params.Add("priority", fmt.Sprintf("%d", *priority)) // Преобразование приоритета в строку
		}
	case "NS":
		endpoint = "zone/add_ns"
		params.Add("dns_server", value)
		if priority != nil {
			params.Add("priority", fmt.Sprintf("%d", *priority))
		}
	case "SRV":
		endpoint = "zone/add_srv"
		params.Add("target", value)
		if priority != nil {
			params.Add("priority", fmt.Sprintf("%d", *priority))
		}
		// Note: Weight and port will need to be added separately
		// as the current function signature doesn't support them
	case "CAA":
		endpoint = "zone/add_caa"
		params.Add("value", value)
		// Note: Flag and tag will need to be added separately
		// as the current function signature doesn't support them
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

// AddSRVRecord adds an SRV record with priority, weight, and port
func (c *Client) AddSRVRecord(domainName, subdomain, target string, priority, weight, port *int) ([]byte, error) {
	params := url.Values{}
	params.Add("domain_name", domainName)
	params.Add("subdomain", subdomain)
	params.Add("output_content_type", "plain")
	params.Add("target", target)

	if priority != nil {
		params.Add("priority", fmt.Sprintf("%d", *priority))
	}
	if weight != nil {
		params.Add("weight", fmt.Sprintf("%d", *weight))
	}
	if port != nil {
		params.Add("port", fmt.Sprintf("%d", *port))
	}

	return c.doRequest("zone/add_srv", params)
}

// AddCAARecord adds a CAA record with flag and tag
func (c *Client) AddCAARecord(domainName, subdomain, value string, flag *int, tag *string) ([]byte, error) {
	params := url.Values{}
	params.Add("domain_name", domainName)
	params.Add("subdomain", subdomain)
	params.Add("output_content_type", "plain")
	params.Add("value", value)

	log.Printf("[DEBUG] AddCAARecord called with flag=%v, tag=%v", flag, tag)

	// Always send flags parameter - API requires it
	if flag != nil {
		params.Add("flags", fmt.Sprintf("%d", *flag))
		log.Printf("[DEBUG] Added flags parameter: %d", *flag)
	} else {
		// Default to 0 if not specified
		params.Add("flags", "0")
		log.Printf("[DEBUG] Added default flags parameter: 0")
	}

	// Always send tag parameter - API requires it
	if tag != nil {
		params.Add("tag", *tag)
		log.Printf("[DEBUG] Added tag parameter: %s", *tag)
	} else {
		// Default to "issue" if not specified
		params.Add("tag", "issue")
		log.Printf("[DEBUG] Added default tag parameter: issue")
	}

	log.Printf("[DEBUG] Final parameters: %v", params)

	return c.doRequest("zone/add_caa", params)
}

// RemoveCAARecord removes a CAA record with flag and tag
func (c *Client) RemoveCAARecord(domainName, subdomain, value string, flag *int, tag *string) ([]byte, error) {
	params := url.Values{}
	params.Add("domain_name", domainName)
	params.Add("subdomain", subdomain)
	params.Add("output_content_type", "plain")
	params.Add("record_type", "CAA")
	params.Add("content", value)

	// Always send flags parameter - API requires it
	if flag != nil {
		params.Add("flags", fmt.Sprintf("%d", *flag))
	} else {
		// Default to 0 if not specified
		params.Add("flags", "0")
	}

	// Always send tag parameter - API requires it
	if tag != nil {
		params.Add("tag", *tag)
	} else {
		// Default to "issue" if not specified
		params.Add("tag", "issue")
	}

	// Use the generic remove_record endpoint
	return c.doRequest("zone/remove_record", params)
}

// RemoveSRVRecord removes an SRV record with priority, weight, and port
func (c *Client) RemoveSRVRecord(domainName, subdomain, target string, priority, weight, port *int) ([]byte, error) {
	params := url.Values{}
	params.Add("domain_name", domainName)
	params.Add("subdomain", subdomain)
	params.Add("output_content_type", "plain")
	params.Add("record_type", "SRV")
	params.Add("content", target)

	if priority != nil {
		params.Add("priority", fmt.Sprintf("%d", *priority))
	}
	if weight != nil {
		params.Add("weight", fmt.Sprintf("%d", *weight))
	}
	if port != nil {
		params.Add("port", fmt.Sprintf("%d", *port))
	}

	// Use the generic remove_record endpoint instead of remove_srv
	return c.doRequest("zone/remove_record", params)
}

// RemoveRecord удаляет запись
func (c *Client) RemoveRecord(domainName, subdomain, recordType, content string, priority *int) ([]byte, error) {
	params := url.Values{}
	params.Add("domain_name", domainName)
	params.Add("subdomain", subdomain)
	params.Add("record_type", recordType)
	params.Add("content", content)
	params.Add("output_content_type", "plain")

	// Добавляем приоритет для MX, NS и SRV записей
	if (recordType == "MX" || recordType == "NS" || recordType == "SRV") && priority != nil {
		params.Add("priority", fmt.Sprintf("%d", *priority))
	}

	return c.doRequest("zone/remove_record", params)
}

// GetRecords получает все записи для зоны
func (c *Client) GetRecords(domainName string) ([]byte, error) {
	params := url.Values{}
	params.Add("dname", domainName)

	return c.doRequest("zone/get_resource_records", params)
}
