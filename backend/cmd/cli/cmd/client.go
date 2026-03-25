package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/viper"
)

var (
	serverURL string
	apiToken  string
	output    string
	cfgFile   string
)

// initConfig loads configuration from file and environment.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		cfgDir := filepath.Join(home, ".flowforge")
		os.MkdirAll(cfgDir, 0755)
		viper.AddConfigPath(cfgDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("FLOWFORGE")
	viper.AutomaticEnv()
	viper.ReadInConfig()

	if serverURL == "" {
		serverURL = viper.GetString("server_url")
	}
	if serverURL == "" {
		serverURL = "http://localhost:8081"
	}

	if apiToken == "" {
		apiToken = viper.GetString("api_token")
	}
}

// apiClient provides HTTP client methods for the FlowForge API.
type apiClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func newClient() *apiClient {
	return &apiClient{
		baseURL: strings.TrimRight(serverURL, "/"),
		token:   apiToken,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *apiClient) get(path string) ([]byte, error) {
	return c.request("GET", path, nil)
}

func (c *apiClient) post(path string, body interface{}) ([]byte, error) {
	return c.request("POST", path, body)
}

func (c *apiClient) put(path string, body interface{}) ([]byte, error) {
	return c.request("PUT", path, body)
}

func (c *apiClient) del(path string) ([]byte, error) {
	return c.request("DELETE", path, nil)
}

func (c *apiClient) request(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = strings.NewReader(string(data))
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// printTable prints data as an ASCII table.
func printTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	fmt.Fprintln(w, strings.Repeat("-", len(strings.Join(headers, "  "))))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// printJSON pretty-prints JSON data.
func printJSON(data []byte) {
	var v interface{}
	if json.Unmarshal(data, &v) == nil {
		formatted, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(formatted))
	} else {
		fmt.Println(string(data))
	}
}

// Execute runs the root command.
func Execute() error {
	return rootCmd()
}
