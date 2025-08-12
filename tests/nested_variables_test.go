package tests

import (
	"testing"

	"github.com/nulln0ne/fuego/pkg/variables"
)

func TestNestedVariableAccess(t *testing.T) {
	ctx := variables.NewContext()

	// Set up test data
	testData := map[string]interface{}{
		"id":   1,
		"name": "Test User",
		"details": map[string]interface{}{
			"email": "test@example.com",
			"age":   25,
		},
	}

	ctx.SetStep("user", testData)

	// Test basic access
	if value, exists := ctx.GetNested("user"); !exists {
		t.Error("Should be able to access 'user'")
	} else {
		userMap := value.(map[string]interface{})
		if userMap["id"] != 1 {
			t.Errorf("Expected user.id=1, got %v", userMap["id"])
		}
	}

	// Test nested access
	if value, exists := ctx.GetNested("user.id"); !exists {
		t.Error("Should be able to access 'user.id'")
	} else if value != 1 {
		t.Errorf("Expected user.id=1, got %v", value)
	}

	if value, exists := ctx.GetNested("user.name"); !exists {
		t.Error("Should be able to access 'user.name'")
	} else if value != "Test User" {
		t.Errorf("Expected user.name='Test User', got %v", value)
	}

	// Test deeply nested access
	if value, exists := ctx.GetNested("user.details.email"); !exists {
		t.Error("Should be able to access 'user.details.email'")
	} else if value != "test@example.com" {
		t.Errorf("Expected user.details.email='test@example.com', got %v", value)
	}

	// Test non-existent path
	if _, exists := ctx.GetNested("user.nonexistent"); exists {
		t.Error("Should not be able to access non-existent path")
	}
}

func TestVariableInterpolationWithDotNotation(t *testing.T) {
	ctx := variables.NewContext()

	// Set up test data
	ctx.SetStep("item", map[string]interface{}{
		"id":   42,
		"name": "Test Item",
	})

	// Test interpolation
	result, err := ctx.InterpolateString("User ID is ${{item.id}}")
	if err != nil {
		t.Fatalf("Interpolation failed: %v", err)
	}

	expected := "User ID is 42"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test multiple interpolations
	result, err = ctx.InterpolateString("Item: ${{item.name}} with ID ${{item.id}}")
	if err != nil {
		t.Fatalf("Interpolation failed: %v", err)
	}

	expected = "Item: Test Item with ID 42"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
