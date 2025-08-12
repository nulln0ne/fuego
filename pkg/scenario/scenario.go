package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Scenario struct {
	Version     string                `yaml:"version" json:"version"`
	Name        string                `yaml:"name" json:"name"`
	Description string                `yaml:"description,omitempty" json:"description,omitempty"`
	Env         map[string]any        `yaml:"env,omitempty" json:"env,omitempty"`
	Variables   map[string]any        `yaml:"variables,omitempty" json:"variables,omitempty"`
	Config      *ScenarioConfig       `yaml:"config,omitempty" json:"config,omitempty"`
	Before      *TestGroup            `yaml:"before,omitempty" json:"before,omitempty"`
	Setup       []Step                `yaml:"setup,omitempty" json:"setup,omitempty"`
	Steps       []Step                `yaml:"steps,omitempty" json:"steps,omitempty"`
	Tests       map[string]*TestGroup `yaml:"tests,omitempty" json:"tests,omitempty"`
	Teardown    []Step                `yaml:"teardown,omitempty" json:"teardown,omitempty"`
	After       *TestGroup            `yaml:"after,omitempty" json:"after,omitempty"`
	Metadata    ScenarioMetadata      `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type TestGroup struct {
	Name           string         `yaml:"name,omitempty" json:"name,omitempty"`
	Env            map[string]any `yaml:"env,omitempty" json:"env,omitempty"`
	Skip           bool           `yaml:"skip,omitempty" json:"skip,omitempty"`
	ContinueOnFail bool           `yaml:"continueOnFail,omitempty" json:"continueOnFail,omitempty"`
	Steps          []Step         `yaml:"steps" json:"steps"`
}

type ScenarioConfig struct {
	HTTP        *HTTPConfig   `yaml:"http,omitempty" json:"http,omitempty"`
	Parallel    bool          `yaml:"parallel,omitempty" json:"parallel,omitempty"`
	Concurrency int           `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries     int           `yaml:"retries,omitempty" json:"retries,omitempty"`
	FailFast    bool          `yaml:"fail_fast,omitempty" json:"fail_fast,omitempty"`
	Environment string        `yaml:"environment,omitempty" json:"environment,omitempty"`
}

type HTTPConfig struct {
	Timeout         time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	FollowRedirects bool          `yaml:"followRedirects,omitempty" json:"followRedirects,omitempty"`
	VerifySSL       bool          `yaml:"verifySSL,omitempty" json:"verifySSL,omitempty"`
}

