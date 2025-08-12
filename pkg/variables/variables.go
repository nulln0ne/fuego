package variables

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Context struct {
	global map[string]interface{}
	local  map[string]interface{}
	step   map[string]interface{}
}

func NewContext() *Context {
	return &Context{
		global: make(map[string]interface{}),
		local:  make(map[string]interface{}),
		step:   make(map[string]interface{}),
	}
}

func (c *Context) SetGlobal(key string, value interface{}) {
	c.global[key] = value
}

func (c *Context) SetLocal(key string, value interface{}) {
	c.local[key] = value
}

func (c *Context) SetStep(key string, value interface{}) {
	c.step[key] = value
}

func (c *Context) Get(key string) (interface{}, bool) {
	// Check step scope first, then local, then global
	if value, exists := c.step[key]; exists {
		return value, true
	}
	if value, exists := c.local[key]; exists {
		return value, true
	}
	if value, exists := c.global[key]; exists {
		return value, true
	}
	return nil, false
}

func (c *Context) GetNested(path string) (interface{}, bool) {
	// If no dot notation, use regular Get
	if !strings.Contains(path, ".") {
		return c.Get(path)
	}

	// Split path into parts
	parts := strings.Split(path, ".")
	rootKey := parts[0]

	// Get the root object
	current, exists := c.Get(rootKey)
	if !exists {
		return nil, false
	}

	// Navigate through the path
	for _, part := range parts[1:] {
		switch v := current.(type) {
		case map[string]interface{}:
			if value, ok := v[part]; ok {
				current = value
			} else {
				return nil, false
			}
		case map[interface{}]interface{}:
			if value, ok := v[part]; ok {
				current = value
			} else {
				return nil, false
			}
		default:
			// Try to access as JSON-like object through reflection
			if jsonData, err := c.accessJSONPath(current, part); err == nil {
				current = jsonData
			} else {
				return nil, false
			}
		}
	}

	return current, true
}

func (c *Context) accessJSONPath(data interface{}, key string) (interface{}, error) {
	// Try to convert to map[string]interface{} via JSON marshaling/unmarshaling
	// This handles cases where the data might be in a different map type
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		return nil, err
	}

	if value, exists := jsonMap[key]; exists {
		return value, nil
	}

	return nil, fmt.Errorf("key %s not found", key)
}

func (c *Context) GetAll() map[string]interface{} {
	result := make(map[string]interface{})

	// Add in order: global, local, step (later ones override earlier ones)
	for k, v := range c.global {
		result[k] = v
	}
	for k, v := range c.local {
		result[k] = v
	}
	for k, v := range c.step {
		result[k] = v
	}

	return result
}

func (c *Context) ClearStep() {
	c.step = make(map[string]interface{})
}

func (c *Context) Clone() *Context {
	clone := NewContext()

	// Deep copy global variables
	for k, v := range c.global {
		clone.global[k] = v
	}

	// Deep copy local variables
	for k, v := range c.local {
		clone.local[k] = v
	}

	// Step variables are not cloned as they're step-specific

	return clone
}

// Template interpolation
var templateRegex = regexp.MustCompile(`\$\{\{([^}]+)\}\}|\{\{([^}]+)\}\}`)

func (c *Context) InterpolateString(input string) (string, error) {
	return templateRegex.ReplaceAllStringFunc(input, func(match string) string {
		var varName string
		if strings.HasPrefix(match, "${{") {
			// Extract variable name from ${{varname}}
			varName = strings.TrimSpace(match[3 : len(match)-2])
		} else {
			// Extract variable name from {{varname}}
			varName = strings.TrimSpace(match[2 : len(match)-2])
		}

		// Handle env.variable syntax
		varName = strings.TrimPrefix(varName, "env.")

		// Handle dot notation for nested object access
		value, exists := c.GetNested(varName)
		if !exists {
			return match // Return original if variable not found
		}

		return fmt.Sprintf("%v", value)
	}), nil
}

func (c *Context) InterpolateMap(input map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	for key, value := range input {
		interpolatedKey, err := c.InterpolateString(key)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate key %s: %w", key, err)
		}

		interpolatedValue, err := c.InterpolateString(value)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate value for key %s: %w", key, err)
		}

		result[interpolatedKey] = interpolatedValue
	}

	return result, nil
}

func (c *Context) InterpolateInterface(input interface{}) (interface{}, error) {
	switch v := input.(type) {
	case string:
		return c.InterpolateString(v)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			interpolatedKey, err := c.InterpolateString(key)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate key %s: %w", key, err)
			}

			interpolatedValue, err := c.InterpolateInterface(value)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate value for key %s: %w", key, err)
			}

			result[interpolatedKey] = interpolatedValue
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			interpolatedItem, err := c.InterpolateInterface(item)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate array item at index %d: %w", i, err)
			}
			result[i] = interpolatedItem
		}
		return result, nil
	case map[string]string:
		return c.InterpolateMap(v)
	default:
		return input, nil // Return as-is for non-string types
	}
}

// Built-in functions for variable extraction
func ExtractFromResponse(response map[string]interface{}, extractor string) (interface{}, error) {
	switch {
	case strings.HasPrefix(extractor, "json:"):
		return extractJSONPath(response, strings.TrimPrefix(extractor, "json:"))
	case strings.HasPrefix(extractor, "header:"):
		return extractHeader(response, strings.TrimPrefix(extractor, "header:"))
	case strings.HasPrefix(extractor, "status"):
		return response["status_code"], nil
	case strings.HasPrefix(extractor, "body"):
		return response["body_text"], nil
	default:
		return nil, fmt.Errorf("unsupported extractor: %s", extractor)
	}
}

func extractJSONPath(response map[string]interface{}, path string) (interface{}, error) {
	bodyText, ok := response["body_text"].(string)
	if !ok {
		return nil, fmt.Errorf("response body is not a string")
	}

	var bodyData interface{}
	if err := json.Unmarshal([]byte(bodyText), &bodyData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON body: %w", err)
	}

	return getNestedValue(bodyData, path)
}

func extractHeader(response map[string]interface{}, headerName string) (interface{}, error) {
	headers, ok := response["headers"].(map[string][]string)
	if !ok {
		return nil, fmt.Errorf("response headers are not in expected format")
	}

	values, exists := headers[headerName]
	if !exists || len(values) == 0 {
		return nil, fmt.Errorf("header %s not found", headerName)
	}

	return values[0], nil
}

func getNestedValue(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("key %s not found", part)
			}
		case []interface{}:
			index, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", part)
			}
			if index < 0 || index >= len(v) {
				return nil, fmt.Errorf("array index out of bounds: %d", index)
			}
			current = v[index]
		default:
			return nil, fmt.Errorf("cannot access property %s on %T", part, current)
		}
	}

	return current, nil
}

// Built-in variable functions
func (c *Context) AddBuiltins() {
	now := time.Now()
	c.SetGlobal("timestamp", now.Unix())
	c.SetGlobal("timestamp_ms", now.UnixMilli())
	c.SetGlobal("iso_timestamp", now.Format(time.RFC3339))
	c.SetGlobal("date", now.Format("2006-01-02"))
	c.SetGlobal("time", now.Format("15:04:05"))
}
