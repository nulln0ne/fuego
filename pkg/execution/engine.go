package execution

import (
	"fmt"
	"sync"
	"time"

	"github.com/nulln0ne/fuego/pkg/assertions"
	"github.com/nulln0ne/fuego/pkg/config"
	"github.com/nulln0ne/fuego/pkg/protocols"
	"github.com/nulln0ne/fuego/pkg/reporting"
	"github.com/nulln0ne/fuego/pkg/scenario"
	"github.com/nulln0ne/fuego/pkg/variables"
)

type Engine struct {
	config     *config.Config
	reporter   *reporting.Reporter
	varContext *variables.Context
	httpClient *protocols.HTTPClient
}

func NewEngine(cfg *config.Config, reporter *reporting.Reporter) *Engine {
	varContext := variables.NewContext()
	varContext.AddBuiltins()

	// Add global variables from config
	for k, v := range cfg.Global.Variables {
		varContext.SetGlobal(k, v)
	}

	// Create HTTP client
	httpClient := protocols.NewHTTPClient(protocols.HTTPClientConfig{
		BaseURL:         cfg.Global.BaseURL,
		Headers:         cfg.Global.Headers,
		Timeout:         cfg.Defaults.HTTPTimeout,
		VerifySSL:       cfg.Defaults.VerifySSL,
		FollowRedirects: cfg.Defaults.FollowRedirect,
	})

	return &Engine{
		config:     cfg,
		reporter:   reporter,
		varContext: varContext,
		httpClient: httpClient,
	}
}

func (e *Engine) ExecuteScenarios(scenarios []*scenario.Scenario) error {
	e.reporter.Start()

	for _, sc := range scenarios {
		result := e.executeScenario(sc)
		e.reporter.AddScenarioResult(result)
	}

	return e.reporter.GenerateReport()
}

