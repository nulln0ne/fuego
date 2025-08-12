package tests

import (
	"testing"

	"github.com/nulln0ne/fuego/pkg/assertions"
	"github.com/nulln0ne/fuego/pkg/scenario"
	"github.com/nulln0ne/fuego/pkg/variables"
)

func TestJSONSchemaValidation(t *testing.T) {
	tests := []struct {
		name           string
		response       map[string]interface{}
		schema         map[string]interface{}
		expectedPassed bool
		description    string
	}{
		{
			name: "Valid user object against schema",
			response: map[string]interface{}{
				"status_code": 200,
				"headers":     map[string][]string{},
				"body_text":   `{"id": 1, "name": "John Doe", "email": "john@example.com"}`,
			},
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type": "integer",
					},
					"name": map[string]interface{}{
						"type": "string",
					},
					"email": map[string]interface{}{
						"type":   "string",
						"format": "email",
					},
				},
				"required": []string{"id", "name", "email"},
			},
			expectedPassed: true,
			description:    "Valid user object should pass schema validation",
		},
		{
			name: "Invalid user object missing required field",
			response: map[string]interface{}{
				"status_code": 200,
				"headers":     map[string][]string{},
				"body_text":   `{"id": 1, "name": "John Doe"}`,
			},
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type": "integer",
					},
					"name": map[string]interface{}{
						"type": "string",
					},
					"email": map[string]interface{}{
						"type":   "string",
						"format": "email",
					},
				},
				"required": []string{"id", "name", "email"},
			},
			expectedPassed: false,
			description:    "User object missing required email field should fail validation",
		},
		{
			name: "Valid array against schema",
			response: map[string]interface{}{
				"status_code": 200,
				"headers":     map[string][]string{},
				"body_text":   `[{"id": 1, "title": "Post 1"}, {"id": 2, "title": "Post 2"}]`,
			},
			schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "integer",
						},
						"title": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"id", "title"},
				},
			},
			expectedPassed: true,
			description:    "Valid array of objects should pass schema validation",
		},
		{
			name: "Invalid type against schema",
			response: map[string]interface{}{
				"status_code": 200,
				"headers":     map[string][]string{},
				"body_text":   `{"id": "not-a-number", "name": "John Doe", "email": "john@example.com"}`,
			},
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type": "integer",
					},
					"name": map[string]interface{}{
						"type": "string",
					},
					"email": map[string]interface{}{
						"type":   "string",
						"format": "email",
					},
				},
				"required": []string{"id", "name", "email"},
			},
			expectedPassed: false,
			description:    "Object with wrong type should fail schema validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create variable context
			varContext := variables.NewContext()

			// Create assertion engine
			engine := assertions.NewEngine(varContext)

			// Create assertion
			assertion := scenario.Assertion{
				Type:        "json_schema",
				Operator:    "json_schema",
				Value:       tt.schema,
				Description: tt.description,
			}

			// Run assertion
			result, err := engine.RunAssertions([]scenario.Assertion{assertion}, tt.response)
			if err != nil {
				t.Fatalf("Failed to run assertion: %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(result))
			}

			if result[0].Passed != tt.expectedPassed {
				t.Errorf("Expected passed=%v, got passed=%v. Message: %s",
					tt.expectedPassed, result[0].Passed, result[0].Message)
			}

			// Verify the result contains expected information
			if tt.expectedPassed && result[0].Message == "" {
				t.Error("Expected success message but got empty string")
			}
			if !tt.expectedPassed && result[0].Message == "" {
				t.Error("Expected error message but got empty string")
			}
		})
	}
}

func TestJSONSchemaWithVariableInterpolation(t *testing.T) {
	// Create variable context with schema
	varContext := variables.NewContext()
	userSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type": "integer",
			},
			"name": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"id", "name"},
	}
	varContext.SetGlobal("user_schema", userSchema)

	// Create assertion engine
	engine := assertions.NewEngine(varContext)

	// Create response
	response := map[string]interface{}{
		"status_code": 200,
		"headers":     map[string][]string{},
		"body_text":   `{"id": 1, "name": "John Doe"}`,
	}

	// Create assertion using variable reference
	assertion := scenario.Assertion{
		Type:        "json_schema",
		Operator:    "json_schema",
		Value:       "{{user_schema}}", // This should be interpolated
		Description: "Validate user against schema from variable",
	}

	// Run assertion
	result, err := engine.RunAssertions([]scenario.Assertion{assertion}, response)
	if err != nil {
		t.Fatalf("Failed to run assertion: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Passed {
		t.Errorf("Expected validation to pass, but got: %s", result[0].Message)
	}
}
