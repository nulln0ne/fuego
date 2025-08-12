package assertions

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nulln0ne/fuego/pkg/scenario"
	"github.com/nulln0ne/fuego/pkg/variables"
	"github.com/xeipuuv/gojsonschema"
)

type Result struct {
	Passed    bool                `json:"passed"`
	Message   string              `json:"message"`
	Expected  interface{}         `json:"expected,omitempty"`
	Actual    interface{}         `json:"actual,omitempty"`
	Assertion *scenario.Assertion `json:"assertion,omitempty"`
	Duration  time.Duration       `json:"duration"`
}

type Engine struct {
	varContext *variables.Context
}

func NewEngine(varContext *variables.Context) *Engine {
	return &Engine{
		varContext: varContext,
	}
}

func (e *Engine) RunAssertions(assertions []scenario.Assertion, response interface{}) ([]Result, error) {
	results := make([]Result, 0, len(assertions))

	for _, assertion := range assertions {
		result, err := e.runAssertion(assertion, response)
		if err != nil {
			return nil, fmt.Errorf("failed to run assertion %s: %w", assertion.Description, err)
		}
		results = append(results, result)
	}

	return results, nil
}

func (e *Engine) runAssertion(assertion scenario.Assertion, response interface{}) (Result, error) {
	startTime := time.Now()

	result := Result{
		Assertion: &assertion,
		Duration:  0,
	}

	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Interpolate expected value if it's a string template
	expectedValue := assertion.Value
	if expectedStr, ok := assertion.Value.(string); ok {
		// Special handling for JSON schema assertions with variable references
		if assertion.Type == "json_schema" && strings.Contains(expectedStr, "{{") {
			// For JSON schema, try to get the variable directly instead of interpolating as string
			variableName := strings.TrimSpace(strings.Trim(strings.Trim(expectedStr, "{"), "}"))
			if varValue, exists := e.varContext.Get(variableName); exists {
				expectedValue = varValue
			} else {
				// Fall back to string interpolation
				interpolated, err := e.varContext.InterpolateString(expectedStr)
				if err != nil {
					result.Passed = false
					result.Message = fmt.Sprintf("Failed to interpolate expected value: %v", err)
					return result, nil
				}
				expectedValue = interpolated
			}
		} else {
			interpolated, err := e.varContext.InterpolateString(expectedStr)
			if err != nil {
				result.Passed = false
				result.Message = fmt.Sprintf("Failed to interpolate expected value: %v", err)
				return result, nil
			}
			expectedValue = interpolated
		}
	}

	// Extract actual value based on assertion type
	actualValue, err := e.extractValue(assertion, response)
	if err != nil {
		if assertion.Optional {
			result.Passed = true
			result.Message = fmt.Sprintf("Optional assertion skipped: %v", err)
			return result, nil
		}
		result.Passed = false
		result.Message = fmt.Sprintf("Failed to extract value: %v", err)
		return result, nil
	}

	result.Expected = expectedValue
	result.Actual = actualValue

	// Perform comparison based on operator
	passed, message := e.compare(actualValue, expectedValue, assertion.Operator)
	result.Passed = passed

	if assertion.Description != "" {
		result.Message = assertion.Description + ": " + message
	} else {
		result.Message = message
	}

	return result, nil
}

func (e *Engine) extractValue(assertion scenario.Assertion, response interface{}) (interface{}, error) {
	switch assertion.Type {
	case "status", "status_code":
		return e.extractStatusCode(response)
	case "header":
		return e.extractHeader(response, assertion.Field)
	case "body":
		return e.extractBody(response)
	case "json", "json_path":
		return e.extractJSONPath(response, assertion.Field)
	case "xpath":
		return nil, fmt.Errorf("XPath assertions not yet implemented")
	case "regex":
		return e.extractRegex(response, assertion.Field)
	case "response_time":
		return e.extractResponseTime(response)
	case "size":
		return e.extractResponseSize(response)
	case "json_schema":
		return e.extractBody(response) // For JSON schema validation, we validate the entire body
	default:
		return nil, fmt.Errorf("unsupported assertion type: %s", assertion.Type)
	}
}

func (e *Engine) extractStatusCode(response interface{}) (interface{}, error) {
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("response is not a map")
	}

	statusCode, exists := respMap["status_code"]
	if !exists {
		return nil, fmt.Errorf("status_code not found in response")
	}

	return statusCode, nil
}

func (e *Engine) extractHeader(response interface{}, headerName string) (interface{}, error) {
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("response is not a map")
	}

	headers, exists := respMap["headers"]
	if !exists {
		return nil, fmt.Errorf("headers not found in response")
	}

	headersMap, ok := headers.(map[string][]string)
	if !ok {
		return nil, fmt.Errorf("headers are not in expected format")
	}

	values, exists := headersMap[headerName]
	if !exists || len(values) == 0 {
		return nil, fmt.Errorf("header %s not found", headerName)
	}

	return values[0], nil
}