func (e *Engine) executeScenario(sc *scenario.Scenario) reporting.ScenarioResult {
	result := reporting.ScenarioResult{
		Scenario:  sc,
		StartTime: time.Now(),
		Steps:     make([]reporting.StepResult, 0),
		Variables: make(map[string]interface{}),
	}

	// Create scenario-specific variable context
	scenarioContext := e.varContext.Clone()

	// Add environment variables
	for k, v := range sc.Env {
		scenarioContext.SetGlobal(k, v)
	}

	// Add scenario variables
	for k, v := range sc.Variables {
		scenarioContext.SetLocal(k, v)
	}

	// Apply environment-specific configuration if specified
	if sc.Config != nil && sc.Config.Environment != "" {
		if envConfig, exists := e.config.GetEnvironment(sc.Config.Environment); exists {
			for k, v := range envConfig.Variables {
				scenarioContext.SetLocal(k, v)
			}
		}
	}

	// Execute before hook
	if sc.Before != nil {
		for _, step := range sc.Before.Steps {
			stepResult := e.executeStep(&step, scenarioContext)
			result.Steps = append(result.Steps, stepResult)
			if stepResult.Status == "failed" {
				result.Status = "failed"
				result.Error = fmt.Sprintf("Before hook step '%s' failed", step.Name)
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result
			}
		}
	}

	// Execute setup steps (legacy)
	if len(sc.Setup) > 0 {
		for _, step := range sc.Setup {
			stepResult := e.executeStep(&step, scenarioContext)
			result.Steps = append(result.Steps, stepResult)
			if stepResult.Status == "failed" && sc.Config != nil && sc.Config.FailFast {
				result.Status = "failed"
				result.Error = fmt.Sprintf("Setup step '%s' failed", step.Name)
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result
			}
		}
	}

	// Execute main steps (legacy format)
	if len(sc.Steps) > 0 {
		for _, step := range sc.Steps {
			stepResult := e.executeStep(&step, scenarioContext)
			result.Steps = append(result.Steps, stepResult)

			if stepResult.Status == "failed" && sc.Config != nil && sc.Config.FailFast {
				result.Status = "failed"
				result.Error = fmt.Sprintf("Step '%s' failed", step.Name)
				break
			}
		}
	}

	// Execute test groups (new format)
	if len(sc.Tests) > 0 {
		if sc.Config != nil && sc.Config.Parallel {
			// Execute tests concurrently
			e.executeTestsConcurrently(sc.Tests, scenarioContext, &result)
		} else {
			// Execute tests sequentially
			for testName, test := range sc.Tests {
				e.executeTestGroup(test, testName, scenarioContext, &result)
			}
		}
	}

	// Execute teardown steps (legacy)
	if len(sc.Teardown) > 0 {
		for _, step := range sc.Teardown {
			stepResult := e.executeStep(&step, scenarioContext)
			result.Steps = append(result.Steps, stepResult)
		}
	}

	// Execute after hook
	if sc.After != nil {
		for _, step := range sc.After.Steps {
			stepResult := e.executeStep(&step, scenarioContext)
			result.Steps = append(result.Steps, stepResult)
		}
	}

	// Determine overall scenario status
	if result.Status == "" {
		result.Status = "passed"
		for _, stepResult := range result.Steps {
			if stepResult.Status == "failed" {
				result.Status = "failed"
				break
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Variables = scenarioContext.GetAll()

	return result
}

func (e *Engine) executeTestsConcurrently(tests map[string]*scenario.TestGroup, varContext *variables.Context, result *reporting.ScenarioResult) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for testName, test := range tests {
		wg.Add(1)
		go func(name string, t *scenario.TestGroup) {
			defer wg.Done()
			testVarContext := varContext.Clone()

			mu.Lock()
			e.executeTestGroup(t, name, testVarContext, result)
			mu.Unlock()
		}(testName, test)
	}

	wg.Wait()
}

func (e *Engine) executeTestGroup(test *scenario.TestGroup, testName string, varContext *variables.Context, result *reporting.ScenarioResult) {
	if test.Skip {
		return
	}

	// Add test-level environment variables
	for k, v := range test.Env {
		varContext.SetLocal(k, v)
	}

	// Execute test steps
	for _, step := range test.Steps {
		stepResult := e.executeStep(&step, varContext)
		result.Steps = append(result.Steps, stepResult)

		if stepResult.Status == "failed" && !test.ContinueOnFail {
			result.Status = "failed"
			result.Error = fmt.Sprintf("Test '%s' step '%s' failed", testName, step.Name)
			break
		}
	}
}

func (e *Engine) executeStep(step *scenario.Step, varContext *variables.Context) reporting.StepResult {
	result := reporting.StepResult{
		Step:      step,
		StartTime: time.Now(),
		Variables: make(map[string]interface{}),
	}

	// Clear step-specific variables
	varContext.ClearStep()

	// Add step variables
	for k, v := range step.Variables {
		varContext.SetStep(k, v)
	}

	// Check condition if specified
	if step.Condition != "" {
		// TODO: Implement condition evaluation
		// For now, assume all conditions pass
	}

	// Handle new HTTP step format
	if step.HTTP != nil {
		response, err := e.executeHTTPStepNew(step, varContext)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
		} else {
			result.Response = response
			result.Status = "passed"

			// Process captures
			e.processCaptures(step.Capture, response, varContext)

			// Run checks (new format assertions)
			if len(step.Check) > 0 || len(step.HTTP.Check) > 0 {
				checks := step.Check
				if len(step.HTTP.Check) > 0 {
					// Merge HTTP-specific checks
					if checks == nil {
						checks = make(map[string]interface{})
					}
					for k, v := range step.HTTP.Check {
						checks[k] = v
					}
				}
				assertionResults := e.processChecks(checks, response, varContext)
				result.Assertions = assertionResults

				// Check if any assertion failed
				for _, assertionResult := range assertionResults {
					if !assertionResult.Passed {
						result.Status = "failed"
						break
					}
				}
			}
		}
	} else {
		// Execute based on step type (legacy format)
		switch step.Type {
		case "http":
			response, err := e.executeHTTPStep(step, varContext)
			if err != nil {
				result.Status = "failed"
				result.Error = err.Error()
			} else {
				result.Response = response
				result.Status = "passed"

				// Run assertions
				if len(step.Assertions) > 0 {
					assertionEngine := assertions.NewEngine(varContext)
					assertionResults, err := assertionEngine.RunAssertions(step.Assertions, response)
					if err != nil {
						result.Status = "failed"
						result.Error = fmt.Sprintf("Assertion error: %v", err)
					} else {
						result.Assertions = assertionResults

						// Check if any assertion failed
						for _, assertionResult := range assertionResults {
							if !assertionResult.Passed {
								result.Status = "failed"
								break
							}
						}
					}
				}

				// Extract variables from response
				e.extractVariables(step, response, varContext)
			}
		default:
			result.Status = "failed"
			result.Error = fmt.Sprintf("Unsupported step type: %s", step.Type)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Variables = varContext.GetAll()

	return result
}

func (e *Engine) executeHTTPStep(step *scenario.Step, varContext *variables.Context) (interface{}, error) {
	// Interpolate request values
	interpolatedStep := *step

	// Interpolate URL
	if step.Request.URL != "" {
		url, err := varContext.InterpolateString(step.Request.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate URL: %w", err)
		}
		interpolatedStep.Request.URL = url
	}

	// Interpolate headers
	if len(step.Request.Headers) > 0 {
		headers, err := varContext.InterpolateMap(step.Request.Headers)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate headers: %w", err)
		}
		interpolatedStep.Request.Headers = headers
	}

	// Interpolate query parameters
	if len(step.Request.Query) > 0 {
		query, err := varContext.InterpolateMap(step.Request.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate query: %w", err)
		}
		interpolatedStep.Request.Query = query
	}

	// Interpolate body
	if step.Request.Body != nil {
		body, err := varContext.InterpolateInterface(step.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate body: %w", err)
		}
		interpolatedStep.Request.Body = body
	}

	// Execute HTTP request
	response, err := e.httpClient.Execute(&interpolatedStep)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Convert response to map for easy access
	responseMap := map[string]interface{}{
		"status_code": response.StatusCode,
		"headers":     response.Headers,
		"body":        response.Body,
		"body_text":   response.BodyText,
		"duration":    response.Duration,
		"size":        response.Size,
	}

	return responseMap, nil
}

func (e *Engine) extractVariables(step *scenario.Step, response interface{}, varContext *variables.Context) {
	// TODO: Implement variable extraction from response based on step configuration
	// This would parse extraction rules like:
	// variables:
	//   token: "json:access_token"
	//   user_id: "json:user.id"
	//   session_id: "header:Set-Cookie"

	// For now, we'll add basic response data as variables
	if responseMap, ok := response.(map[string]interface{}); ok {
		varContext.SetStep("last_status", responseMap["status_code"])
		varContext.SetStep("last_response", responseMap["body_text"])
	}
}

func (e *Engine) executeHTTPStepNew(step *scenario.Step, varContext *variables.Context) (interface{}, error) {
	// Convert new format to legacy format for HTTP client compatibility
	legacyStep := &scenario.Step{
		Name: step.Name,
		Type: "http",
		Request: scenario.Request{
			Method:  step.HTTP.Method,
			URL:     step.HTTP.URL,
			Headers: step.HTTP.Headers,
			Query:   step.HTTP.Query,
			Body:    step.HTTP.Body,
		},
	}

	// Handle JSON body
	if step.HTTP.JSON != nil {
		legacyStep.Request.Body = step.HTTP.JSON
		if legacyStep.Request.Headers == nil {
			legacyStep.Request.Headers = make(map[string]string)
		}
		legacyStep.Request.Headers["Content-Type"] = "application/json"
	}

	// Handle authentication
	if step.HTTP.Auth != nil {
		legacyStep.Request.Auth = step.HTTP.Auth
	}

	return e.executeHTTPStep(legacyStep, varContext)
}

func (e *Engine) processCaptures(captures map[string]scenario.Capture, response interface{}, varContext *variables.Context) {
	for name, capture := range captures {
		var value interface{}
		var err error

		switch {
		case capture.JSONPath != "":
			value, err = variables.ExtractFromResponse(response.(map[string]interface{}), "json:"+capture.JSONPath)
		case capture.Header != "":
			value, err = variables.ExtractFromResponse(response.(map[string]interface{}), "header:"+capture.Header)
		case capture.Regex != "":
			// TODO: Implement regex capture
			err = fmt.Errorf("regex capture not yet implemented")
		default:
			err = fmt.Errorf("unknown capture type")
		}

		if err == nil {
			varContext.SetStep(name, value)
		}
	}
}

func (e *Engine) processChecks(checks map[string]interface{}, response interface{}, varContext *variables.Context) []assertions.Result {
	var results []assertions.Result
	assertionEngine := assertions.NewEngine(varContext)

	// Convert checks to assertions format
	var assertionList []scenario.Assertion
	for checkType, expectedValue := range checks {
		assertion := scenario.Assertion{
			Type:     checkType,
			Operator: "eq",
			Value:    expectedValue,
		}

		// Handle special check types
		switch checkType {
		case "status":
			assertion.Type = "status_code"
		}

		assertionList = append(assertionList, assertion)
	}

	// Run assertions
	assertionResults, err := assertionEngine.RunAssertions(assertionList, response)
	if err != nil {
		// Create a failed result if assertion engine fails
		results = append(results, assertions.Result{
			Passed:  false,
			Message: fmt.Sprintf("Assertion engine error: %v", err),
		})
	} else {
		results = assertionResults
	}

	return results
}
