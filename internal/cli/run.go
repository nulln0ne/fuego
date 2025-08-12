package cli

import (
	"fmt"
	"os"

	"github.com/nulln0ne/fuego/pkg/config"
	"github.com/nulln0ne/fuego/pkg/execution"
	"github.com/nulln0ne/fuego/pkg/reporting"
	"github.com/nulln0ne/fuego/pkg/scenario"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run [scenario file or directory]",
	Short: "Run API test scenarios",
	Long: `Run API test scenarios from YAML/JSON files.
	
Examples:
  fuego run test.yaml          Run a single test scenario
  fuego run tests/             Run all test scenarios in directory
  fuego run --parallel tests/  Run tests in parallel`,
	Args: cobra.MinimumNArgs(1),
	RunE: runScenarios,
}

var (
	parallel     bool
	timeout      int
	environment  string
	outputFormat string
	outputFile   string
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "run tests in parallel")
	runCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "timeout in seconds for each test")
	runCmd.Flags().StringVarP(&environment, "env", "e", "", "environment to use for variable substitution")
	runCmd.Flags().StringVarP(&outputFormat, "format", "f", "console", "output format (console, json, html, markdown)")
	runCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file path")
}

func runScenarios(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override with environment if specified
	if environment != "" {
		cfg = cfg.MergeEnvironment(environment)
	}

	// Create reporter
	reporterConfig := reporting.ReportConfig{
		Format:     outputFormat,
		OutputFile: outputFile,
		Verbose:    viper.GetBool("verbose"),
	}
	reporter := reporting.NewReporter(reporterConfig)

	// Create execution engine
	engine := execution.NewEngine(cfg, reporter)

	// Load scenarios
	var scenarios []*scenario.Scenario
	for _, arg := range args {
		stat, err := os.Stat(arg)
		if err != nil {
			return fmt.Errorf("failed to access %s: %w", arg, err)
		}

		if stat.IsDir() {
			dirScenarios, err := scenario.LoadScenariosFromDir(arg)
			if err != nil {
				return fmt.Errorf("failed to load scenarios from directory %s: %w", arg, err)
			}
			scenarios = append(scenarios, dirScenarios...)
		} else {
			sc, err := scenario.LoadScenario(arg)
			if err != nil {
				return fmt.Errorf("failed to load scenario %s: %w", arg, err)
			}
			scenarios = append(scenarios, sc)
		}
	}

	if len(scenarios) == 0 {
		return fmt.Errorf("no scenarios found")
	}

	fmt.Printf("Found %d scenario(s) to execute\n", len(scenarios))

	// Execute scenarios
	return engine.ExecuteScenarios(scenarios)
}
