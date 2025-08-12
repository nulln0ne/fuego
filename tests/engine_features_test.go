package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nulln0ne/fuego/pkg/config"
	"github.com/nulln0ne/fuego/pkg/execution"
	"github.com/nulln0ne/fuego/pkg/reporting"
	"github.com/nulln0ne/fuego/pkg/scenario"
	"github.com/stretchr/testify/assert"
)

// setupTestServer creates a mock HTTP server for testing.
func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test-Header", "fuego-test")
		fmt.Fprintln(w, `{"user": {"id": 123, "name": "fuego"}, "status": "ok"}`)
	})
	mux.HandleFunc("/text", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, Fuego! Your session ID is sess-abc-123.")
	})
	mux.HandleFunc("/user/123", func(w http.ResponseWriter, r *http.Request) {
		// Endpoint for checking variable interpolation
		w.WriteHeader(http.StatusOK)
	})
	return httptest.NewServer(mux)
}

func runTestScenario(t *testing.T, sc *scenario.Scenario) *reporting.Report {
	cfg := &config.Config{}
	// Create a null reporter to avoid console output during tests
	reporter := reporting.NewReporter(reporting.ReportConfig{Format: "json", OutputFile: os.DevNull})
	engine := execution.NewEngine(cfg, reporter)

	err := engine.ExecuteScenarios([]*scenario.Scenario{sc})
	assert.NoError(t, err)

	// Since the report is written to /dev/null, we need to access it from the reporter directly.
	// The reporter's GenerateReport method populates the report field.
	return reporter.GetReport()
}

func TestConditionEvaluation(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	sc := &scenario.Scenario{
		Name: "Condition Evaluation Test",
		Tests: map[string]*scenario.TestGroup{
			"main": {
				Steps: []scenario.Step{
					{
						Name: "Set condition variable to true",
						Variables: map[string]interface{}{
							"should_run": true,
						},
					},
					{
						Name:      "This step should run",
						Condition: "{{should_run}}",
						HTTP: &scenario.HTTPStep{
							Method: "GET",
							URL:    server.URL + "/json",
						},
						Check: map[string]interface{}{
							"status": 200,
						},
					},
					{
						Name: "Set condition variable to false",
						Variables: map[string]interface{}{
							"should_run": false,
						},
					},
					{
						Name:      "This step should be skipped",
						Condition: "{{should_run}}",
						HTTP: &scenario.HTTPStep{
							Method: "GET",
							URL:    server.URL + "/json",
						},
					},
				},
			},
		},
	}

	report := runTestScenario(t, sc)

	assert.Len(t, report.Scenarios, 1)
	scenarioResult := report.Scenarios[0]

	// 4 steps total: set true, run, set false, skip
	assert.Len(t, scenarioResult.Steps, 4)

	stepRun := scenarioResult.Steps[1]
	assert.Equal(t, "passed", stepRun.Status, "Expected step '%s' to be passed, but was %s", stepRun.Step.Name, stepRun.Status)

	stepSkipped := scenarioResult.Steps[3]
	assert.Equal(t, "skipped", stepSkipped.Status, "Expected step '%s' to be skipped, but was %s", stepSkipped.Step.Name, stepSkipped.Status)
}

func TestLegacyVariableExtraction(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	sc := &scenario.Scenario{
		Name: "Legacy Variable Extraction Test",
		Steps: []scenario.Step{
			{
				Name: "Get user and extract variables",
				Type: "http",
				Request: scenario.Request{
					Method: "GET",
					URL:    server.URL + "/json",
				},
				Variables: map[string]interface{}{
					"user_id":    "json:user.id",
					"header_val": "header:X-Test-Header",
				},
			},
			{
				Name: "Use extracted variables",
				Type: "http",
				Request: scenario.Request{
					Method: "GET",
					URL:    server.URL + "/user/{{user_id}}",
					Headers: map[string]string{
						"X-User-ID": "{{user_id}}",
						"X-Test":    "{{header_val}}",
					},
				},
				Assertions: []scenario.Assertion{
					{Type: "status", Operator: "eq", Value: 200},
				},
			},
		},
	}

	report := runTestScenario(t, sc)

	assert.Len(t, report.Scenarios, 1)
	scenarioResult := report.Scenarios[0]
	assert.Equal(t, "passed", scenarioResult.Status)

	finalVars := scenarioResult.Variables
	assert.Equal(t, float64(123), finalVars["user_id"])
	assert.Equal(t, "fuego-test", finalVars["header_val"])
}

func TestRegexCapture(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	sc := &scenario.Scenario{
		Name: "Regex Capture Test",
		Tests: map[string]*scenario.TestGroup{
			"main": {
				Steps: []scenario.Step{
					{
						Name: "Get text and capture session ID",
						HTTP: &scenario.HTTPStep{
							Method: "GET",
							URL:    server.URL + "/text",
						},
						Capture: map[string]scenario.Capture{
							"session_id": {Regex: `sess-([a-z0-9-]+)`},
						},
					},
				},
			},
		},
	}

	report := runTestScenario(t, sc)

	assert.Len(t, report.Scenarios, 1)
	scenarioResult := report.Scenarios[0]
	assert.Equal(t, "passed", scenarioResult.Status)

	finalVars := scenarioResult.Variables
	assert.Equal(t, "abc-123", finalVars["session_id"])
}