func (e *Engine) extractBody(response interface{}) (interface{}, error) {
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("response is not a map")
	}

	body, exists := respMap["body_text"]
	if !exists {
		return nil, fmt.Errorf("body_text not found in response")
	}

	return body, nil
}

func (e *Engine) extractJSONPath(response interface{}, path string) (interface{}, error) {
	body, err := e.extractBody(response)
	if err != nil {
		return nil, err
	}

	bodyStr, ok := body.(string)
	if !ok {
		return nil, fmt.Errorf("body is not a string")
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(bodyStr), &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return e.getNestedValue(jsonData, path)
}

func (e *Engine) extractRegex(response interface{}, pattern string) (interface{}, error) {
	body, err := e.extractBody(response)
	if err != nil {
		return nil, err
	}

	bodyStr, ok := body.(string)
	if !ok {
		return nil, fmt.Errorf("body is not a string")
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	matches := regex.FindStringSubmatch(bodyStr)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches found for pattern: %s", pattern)
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	return matches[1:], nil // Return captured groups
}

func (e *Engine) extractResponseTime(response interface{}) (interface{}, error) {
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("response is not a map")
	}

	duration, exists := respMap["duration"]
	if !exists {
		return nil, fmt.Errorf("duration not found in response")
	}

	if d, ok := duration.(time.Duration); ok {
		return d.Milliseconds(), nil
	}

	return duration, nil
}

func (e *Engine) extractResponseSize(response interface{}) (interface{}, error) {
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("response is not a map")
	}

	size, exists := respMap["size"]
	if !exists {
		return nil, fmt.Errorf("size not found in response")
	}

	return size, nil
}

func (e *Engine) getNestedValue(data interface{}, path string) (interface{}, error) {
	if path == "" {
		return data, nil
	}

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

func (e *Engine) compare(actual, expected interface{}, operator string) (bool, string) {
	if operator == "" {
		operator = "eq" // default to equality
	}

	switch operator {
	case "eq", "equals", "==":
		return e.compareEqual(actual, expected)
	case "ne", "not_equals", "!=":
		passed, msg := e.compareEqual(actual, expected)
		return !passed, strings.Replace(msg, "equals", "does not equal", 1)
	case "gt", ">":
		return e.compareGreater(actual, expected)
	case "gte", ">=":
		return e.compareGreaterEqual(actual, expected)
	case "lt", "<":
		return e.compareLess(actual, expected)
	case "lte", "<=":
		return e.compareLessEqual(actual, expected)
	case "contains":
		return e.compareContains(actual, expected)
	case "not_contains":
		passed, msg := e.compareContains(actual, expected)
		return !passed, strings.Replace(msg, "contains", "does not contain", 1)
	case "matches", "regex":
		return e.compareRegex(actual, expected)
	case "starts_with":
		return e.compareStartsWith(actual, expected)
	case "ends_with":
		return e.compareEndsWith(actual, expected)
	case "length":
		return e.compareLength(actual, expected)
	case "json_schema":
		return e.compareJSONSchema(actual, expected)
	default:
		return false, fmt.Sprintf("unsupported operator: %s", operator)
	}
}

func (e *Engine) compareEqual(actual, expected interface{}) (bool, string) {
	if reflect.DeepEqual(actual, expected) {
		return true, fmt.Sprintf("value equals %v", expected)
	}

	// Try type conversion for numeric comparisons
	if e.isNumeric(actual) && e.isNumeric(expected) {
		actualFloat := e.toFloat64(actual)
		expectedFloat := e.toFloat64(expected)
		if actualFloat == expectedFloat {
			return true, fmt.Sprintf("value equals %v", expected)
		}
	}

	// Try string conversion for templated values
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	if actualStr == expectedStr {
		return true, fmt.Sprintf("value equals %v", expected)
	}

	return false, fmt.Sprintf("expected %v but got %v", expected, actual)
}

func (e *Engine) compareGreater(actual, expected interface{}) (bool, string) {
	if !e.isNumeric(actual) || !e.isNumeric(expected) {
		return false, "comparison requires numeric values"
	}

	actualFloat := e.toFloat64(actual)
	expectedFloat := e.toFloat64(expected)

	if actualFloat > expectedFloat {
		return true, fmt.Sprintf("value %v is greater than %v", actual, expected)
	}

	return false, fmt.Sprintf("expected value greater than %v but got %v", expected, actual)
}

func (e *Engine) compareGreaterEqual(actual, expected interface{}) (bool, string) {
	if !e.isNumeric(actual) || !e.isNumeric(expected) {
		return false, "comparison requires numeric values"
	}

	actualFloat := e.toFloat64(actual)
	expectedFloat := e.toFloat64(expected)

	if actualFloat >= expectedFloat {
		return true, fmt.Sprintf("value %v is greater than or equal to %v", actual, expected)
	}

	return false, fmt.Sprintf("expected value greater than or equal to %v but got %v", expected, actual)
}

func (e *Engine) compareLess(actual, expected interface{}) (bool, string) {
	if !e.isNumeric(actual) || !e.isNumeric(expected) {
		return false, "comparison requires numeric values"
	}

	actualFloat := e.toFloat64(actual)
	expectedFloat := e.toFloat64(expected)

	if actualFloat < expectedFloat {
		return true, fmt.Sprintf("value %v is less than %v", actual, expected)
	}

	return false, fmt.Sprintf("expected value less than %v but got %v", expected, actual)
}

func (e *Engine) compareLessEqual(actual, expected interface{}) (bool, string) {
	if !e.isNumeric(actual) || !e.isNumeric(expected) {
		return false, "comparison requires numeric values"
	}

	actualFloat := e.toFloat64(actual)
	expectedFloat := e.toFloat64(expected)

	if actualFloat <= expectedFloat {
		return true, fmt.Sprintf("value %v is less than or equal to %v", actual, expected)
	}

	return false, fmt.Sprintf("expected value less than or equal to %v but got %v", expected, actual)
}

func (e *Engine) compareContains(actual, expected interface{}) (bool, string) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	if strings.Contains(actualStr, expectedStr) {
		return true, fmt.Sprintf("value contains %v", expected)
	}

	return false, fmt.Sprintf("expected value to contain %v but got %v", expected, actual)
}

