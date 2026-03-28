package secrets

import (
	"strings"
	"testing"
)

func TestInjectSecrets_BasicInjection(t *testing.T) {
	env := map[string]string{"EXISTING": "value"}
	secrets := map[string]string{"API_KEY": "secret123", "DB_PASS": "s3cret"}

	result, err := InjectSecrets(env, secrets)
	if err != nil {
		t.Fatal(err)
	}
	if result["EXISTING"] != "value" {
		t.Error("existing env var should be preserved")
	}
	if result["API_KEY"] != "secret123" {
		t.Errorf("API_KEY = %q, want %q", result["API_KEY"], "secret123")
	}
	if result["DB_PASS"] != "s3cret" {
		t.Errorf("DB_PASS = %q, want %q", result["DB_PASS"], "s3cret")
	}
}

func TestInjectSecrets_NilEnv(t *testing.T) {
	secrets := map[string]string{"KEY": "val"}
	result, err := InjectSecrets(nil, secrets)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result["KEY"] != "val" {
		t.Errorf("KEY = %q, want %q", result["KEY"], "val")
	}
}

func TestInjectSecrets_EmptySecrets(t *testing.T) {
	env := map[string]string{"A": "1"}
	result, err := InjectSecrets(env, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if result["A"] != "1" {
		t.Error("existing env should be unchanged")
	}
}

func TestInjectSecrets_ReservedEnvVars(t *testing.T) {
	reserved := []string{
		"PATH", "HOME", "USER", "SHELL", "LANG", "TERM", "PWD",
		"HOSTNAME", "LOGNAME", "SHLVL", "_", "LD_PRELOAD", "LD_LIBRARY_PATH",
		"FLOWFORGE_RUN_ID", "FLOWFORGE_PIPELINE_ID", "FLOWFORGE_PROJECT_ID",
		"FLOWFORGE_STEP_ID", "FLOWFORGE_AGENT_ID",
	}
	for _, name := range reserved {
		t.Run(name, func(t *testing.T) {
			_, err := InjectSecrets(nil, map[string]string{name: "hacked"})
			if err == nil {
				t.Errorf("should reject reserved variable %s", name)
			}
			if !strings.Contains(err.Error(), "reserved") {
				t.Errorf("error should mention reserved: %v", err)
			}
		})
	}
}

func TestInjectSecrets_ReservedCaseInsensitive(t *testing.T) {
	// Even lowercase "path" should be rejected because reservedEnvVars checks upper
	_, err := InjectSecrets(nil, map[string]string{"path": "hacked"})
	if err == nil {
		t.Error("should reject lowercase reserved variable")
	}
}

func TestInjectSecrets_MultipleReservedReported(t *testing.T) {
	_, err := InjectSecrets(nil, map[string]string{"PATH": "a", "HOME": "b"})
	if err == nil {
		t.Fatal("should reject multiple reserved variables")
	}
	// Both should be listed in the error
	if !strings.Contains(err.Error(), "PATH") || !strings.Contains(err.Error(), "HOME") {
		t.Errorf("error should list both: %v", err)
	}
}

func TestInjectSecrets_OverwritesExistingNonReserved(t *testing.T) {
	env := map[string]string{"MY_VAR": "old"}
	result, err := InjectSecrets(env, map[string]string{"MY_VAR": "new"})
	if err != nil {
		t.Fatal(err)
	}
	if result["MY_VAR"] != "new" {
		t.Errorf("MY_VAR = %q, want %q", result["MY_VAR"], "new")
	}
}