type ScenarioMetadata struct {
	Author     string            `yaml:"author,omitempty" json:"author,omitempty"`
	Version    string            `yaml:"version,omitempty" json:"version,omitempty"`
	Tags       []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	CreatedAt  time.Time         `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	ModifiedAt time.Time         `yaml:"modified_at,omitempty" json:"modified_at,omitempty"`
}

type Step struct {
	Name        string                 `yaml:"name" json:"name"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Type        string                 `yaml:"type,omitempty" json:"type,omitempty"` // http, grpc, websocket, etc.
	HTTP        *HTTPStep              `yaml:"http,omitempty" json:"http,omitempty"`
	Request     Request                `yaml:"request,omitempty" json:"request,omitempty"`
	Capture     map[string]Capture     `yaml:"capture,omitempty" json:"capture,omitempty"`
	Check       map[string]interface{} `yaml:"check,omitempty" json:"check,omitempty"`
	Assertions  []Assertion            `yaml:"assertions,omitempty" json:"assertions,omitempty"`
	Variables   map[string]any         `yaml:"variables,omitempty" json:"variables,omitempty"`
	Condition   string                 `yaml:"condition,omitempty" json:"condition,omitempty"`
	Loop        *LoopConfig            `yaml:"loop,omitempty" json:"loop,omitempty"`
	Retry       *RetryConfig           `yaml:"retry,omitempty" json:"retry,omitempty"`
	Timeout     time.Duration          `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	DependsOn   []string               `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Config      map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

type HTTPStep struct {
	URL     string                 `yaml:"url" json:"url"`
	Method  string                 `yaml:"method,omitempty" json:"method,omitempty"`
	Headers map[string]string      `yaml:"headers,omitempty" json:"headers,omitempty"`
	Query   map[string]string      `yaml:"query,omitempty" json:"query,omitempty"`
	Body    interface{}            `yaml:"body,omitempty" json:"body,omitempty"`
	JSON    interface{}            `yaml:"json,omitempty" json:"json,omitempty"`
	Auth    *AuthConfig            `yaml:"auth,omitempty" json:"auth,omitempty"`
	Check   map[string]interface{} `yaml:"check,omitempty" json:"check,omitempty"`
}

type Capture struct {
	JSONPath string `yaml:"jsonpath,omitempty" json:"jsonpath,omitempty"`
	Header   string `yaml:"header,omitempty" json:"header,omitempty"`
	Regex    string `yaml:"regex,omitempty" json:"regex,omitempty"`
}

type Request struct {
	Method         string                 `yaml:"method,omitempty" json:"method,omitempty"`
	URL            string                 `yaml:"url,omitempty" json:"url,omitempty"`
	Headers        map[string]string      `yaml:"headers,omitempty" json:"headers,omitempty"`
	Query          map[string]string      `yaml:"query,omitempty" json:"query,omitempty"`
	Body           interface{}            `yaml:"body,omitempty" json:"body,omitempty"`
	Auth           *AuthConfig            `yaml:"auth,omitempty" json:"auth,omitempty"`
	Cookies        map[string]string      `yaml:"cookies,omitempty" json:"cookies,omitempty"`
	Files          map[string]string      `yaml:"files,omitempty" json:"files,omitempty"`
	FollowRedirect bool                   `yaml:"follow_redirect,omitempty" json:"follow_redirect,omitempty"`
	Config         map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

type AuthConfig struct {
	Type     string                 `yaml:"type" json:"type"` // basic, bearer, oauth2, api_key
	Username string                 `yaml:"username,omitempty" json:"username,omitempty"`
	Password string                 `yaml:"password,omitempty" json:"password,omitempty"`
	Token    string                 `yaml:"token,omitempty" json:"token,omitempty"`
	Config   map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

type Assertion struct {
	Type        string      `yaml:"type" json:"type"` // status, header, body, json_path, xpath, regex, etc.
	Field       string      `yaml:"field,omitempty" json:"field,omitempty"`
	Operator    string      `yaml:"operator,omitempty" json:"operator,omitempty"` // eq, ne, gt, lt, contains, matches, etc.
	Value       interface{} `yaml:"value,omitempty" json:"value,omitempty"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Optional    bool        `yaml:"optional,omitempty" json:"optional,omitempty"`
}

type LoopConfig struct {
	Type      string      `yaml:"type" json:"type"` // count, while, foreach
	Count     int         `yaml:"count,omitempty" json:"count,omitempty"`
	Condition string      `yaml:"condition,omitempty" json:"condition,omitempty"`
	Items     interface{} `yaml:"items,omitempty" json:"items,omitempty"`
	Variable  string      `yaml:"variable,omitempty" json:"variable,omitempty"`
}

type RetryConfig struct {
	Count     int           `yaml:"count" json:"count"`
	Delay     time.Duration `yaml:"delay,omitempty" json:"delay,omitempty"`
	Backoff   string        `yaml:"backoff,omitempty" json:"backoff,omitempty"` // linear, exponential
	Condition string        `yaml:"condition,omitempty" json:"condition,omitempty"`
}

func LoadScenario(filename string) (*Scenario, error) {
	if !filepath.IsAbs(filename) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		filename = filepath.Join(wd, filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenario file %s: %w", filename, err)
	}

	var scenario Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("failed to parse scenario file %s: %w", filename, err)
	}

	if err := validateScenario(&scenario); err != nil {
		return nil, fmt.Errorf("invalid scenario in %s: %w", filename, err)
	}

	return &scenario, nil
}

func LoadScenariosFromDir(dir string) ([]*Scenario, error) {
	var scenarios []*Scenario

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		scenarioPath := filepath.Join(dir, entry.Name())
		scenario, err := LoadScenario(scenarioPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load scenario %s: %w", scenarioPath, err)
		}

		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

func validateScenario(scenario *Scenario) error {
	if scenario.Name == "" {
		return fmt.Errorf("scenario name is required")
	}

	// Validate either steps (legacy) or tests (new format)
	if len(scenario.Steps) == 0 && len(scenario.Tests) == 0 {
		return fmt.Errorf("scenario must have either steps or tests")
	}

	// Validate legacy steps
	for i, step := range scenario.Steps {
		if err := validateStep(&step, i); err != nil {
			return fmt.Errorf("step %d (%s): %w", i+1, step.Name, err)
		}
	}

	// Validate test groups
	for testName, test := range scenario.Tests {
		if err := validateTestGroup(test, testName); err != nil {
			return fmt.Errorf("test '%s': %w", testName, err)
		}
	}

	// Validate before/after groups
	if scenario.Before != nil {
		if err := validateTestGroup(scenario.Before, "before"); err != nil {
			return fmt.Errorf("before group: %w", err)
		}
	}

	if scenario.After != nil {
		if err := validateTestGroup(scenario.After, "after"); err != nil {
			return fmt.Errorf("after group: %w", err)
		}
	}

	return nil
}

func validateTestGroup(group *TestGroup, name string) error {
	if len(group.Steps) == 0 {
		return fmt.Errorf("test group must have at least one step")
	}

	for i, step := range group.Steps {
		if err := validateStep(&step, i); err != nil {
			return fmt.Errorf("step %d (%s): %w", i+1, step.Name, err)
		}
	}

	return nil
}

func validateStep(step *Step, index int) error {
	if step.Name == "" {
		return fmt.Errorf("step name is required")
	}

	// Handle new HTTP step format
	if step.HTTP != nil {
		if step.HTTP.URL == "" {
			return fmt.Errorf("HTTP step URL is required")
		}
		if step.HTTP.Method == "" {
			step.HTTP.Method = "GET" // default to GET
		}
		return nil
	}

	// Handle legacy format
	if step.Type == "" {
		step.Type = "http" // default to HTTP
	}

	// Validate step type
	validTypes := []string{"http", "grpc", "websocket", "trpc", "soap", "custom"}
	valid := false
	for _, validType := range validTypes {
		if step.Type == validType {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid step type: %s", step.Type)
	}

	// Type-specific validation
	switch step.Type {
	case "http":
		if step.Request.Method == "" {
			step.Request.Method = "GET" // default to GET
		}
		if step.Request.URL == "" && step.HTTP == nil {
			return fmt.Errorf("HTTP request URL is required")
		}
	}

	return nil
}