func (e *Engine) compareRegex(actual, expected interface{}) (bool, string) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	matched, err := regexp.MatchString(expectedStr, actualStr)
	if err != nil {
		return false, fmt.Sprintf("invalid regex pattern: %v", err)
	}

	if matched {
		return true, fmt.Sprintf("value matches pattern %v", expected)
	}

	return false, fmt.Sprintf("expected value to match pattern %v but got %v", expected, actual)
}

func (e *Engine) compareStartsWith(actual, expected interface{}) (bool, string) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	if strings.HasPrefix(actualStr, expectedStr) {
		return true, fmt.Sprintf("value starts with %v", expected)
	}

	return false, fmt.Sprintf("expected value to start with %v but got %v", expected, actual)
}

func (e *Engine) compareEndsWith(actual, expected interface{}) (bool, string) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	if strings.HasSuffix(actualStr, expectedStr) {
		return true, fmt.Sprintf("value ends with %v", expected)
	}

	return false, fmt.Sprintf("expected value to end with %v but got %v", expected, actual)
}

func (e *Engine) compareLength(actual, expected interface{}) (bool, string) {
	actualLength := e.getLength(actual)
	expectedInt, ok := expected.(int)
	if !ok {
		if expectedFloat, ok := expected.(float64); ok {
			expectedInt = int(expectedFloat)
		} else {
			return false, "expected length must be a number"
		}
	}

	if actualLength == expectedInt {
		return true, fmt.Sprintf("length equals %d", expectedInt)
	}

	return false, fmt.Sprintf("expected length %d but got %d", expectedInt, actualLength)
}

func (e *Engine) isNumeric(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	default:
		return false
	}
}

func (e *Engine) toFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}

func (e *Engine) getLength(value interface{}) int {
	switch v := value.(type) {
	case string:
		return len(v)
	case []interface{}:
		return len(v)
	case map[string]interface{}:
		return len(v)
	default:
		return 0
	}
}

func (e *Engine) compareJSONSchema(actual, expected interface{}) (bool, string) {
	// Convert actual value to string if it's not already
	actualStr, ok := actual.(string)
	if !ok {
		actualBytes, err := json.Marshal(actual)
		if err != nil {
			return false, fmt.Sprintf("failed to marshal actual value for schema validation: %v", err)
		}
		actualStr = string(actualBytes)
	}

	// Expected should be the JSON schema (now properly resolved)
	schemaBytes, err := json.Marshal(expected)
	if err != nil {
		return false, fmt.Sprintf("failed to marshal schema: %v", err)
	}

	// Create schema loader
	schemaLoader := gojsonschema.NewStringLoader(string(schemaBytes))
	documentLoader := gojsonschema.NewStringLoader(actualStr)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return false, fmt.Sprintf("schema validation error: %v", err)
	}

	if result.Valid() {
		return true, "JSON response validates against provided schema"
	}

	// Collect validation errors
	var errorMessages []string
	for _, desc := range result.Errors() {
		errorMessages = append(errorMessages, desc.String())
	}

	return false, fmt.Sprintf("JSON schema validation failed: %s", strings.Join(errorMessages, "; "))
}
