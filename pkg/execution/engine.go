package execution

import (
	"fmt"
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

	// Add scenario variables
	for k, v := range sc.Variables {
		scenarioContext.SetLocal(k, v)
	}

	// Apply environment-specific configuration if specified
	if sc.Config.Environment != "" {
		if envConfig, exists := e.config.GetEnvironment(sc.Config.Environment); exists {
			for k, v := range envConfig.Variables {
				scenarioContext.SetLocal(k, v)
			}
		}
	}

	// Execute setup steps
	if len(sc.Setup) > 0 {
		for _, step := range sc.Setup {
			stepResult := e.executeStep(&step, scenarioContext)
			result.Steps = append(result.Steps, stepResult)
			if stepResult.Status == "failed" && sc.Config.FailFast {
				result.Status = "failed"
				result.Error = fmt.Sprintf("Setup step '%s' failed", step.Name)
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result
			}
		}
	}

	// Execute main steps
	for _, step := range sc.Steps {
		stepResult := e.executeStep(&step, scenarioContext)
		result.Steps = append(result.Steps, stepResult)

		if stepResult.Status == "failed" && sc.Config.FailFast {
			result.Status = "failed"
			result.Error = fmt.Sprintf("Step '%s' failed", step.Name)
			break
		}
	}

	// Execute teardown steps
	if len(sc.Teardown) > 0 {
		for _, step := range sc.Teardown {
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

	// Execute based on step type
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
