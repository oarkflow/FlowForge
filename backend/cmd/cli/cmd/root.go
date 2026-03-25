package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func rootCmd() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	// Global flags
	fs := flag.NewFlagSet("flowforge", flag.ContinueOnError)
	fs.StringVar(&serverURL, "server", "", "FlowForge server URL")
	fs.StringVar(&apiToken, "token", "", "API token")
	fs.StringVar(&output, "output", "table", "Output format (table, json)")
	fs.StringVar(&cfgFile, "config", "", "Config file path")

	// Find the subcommand
	args := os.Args[1:]
	subcommand := ""
	subArgs := []string{}

	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			subcommand = arg
			subArgs = args[i+1:]
			break
		}
		// Parse global flags before subcommand
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			fs.Parse(args[:i+1])
			subcommand = args[i+1]
			if i+2 < len(args) {
				subArgs = args[i+2:]
			}
			break
		}
	}

	initConfig()

	switch subcommand {
	case "login":
		return loginCmd(subArgs)
	case "pipelines":
		return pipelinesCmd(subArgs)
	case "runs":
		return runsCmd(subArgs)
	case "agents":
		return agentsCmd(subArgs)
	case "secrets":
		return secretsCmd(subArgs)
	case "artifacts":
		return artifactsCmd(subArgs)
	case "config":
		return configCmd(subArgs)
	case "version":
		fmt.Println("flowforge version 1.0.0")
		return nil
	case "help", "":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", subcommand)
	}
}

func printUsage() {
	fmt.Println(`FlowForge CLI - CI/CD Pipeline Management

Usage:
  flowforge <command> [flags]

Commands:
  login       Authenticate with FlowForge server
  pipelines   Manage pipelines (list, get, create, update, delete, trigger)
  runs        Manage pipeline runs (list, get, logs, cancel, rerun)
  agents      Manage build agents (list, get, drain)
  secrets     Manage secrets (list, set, delete)
  artifacts   Manage build artifacts (list, download)
  config      View/set CLI configuration
  version     Print version information

Global Flags:
  --server    FlowForge server URL (default: http://localhost:8081)
  --token     API authentication token
  --output    Output format: table, json (default: table)
  --config    Config file path (default: ~/.flowforge/config.yaml)

Use "flowforge <command> --help" for more information about a command.`)
}

// loginCmd handles the "login" subcommand.
func loginCmd(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	key := fs.String("api-key", "", "API key for authentication")
	fs.Parse(args)

	if *key == "" {
		fmt.Print("Enter API key: ")
		fmt.Scanln(key)
	}

	if *key == "" {
		return fmt.Errorf("API key is required")
	}

	// Save to config file
	home, _ := os.UserHomeDir()
	cfgDir := filepath.Join(home, ".flowforge")
	os.MkdirAll(cfgDir, 0755)
	cfgPath := filepath.Join(cfgDir, "config.yaml")

	content := fmt.Sprintf("server_url: %s\napi_token: %s\n", serverURL, *key)
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("✓ Logged in to %s\n", serverURL)
	fmt.Printf("  Config saved to %s\n", cfgPath)
	return nil
}

