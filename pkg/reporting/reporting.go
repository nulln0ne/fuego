package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nulln0ne/fuego/pkg/assertions"
	"github.com/nulln0ne/fuego/pkg/scenario"
)

type Report struct {
	Summary   Summary          `json:"summary"`
	Scenarios []ScenarioResult `json:"scenarios"`
	StartTime time.Time        `json:"start_time"`
	EndTime   time.Time        `json:"end_time"`
	Duration  time.Duration    `json:"duration"`
	Config    ReportConfig     `json:"config,omitempty"`
}

type Summary struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	Failed   int     `json:"failed"`
	Skipped  int     `json:"skipped"`
	PassRate float64 `json:"pass_rate"`
}

type ScenarioResult struct {
	Scenario  *scenario.Scenario     `json:"scenario"`
	Status    string                 `json:"status"` // passed, failed, skipped
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Steps     []StepResult           `json:"steps"`
	Error     string                 `json:"error,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type StepResult struct {
	Step       *scenario.Step         `json:"step"`
	Status     string                 `json:"status"` // passed, failed, skipped
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Request    interface{}            `json:"request,omitempty"`
	Response   interface{}            `json:"response,omitempty"`
	Assertions []assertions.Result    `json:"assertions,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
}

type ReportConfig struct {
	Format      string `json:"format"` // console, json, html, markdown
	OutputFile  string `json:"output_file,omitempty"`
	Verbose     bool   `json:"verbose"`
	IncludeBody bool   `json:"include_body"`
}

type Reporter struct {
	config ReportConfig
	report *Report
}

func NewReporter(config ReportConfig) *Reporter {
	return &Reporter{
		config: config,
		report: &Report{
			Summary:   Summary{},
			Scenarios: make([]ScenarioResult, 0),
			StartTime: time.Now(),
			Config:    config,
		},
	}
}

func (r *Reporter) Start() {
	r.report.StartTime = time.Now()
}

func (r *Reporter) End() {
	r.report.EndTime = time.Now()
	r.report.Duration = r.report.EndTime.Sub(r.report.StartTime)
	r.calculateSummary()
}

func (r *Reporter) AddScenarioResult(result ScenarioResult) {
	r.report.Scenarios = append(r.report.Scenarios, result)
}

func (r *Reporter) calculateSummary() {
	total := len(r.report.Scenarios)
	passed := 0
	failed := 0
	skipped := 0

	for _, scenario := range r.report.Scenarios {
		switch scenario.Status {
		case "passed":
			passed++
		case "failed":
			failed++
		case "skipped":
			skipped++
		}
	}

	passRate := 0.0
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}

	r.report.Summary = Summary{
		Total:    total,
		Passed:   passed,
		Failed:   failed,
		Skipped:  skipped,
		PassRate: passRate,
	}
}

func (r *Reporter) GenerateReport() error {
	r.End()

	switch r.config.Format {
	case "json":
		return r.generateJSONReport()
	case "html":
		return r.generateHTMLReport()
	case "markdown":
		return r.generateMarkdownReport()
	default:
		return r.generateConsoleReport()
	}
}

func (r *Reporter) generateConsoleReport() error {
	// Print summary
	fmt.Printf("\n=== Test Results Summary ===\n")
	fmt.Printf("Total Scenarios: %d\n", r.report.Summary.Total)
	fmt.Printf("Passed: %d\n", r.report.Summary.Passed)
	fmt.Printf("Failed: %d\n", r.report.Summary.Failed)
	fmt.Printf("Skipped: %d\n", r.report.Summary.Skipped)
	fmt.Printf("Pass Rate: %.2f%%\n", r.report.Summary.PassRate)
	fmt.Printf("Duration: %v\n", r.report.Duration)

	// Print scenario details
	for _, scenario := range r.report.Scenarios {
		status := "✓"
		if scenario.Status == "failed" {
			status = "✗"
		} else if scenario.Status == "skipped" {
			status = "⊖"
		}

		fmt.Printf("\n%s %s (%v)\n", status, scenario.Scenario.Name, scenario.Duration)

		if r.config.Verbose {
			for _, step := range scenario.Steps {
				stepStatus := "  ✓"
				if step.Status == "failed" {
					stepStatus = "  ✗"
				} else if step.Status == "skipped" {
					stepStatus = "  ⊖"
				}

				fmt.Printf("%s %s (%v)\n", stepStatus, step.Step.Name, step.Duration)

				if step.Error != "" {
					fmt.Printf("    Error: %s\n", step.Error)
				}

				// Print assertion results
				for _, assertion := range step.Assertions {
					assertionStatus := "    ✓"
					if !assertion.Passed {
						assertionStatus = "    ✗"
					}
					fmt.Printf("%s %s\n", assertionStatus, assertion.Message)
				}
			}
		}

		if scenario.Error != "" {
			fmt.Printf("  Error: %s\n", scenario.Error)
		}
	}

	return nil
}

