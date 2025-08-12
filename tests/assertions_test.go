package tests

import (
	"testing"

	"github.com/nulln0ne/fuego/pkg/assertions"
	"github.com/nulln0ne/fuego/pkg/scenario"
	"github.com/nulln0ne/fuego/pkg/variables"
)

func TestBasicAssertions(t *testing.T) {
	varContext := variables.NewContext()
	engine := assertions.NewEngine(varContext)

	// Mock response
	response := map[string]interface{}{
		"status_code": 200,
		"body_text":   `{"id": 1, "name": "John Doe", "email": "john@example.com"}`,
		"headers": map[string][]string{
			"Content-Type": {"application/json"},
		},
	}

	tests := []struct {
		name      string
		assertion scenario.Assertion
		wantPass  bool
	}{
		{
			name: "Status code equals 200",
			assertion: scenario.Assertion{
				Type:     "status",
				Operator: "eq",
				Value:    200,
			},
			wantPass: true,
		},
		{
			name: "JSON path assertion",
			assertion: scenario.Assertion{
				Type:     "json_path",
				Field:    "id",
				Operator: "eq",
				Value:    1,
			},
			wantPass: true,
		},
		{
			name: "Header contains assertion",
			assertion: scenario.Assertion{
				Type:     "header",
				Field:    "Content-Type",
				Operator: "contains",
				Value:    "json",
			},
			wantPass: true,
		},
		{
			name: "Failing assertion",
			assertion: scenario.Assertion{
				Type:     "status",
				Operator: "eq",
				Value:    404,
			},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := engine.RunAssertions([]scenario.Assertion{tt.assertion}, response)
			if err != nil {
				t.Errorf("RunAssertions() error = %v", err)
				return
			}

			if len(results) != 1 {
				t.Errorf("Expected 1 result, got %d", len(results))
				return
			}

			result := results[0]
			if result.Passed != tt.wantPass {
				t.Errorf("Expected assertion to pass=%v, got pass=%v. Message: %s", tt.wantPass, result.Passed, result.Message)
			}
		})
	}
}

func TestVariableInterpolation(t *testing.T) {
	varContext := variables.NewContext()
	varContext.SetGlobal("expected_status", 200)
	varContext.SetLocal("user_id", 1)

	engine := assertions.NewEngine(varContext)

	response := map[string]interface{}{
		"status_code": 200,
		"body_text":   `{"id": 1, "name": "John Doe"}`,
	}

	assertion := scenario.Assertion{
		Type:     "status",
		Operator: "eq",
		Value:    "{{expected_status}}", // This should be interpolated to 200
	}

	results, err := engine.RunAssertions([]scenario.Assertion{assertion}, response)
	if err != nil {
		t.Errorf("RunAssertions() error = %v", err)
		return
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
		return
	}

	result := results[0]
	if !result.Passed {
		t.Errorf("Expected assertion to pass, but it failed. Message: %s", result.Message)
	}
}
