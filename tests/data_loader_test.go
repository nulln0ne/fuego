package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nulln0ne/fuego/pkg/data"
)

func TestCSVDataLoader(t *testing.T) {
	// Create temporary CSV file
	tempDir := t.TempDir()
	csvFile := filepath.Join(tempDir, "test.csv")
	
	csvContent := `id,name,email,active
1,Alice,alice@example.com,true
2,Bob,bob@example.com,false
3,Charlie,charlie@example.com,true`

	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV file: %v", err)
	}

	// Test CSV loading
	loader := data.NewDataLoader(tempDir)
	source := data.DataSource{
		Type: "csv",
		Path: "test.csv",
	}

	result, err := loader.LoadData(source)
	if err != nil {
		t.Fatalf("Failed to load CSV data: %v", err)
	}

	// Verify results
	if len(result) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(result))
	}

	// Check first row
	firstRow := result[0]
	if firstRow["id"] != 1 {
		t.Errorf("Expected id=1, got %v", firstRow["id"])
	}
	if firstRow["name"] != "Alice" {
		t.Errorf("Expected name=Alice, got %v", firstRow["name"])
	}
	if firstRow["email"] != "alice@example.com" {
		t.Errorf("Expected email=alice@example.com, got %v", firstRow["email"])
	}
	if firstRow["active"] != true {
		t.Errorf("Expected active=true, got %v", firstRow["active"])
	}

	// Check data types
	if _, ok := firstRow["id"].(int); !ok {
		t.Errorf("Expected id to be int, got %T", firstRow["id"])
	}
	if _, ok := firstRow["active"].(bool); !ok {
		t.Errorf("Expected active to be bool, got %T", firstRow["active"])
	}
}

func TestJSONDataLoader(t *testing.T) {
	// Create temporary JSON file
	tempDir := t.TempDir()
	jsonFile := filepath.Join(tempDir, "test.json")
	
	jsonContent := `[
		{"id": 1, "name": "Alice", "score": 95.5},
		{"id": 2, "name": "Bob", "score": 87.2},
		{"id": 3, "name": "Charlie", "score": 92.1}
	]`

	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}

	// Test JSON loading
	loader := data.NewDataLoader(tempDir)
	source := data.DataSource{
		Type: "json",
		Path: "test.json",
	}

	result, err := loader.LoadData(source)
	if err != nil {
		t.Fatalf("Failed to load JSON data: %v", err)
	}

	// Verify results
	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	// Check first item
	firstItem := result[0]
	if firstItem["id"].(float64) != 1 {
		t.Errorf("Expected id=1, got %v", firstItem["id"])
	}
	if firstItem["name"] != "Alice" {
		t.Errorf("Expected name=Alice, got %v", firstItem["name"])
	}
	if firstItem["score"].(float64) != 95.5 {
		t.Errorf("Expected score=95.5, got %v", firstItem["score"])
	}
}

func TestInlineDataLoader(t *testing.T) {
	loader := data.NewDataLoader(".")
	
	// Test with array of objects
	source := data.DataSource{
		Type: "inline",
		Data: []interface{}{
			map[string]interface{}{"id": 1, "name": "Test1"},
			map[string]interface{}{"id": 2, "name": "Test2"},
		},
	}

	result, err := loader.LoadData(source)
	if err != nil {
		t.Fatalf("Failed to load inline data: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	// Test with single object
	singleSource := data.DataSource{
		Type: "inline",
		Data: map[string]interface{}{"id": 1, "name": "Single"},
	}

	singleResult, err := loader.LoadData(singleSource)
	if err != nil {
		t.Fatalf("Failed to load single inline data: %v", err)
	}

	if len(singleResult) != 1 {
		t.Errorf("Expected 1 item, got %d", len(singleResult))
	}

	if singleResult[0]["name"] != "Single" {
		t.Errorf("Expected name=Single, got %v", singleResult[0]["name"])
	}
}

func TestDataIterator(t *testing.T) {
	testData := []map[string]interface{}{
		{"id": 1, "name": "Alice"},
		{"id": 2, "name": "Bob"},
		{"id": 3, "name": "Charlie"},
	}

	iterator := data.NewDataIterator(testData)

	// Test iteration
	count := 0
	for iterator.HasNext() {
		item := iterator.Next()
		count++
		if item["id"].(int) != count {
			t.Errorf("Expected id=%d, got %v", count, item["id"])
		}
	}

	if count != 3 {
		t.Errorf("Expected 3 iterations, got %d", count)
	}

	// Test reset
	iterator.Reset()
	if !iterator.HasNext() {
		t.Error("Expected iterator to have items after reset")
	}

	// Test count
	if iterator.Count() != 3 {
		t.Errorf("Expected count=3, got %d", iterator.Count())
	}
}

func TestCSVValueParsing(t *testing.T) {
	// Test CSV value parsing implicitly through CSV loading
	tempDir := t.TempDir()
	csvFile := filepath.Join(tempDir, "types.csv")
	
	csvContent := `int_val,float_val,bool_val,string_val
123,123.45,true,hello
456,789.01,false,world`

	err := os.WriteFile(csvFile, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSV file: %v", err)
	}

	loader := data.NewDataLoader(tempDir)
	source := data.DataSource{
		Type: "csv",
		Path: "types.csv",
	}

	result, err := loader.LoadData(source)
	if err != nil {
		t.Fatalf("Failed to load CSV data: %v", err)
	}

	// Check first row type parsing
	firstRow := result[0]
	if firstRow["int_val"] != 123 {
		t.Errorf("Expected int_val=123, got %v", firstRow["int_val"])
	}
	if firstRow["float_val"] != 123.45 {
		t.Errorf("Expected float_val=123.45, got %v", firstRow["float_val"])
	}
	if firstRow["bool_val"] != true {
		t.Errorf("Expected bool_val=true, got %v", firstRow["bool_val"])
	}
	if firstRow["string_val"] != "hello" {
		t.Errorf("Expected string_val=hello, got %v", firstRow["string_val"])
	}
}