func (r *Reporter) generateJSONReport() error {
	jsonData, err := json.MarshalIndent(r.report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	if r.config.OutputFile != "" {
		return os.WriteFile(r.config.OutputFile, jsonData, 0644)
	}

	fmt.Println(string(jsonData))
	return nil
}

func (r *Reporter) generateHTMLReport() error {
	html := r.generateHTMLContent()

	if r.config.OutputFile != "" {
		return os.WriteFile(r.config.OutputFile, []byte(html), 0644)
	}

	fmt.Println(html)
	return nil
}

func (r *Reporter) generateHTMLContent() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Fuego Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .summary { background: #f5f5f5; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .scenario { border: 1px solid #ddd; margin: 10px 0; border-radius: 5px; }
        .scenario-header { padding: 15px; background: #f9f9f9; font-weight: bold; }
        .scenario.passed .scenario-header { background: #d4edda; color: #155724; }
        .scenario.failed .scenario-header { background: #f8d7da; color: #721c24; }
        .steps { padding: 15px; }
        .step { margin: 10px 0; padding: 10px; border-left: 3px solid #ddd; }
        .step.passed { border-left-color: #28a745; }
        .step.failed { border-left-color: #dc3545; }
        .assertions { margin-left: 20px; font-size: 0.9em; }
        .assertion.passed { color: #28a745; }
        .assertion.failed { color: #dc3545; }
    </style>
</head>
<body>
    <h1>Fuego Test Report</h1>
    
    <div class="summary">
        <h2>Summary</h2>
        <p>Total Scenarios: %d</p>
        <p>Passed: %d</p>
        <p>Failed: %d</p>
        <p>Skipped: %d</p>
        <p>Pass Rate: %.2f%%</p>
        <p>Duration: %v</p>
    </div>

    <div class="scenarios">
        <h2>Scenarios</h2>
        %s
    </div>
</body>
</html>`,
		r.report.Summary.Total,
		r.report.Summary.Passed,
		r.report.Summary.Failed,
		r.report.Summary.Skipped,
		r.report.Summary.PassRate,
		r.report.Duration,
		r.generateScenariosHTML(),
	)
}

func (r *Reporter) generateScenariosHTML() string {
	html := ""
	for _, scenario := range r.report.Scenarios {
		stepsHTML := ""
		for _, step := range scenario.Steps {
			assertionsHTML := ""
			for _, assertion := range step.Assertions {
				status := "passed"
				if !assertion.Passed {
					status = "failed"
				}
				assertionsHTML += fmt.Sprintf(`<div class="assertion %s">%s</div>`, status, assertion.Message)
			}

			stepsHTML += fmt.Sprintf(`
				<div class="step %s">
					<strong>%s</strong> (%v)
					<div class="assertions">%s</div>
				</div>`,
				step.Status, step.Step.Name, step.Duration, assertionsHTML)
		}

		html += fmt.Sprintf(`
			<div class="scenario %s">
				<div class="scenario-header">%s (%v)</div>
				<div class="steps">%s</div>
			</div>`,
			scenario.Status, scenario.Scenario.Name, scenario.Duration, stepsHTML)
	}
	return html
}

func (r *Reporter) generateMarkdownReport() error {
	markdown := r.generateMarkdownContent()

	if r.config.OutputFile != "" {
		return os.WriteFile(r.config.OutputFile, []byte(markdown), 0644)
	}

	fmt.Println(markdown)
	return nil
}

func (r *Reporter) generateMarkdownContent() string {
	scenariosMarkdown := ""
	for _, scenario := range r.report.Scenarios {
		status := "✅"
		if scenario.Status == "failed" {
			status = "❌"
		} else if scenario.Status == "skipped" {
			status = "⏭️"
		}

		scenariosMarkdown += fmt.Sprintf("## %s %s\n\n", status, scenario.Scenario.Name)
		scenariosMarkdown += fmt.Sprintf("**Duration:** %v\n\n", scenario.Duration)

		if len(scenario.Steps) > 0 {
			scenariosMarkdown += "### Steps\n\n"
			for _, step := range scenario.Steps {
				stepStatus := "✅"
				if step.Status == "failed" {
					stepStatus = "❌"
				} else if step.Status == "skipped" {
					stepStatus = "⏭️"
				}

				scenariosMarkdown += fmt.Sprintf("- %s **%s** (%v)\n", stepStatus, step.Step.Name, step.Duration)

				if len(step.Assertions) > 0 {
					for _, assertion := range step.Assertions {
						assertionStatus := "✅"
						if !assertion.Passed {
							assertionStatus = "❌"
						}
						scenariosMarkdown += fmt.Sprintf("  - %s %s\n", assertionStatus, assertion.Message)
					}
				}
			}
			scenariosMarkdown += "\n"
		}
	}

	return fmt.Sprintf(`# Fuego Test Report

## Summary

- **Total Scenarios:** %d
- **Passed:** %d
- **Failed:** %d
- **Skipped:** %d
- **Pass Rate:** %.2f%%
- **Duration:** %v

## Scenarios

%s`,
		r.report.Summary.Total,
		r.report.Summary.Passed,
		r.report.Summary.Failed,
		r.report.Summary.Skipped,
		r.report.Summary.PassRate,
		r.report.Duration,
		scenariosMarkdown,
	)
}
