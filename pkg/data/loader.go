package data

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// DataSource represents different types of data sources
type DataSource struct {
	Type string      `yaml:"type" json:"type"` // csv, json, inline
	Path string      `yaml:"path,omitempty" json:"path,omitempty"`
	Data interface{} `yaml:"data,omitempty" json:"data,omitempty"`
}

// DataLoader handles loading data from various sources
type DataLoader struct {
	baseDir string
}

// NewDataLoader creates a new data loader with a base directory
func NewDataLoader(baseDir string) *DataLoader {
	return &DataLoader{
		baseDir: baseDir,
	}
}

// LoadData loads data from the specified source
func (dl *DataLoader) LoadData(source DataSource) ([]map[string]interface{}, error) {
	switch source.Type {
	case "csv":
		return dl.loadCSV(source.Path)
	case "json":
		return dl.loadJSON(source.Path)
	case "inline":
		return dl.loadInline(source.Data)
	default:
		return nil, fmt.Errorf("unsupported data source type: %s", source.Type)
	}
}

// loadCSV loads data from a CSV file
func (dl *DataLoader) loadCSV(path string) ([]map[string]interface{}, error) {
	fullPath := dl.resolvePath(path)
	
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %s: %w", fullPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var data []map[string]interface{}
	
	// Read data rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		if len(record) != len(headers) {
			return nil, fmt.Errorf("CSV row has %d columns, expected %d", len(record), len(headers))
		}

		row := make(map[string]interface{})
		for i, value := range record {
			row[headers[i]] = dl.parseCSVValue(value)
		}
		data = append(data, row)
	}

	return data, nil
}

// loadJSON loads data from a JSON file
func (dl *DataLoader) loadJSON(path string) ([]map[string]interface{}, error) {
	fullPath := dl.resolvePath(path)
	
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSON file %s: %w", fullPath, err)
	}
	defer file.Close()

	var data interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON file %s: %w", fullPath, err)
	}

	// Convert to array of maps
	switch v := data.(type) {
	case []interface{}:
		var result []map[string]interface{}
		for i, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result = append(result, itemMap)
			} else {
				return nil, fmt.Errorf("JSON array item %d is not an object", i)
			}
		}
		return result, nil
	case map[string]interface{}:
		// Single object, wrap in array
		return []map[string]interface{}{v}, nil
	default:
		return nil, fmt.Errorf("JSON file must contain an object or array of objects")
	}
}

// loadInline loads inline data
func (dl *DataLoader) loadInline(data interface{}) ([]map[string]interface{}, error) {
	switch v := data.(type) {
	case []interface{}:
		var result []map[string]interface{}
		for i, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result = append(result, itemMap)
			} else {
				return nil, fmt.Errorf("inline data item %d is not an object", i)
			}
		}
		return result, nil
	case map[string]interface{}:
		// Single object, wrap in array
		return []map[string]interface{}{v}, nil
	default:
		return nil, fmt.Errorf("inline data must be an object or array of objects")
	}
}

// parseCSVValue attempts to parse a CSV value as the most appropriate type
func (dl *DataLoader) parseCSVValue(value string) interface{} {
	// Try to parse as integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}
	
	// Try to parse as float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}
	
	// Try to parse as boolean
	if boolVal, err := strconv.ParseBool(value); err == nil {
		return boolVal
	}
	
	// Return as string
	return value
}

// resolvePath resolves a path relative to the base directory
func (dl *DataLoader) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dl.baseDir, path)
}

// DataIterator provides iteration over data sets for data-driven tests
type DataIterator struct {
	data  []map[string]interface{}
	index int
}

// NewDataIterator creates a new data iterator
func NewDataIterator(data []map[string]interface{}) *DataIterator {
	return &DataIterator{
		data:  data,
		index: 0,
	}
}

// HasNext returns true if there are more data items
func (di *DataIterator) HasNext() bool {
	return di.index < len(di.data)
}

// Next returns the next data item
func (di *DataIterator) Next() map[string]interface{} {
	if !di.HasNext() {
		return nil
	}
	item := di.data[di.index]
	di.index++
	return item
}

// Reset resets the iterator to the beginning
func (di *DataIterator) Reset() {
	di.index = 0
}

// Count returns the total number of data items
func (di *DataIterator) Count() int {
	return len(di.data)
}

// All returns all data items
func (di *DataIterator) All() []map[string]interface{} {
	return di.data
}