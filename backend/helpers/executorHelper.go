package helpers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func BuildURL(baseURL string, path string, params map[string]interface{}) (string, error) {
	base := strings.TrimRight(baseURL, "/")
	rel := path
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}

	u, err := url.Parse(base + rel)
	if err != nil {
		return "", err
	}

	if len(params) == 0 {
		return u.String(), nil
	}

	query := u.Query()
	for key, value := range params {
		query.Set(key, fmt.Sprintf("%v", value))
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func ExtractResourceID(responseBody string) string {
	if strings.TrimSpace(responseBody) == "" {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(responseBody), &payload); err != nil {
		return ""
	}

	candidateKeys := []string{"id", "_id", "user_id", "resource_id"}
	for _, key := range candidateKeys {
		if value, ok := payload[key]; ok {
			return fmt.Sprintf("%v", value)
		}
	}
	if nested, ok := payload["data"].(map[string]any); ok {
		for _, key := range candidateKeys {
			if value, exists := nested[key]; exists {
				return fmt.Sprintf("%v", value)
			}
		}
	}

	return ""
}

func SanitizeName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-")
	return replacer.Replace(trimmed)
}