// pipelinesCmd handles the "pipelines" subcommand.
func pipelinesCmd(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: flowforge pipelines <list|get|create|delete|trigger> [flags]")
		return nil
	}

	client := newClient()
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		fs := flag.NewFlagSet("pipelines list", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		fs.Parse(subArgs)

		if *projectID == "" {
			return fmt.Errorf("--project-id is required")
		}

		data, err := client.get(fmt.Sprintf("/api/v1/projects/%s/pipelines", *projectID))
		if err != nil {
			return err
		}

		if output == "json" {
			printJSON(data)
			return nil
		}

		var result struct {
			Pipelines []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				IsActive bool   `json:"is_active"`
			} `json:"pipelines"`
		}
		json.Unmarshal(data, &result)

		headers := []string{"ID", "NAME", "ACTIVE"}
		var rows [][]string
		for _, p := range result.Pipelines {
			active := "yes"
			if !p.IsActive {
				active = "no"
			}
			rows = append(rows, []string{p.ID, p.Name, active})
		}
		printTable(headers, rows)
		return nil

	case "get":
		if len(subArgs) < 2 {
			return fmt.Errorf("usage: flowforge pipelines get --project-id <pid> <pipeline-id>")
		}
		fs := flag.NewFlagSet("pipelines get", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		fs.Parse(subArgs)
		pipelineID := fs.Arg(0)

		data, err := client.get(fmt.Sprintf("/api/v1/projects/%s/pipelines/%s", *projectID, pipelineID))
		if err != nil {
			return err
		}
		printJSON(data)
		return nil

	case "trigger":
		fs := flag.NewFlagSet("pipelines trigger", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		branch := fs.String("branch", "main", "Branch to trigger")
		fs.Parse(subArgs)
		pipelineID := fs.Arg(0)

		if *projectID == "" || pipelineID == "" {
			return fmt.Errorf("usage: flowforge pipelines trigger --project-id <pid> <pipeline-id>")
		}

		body := map[string]string{"branch": *branch}
		data, err := client.post(fmt.Sprintf("/api/v1/projects/%s/pipelines/%s/trigger", *projectID, pipelineID), body)
		if err != nil {
			return err
		}

		var result struct {
			RunID string `json:"run_id"`
		}
		json.Unmarshal(data, &result)
		fmt.Printf("✓ Pipeline triggered. Run ID: %s\n", result.RunID)
		return nil

	case "delete":
		fs := flag.NewFlagSet("pipelines delete", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		fs.Parse(subArgs)
		pipelineID := fs.Arg(0)

		_, err := client.del(fmt.Sprintf("/api/v1/projects/%s/pipelines/%s", *projectID, pipelineID))
		if err != nil {
			return err
		}
		fmt.Println("✓ Pipeline deleted")
		return nil

	default:
		return fmt.Errorf("unknown pipelines subcommand: %s", sub)
	}
}

// runsCmd handles the "runs" subcommand.
func runsCmd(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: flowforge runs <list|get|logs|cancel|rerun> [flags]")
		return nil
	}

	client := newClient()
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		fs := flag.NewFlagSet("runs list", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		pipelineID := fs.String("pipeline-id", "", "Pipeline ID")
		fs.Parse(subArgs)

		if *projectID == "" || *pipelineID == "" {
			return fmt.Errorf("--project-id and --pipeline-id are required")
		}

		data, err := client.get(fmt.Sprintf("/api/v1/projects/%s/pipelines/%s/runs", *projectID, *pipelineID))
		if err != nil {
			return err
		}

		if output == "json" {
			printJSON(data)
			return nil
		}

		var result struct {
			Runs []struct {
				ID     string `json:"id"`
				Number int    `json:"number"`
				Status string `json:"status"`
				Branch string `json:"branch"`
			} `json:"runs"`
		}
		json.Unmarshal(data, &result)

		headers := []string{"ID", "NUMBER", "STATUS", "BRANCH"}
		var rows [][]string
		for _, r := range result.Runs {
			rows = append(rows, []string{r.ID, fmt.Sprintf("#%d", r.Number), r.Status, r.Branch})
		}
		printTable(headers, rows)
		return nil

	case "get":
		fs := flag.NewFlagSet("runs get", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		pipelineID := fs.String("pipeline-id", "", "Pipeline ID")
		fs.Parse(subArgs)
		runID := fs.Arg(0)

		data, err := client.get(fmt.Sprintf("/api/v1/projects/%s/pipelines/%s/runs/%s", *projectID, *pipelineID, runID))
		if err != nil {
			return err
		}
		printJSON(data)
		return nil

	case "logs":
		fs := flag.NewFlagSet("runs logs", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		pipelineID := fs.String("pipeline-id", "", "Pipeline ID")
		fs.Parse(subArgs)
		runID := fs.Arg(0)

		data, err := client.get(fmt.Sprintf("/api/v1/projects/%s/pipelines/%s/runs/%s/logs", *projectID, *pipelineID, runID))
		if err != nil {
			return err
		}

		var logs struct {
			Logs []struct {
				Stream  string `json:"stream"`
				Content string `json:"content"`
			} `json:"logs"`
		}
		if json.Unmarshal(data, &logs) == nil {
			for _, l := range logs.Logs {
				fmt.Print(l.Content)
			}
		} else {
			fmt.Print(string(data))
		}
		return nil

	case "cancel":
		fs := flag.NewFlagSet("runs cancel", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		pipelineID := fs.String("pipeline-id", "", "Pipeline ID")
		fs.Parse(subArgs)
		runID := fs.Arg(0)

		_, err := client.post(fmt.Sprintf("/api/v1/projects/%s/pipelines/%s/runs/%s/cancel", *projectID, *pipelineID, runID), nil)
		if err != nil {
			return err
		}
		fmt.Println("✓ Run cancelled")
		return nil

	case "rerun":
		fs := flag.NewFlagSet("runs rerun", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		pipelineID := fs.String("pipeline-id", "", "Pipeline ID")
		fs.Parse(subArgs)
		runID := fs.Arg(0)

		data, err := client.post(fmt.Sprintf("/api/v1/projects/%s/pipelines/%s/runs/%s/rerun", *projectID, *pipelineID, runID), nil)
		if err != nil {
			return err
		}
		var result struct {
			RunID string `json:"run_id"`
		}
		json.Unmarshal(data, &result)
		fmt.Printf("✓ Pipeline rerun started. Run ID: %s\n", result.RunID)
		return nil

	default:
		return fmt.Errorf("unknown runs subcommand: %s", sub)
	}
}

// agentsCmd handles the "agents" subcommand.
func agentsCmd(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: flowforge agents <list|get|drain> [flags]")
		return nil
	}

	client := newClient()
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		data, err := client.get("/api/v1/agents")
		if err != nil {
			return err
		}

		if output == "json" {
			printJSON(data)
			return nil
		}

		var result struct {
			Agents []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Status   string `json:"status"`
				Executor string `json:"executor"`
			} `json:"agents"`
		}
		json.Unmarshal(data, &result)

		headers := []string{"ID", "NAME", "STATUS", "EXECUTOR"}
		var rows [][]string
		for _, a := range result.Agents {
			rows = append(rows, []string{a.ID, a.Name, a.Status, a.Executor})
		}
		printTable(headers, rows)
		return nil

	case "get":
		if len(subArgs) == 0 {
			return fmt.Errorf("usage: flowforge agents get <agent-id>")
		}
		data, err := client.get(fmt.Sprintf("/api/v1/agents/%s", subArgs[0]))
		if err != nil {
			return err
		}
		printJSON(data)
		return nil

	case "drain":
		if len(subArgs) == 0 {
			return fmt.Errorf("usage: flowforge agents drain <agent-id>")
		}
		_, err := client.post(fmt.Sprintf("/api/v1/agents/%s/drain", subArgs[0]), nil)
		if err != nil {
			return err
		}
		fmt.Println("✓ Agent set to draining")
		return nil

	default:
		return fmt.Errorf("unknown agents subcommand: %s", sub)
	}
}

// secretsCmd handles the "secrets" subcommand.
func secretsCmd(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: flowforge secrets <list|set|delete> [flags]")
		return nil
	}

	client := newClient()
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		fs := flag.NewFlagSet("secrets list", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		fs.Parse(subArgs)

		if *projectID == "" {
			return fmt.Errorf("--project-id is required")
		}

		data, err := client.get(fmt.Sprintf("/api/v1/projects/%s/secrets", *projectID))
		if err != nil {
			return err
		}

		if output == "json" {
			printJSON(data)
			return nil
		}

		var result struct {
			Secrets []struct {
				ID  string `json:"id"`
				Key string `json:"key"`
			} `json:"secrets"`
		}
		json.Unmarshal(data, &result)

		headers := []string{"ID", "KEY", "VALUE"}
		var rows [][]string
		for _, s := range result.Secrets {
			rows = append(rows, []string{s.ID, s.Key, "********"})
		}
		printTable(headers, rows)
		return nil

	case "set":
		fs := flag.NewFlagSet("secrets set", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		key := fs.String("key", "", "Secret key")
		value := fs.String("value", "", "Secret value")
		fs.Parse(subArgs)

		if *projectID == "" || *key == "" || *value == "" {
			return fmt.Errorf("--project-id, --key, and --value are required")
		}

		body := map[string]string{"key": *key, "value": *value}
		_, err := client.post(fmt.Sprintf("/api/v1/projects/%s/secrets", *projectID), body)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Secret '%s' set\n", *key)
		return nil

	case "delete":
		fs := flag.NewFlagSet("secrets delete", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		fs.Parse(subArgs)
		secretID := fs.Arg(0)

		if *projectID == "" || secretID == "" {
			return fmt.Errorf("--project-id and secret ID are required")
		}

		_, err := client.del(fmt.Sprintf("/api/v1/projects/%s/secrets/%s", *projectID, secretID))
		if err != nil {
			return err
		}
		fmt.Println("✓ Secret deleted")
		return nil

	default:
		return fmt.Errorf("unknown secrets subcommand: %s", sub)
	}
}

// artifactsCmd handles the "artifacts" subcommand.
func artifactsCmd(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: flowforge artifacts <list|download> [flags]")
		return nil
	}

	client := newClient()
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "list":
		fs := flag.NewFlagSet("artifacts list", flag.ContinueOnError)
		projectID := fs.String("project-id", "", "Project ID")
		pipelineID := fs.String("pipeline-id", "", "Pipeline ID")
		runID := fs.String("run-id", "", "Run ID")
		fs.Parse(subArgs)

		if *projectID == "" || *pipelineID == "" || *runID == "" {
			return fmt.Errorf("--project-id, --pipeline-id, and --run-id are required")
		}

		data, err := client.get(fmt.Sprintf("/api/v1/projects/%s/pipelines/%s/runs/%s/artifacts", *projectID, *pipelineID, *runID))
		if err != nil {
			return err
		}

		if output == "json" {
			printJSON(data)
			return nil
		}

		var result struct {
			Artifacts []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Size int64  `json:"size_bytes"`
			} `json:"artifacts"`
		}
		json.Unmarshal(data, &result)

		headers := []string{"ID", "NAME", "SIZE"}
		var rows [][]string
		for _, a := range result.Artifacts {
			rows = append(rows, []string{a.ID, a.Name, formatBytes(a.Size)})
		}
		printTable(headers, rows)
		return nil

	case "download":
		fs := flag.NewFlagSet("artifacts download", flag.ContinueOnError)
		outputPath := fs.String("output", "", "Output file path")
		fs.Parse(subArgs)
		artifactID := fs.Arg(0)

		if artifactID == "" {
			return fmt.Errorf("artifact ID is required")
		}

		data, err := client.get(fmt.Sprintf("/api/v1/artifacts/%s/download", artifactID))
		if err != nil {
			return err
		}

		if *outputPath == "" {
			*outputPath = artifactID
		}

		if err := os.WriteFile(*outputPath, data, 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}

		fmt.Printf("✓ Artifact downloaded to %s (%s)\n", *outputPath, formatBytes(int64(len(data))))
		return nil

	default:
		return fmt.Errorf("unknown artifacts subcommand: %s", sub)
	}
}

// configCmd handles the "config" subcommand.
func configCmd(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: flowforge config <view|set> [flags]")
		return nil
	}

	switch args[0] {
	case "view":
		fmt.Printf("Server URL: %s\n", serverURL)
		if apiToken != "" {
			fmt.Printf("API Token:  %s...%s\n", apiToken[:4], apiToken[len(apiToken)-4:])
		} else {
			fmt.Println("API Token:  (not set)")
		}
		fmt.Printf("Output:     %s\n", output)
		return nil

	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: flowforge config set <key> <value>")
		}
		key, value := args[1], args[2]

		home, _ := os.UserHomeDir()
		cfgPath := filepath.Join(home, ".flowforge", "config.yaml")

		viper.Set(key, value)
		if err := viper.WriteConfigAs(cfgPath); err != nil {
			// Try to create the file
			os.MkdirAll(filepath.Dir(cfgPath), 0755)
			if err := viper.SafeWriteConfigAs(cfgPath); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
		}

		fmt.Printf("✓ Set %s = %s\n", key, value)
		return nil

	default:
		return fmt.Errorf("unknown config subcommand: %s", args[0])
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
