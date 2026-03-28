package secrets

import (
	"fmt"
	"strings"
)

// reservedEnvVars is the set of environment variable names that must not be
// overwritten by user-defined secrets. These are critical process-level or
// OS-level variables whose modification could cause unpredictable behaviour or
// security issues.
var reservedEnvVars = map[string]bool{
	"PATH":            true,
	"HOME":            true,
	"USER":            true,
	"SHELL":           true,
	"LANG":            true,
	"TERM":            true,
	"PWD":             true,
	"HOSTNAME":        true,
	"LOGNAME":         true,
	"SHLVL":           true,
	"_":               true,
	"LD_PRELOAD":      true,
	"LD_LIBRARY_PATH": true,

	// FlowForge internal variables that pipelines must not override.
	"FLOWFORGE_RUN_ID":      true,
	"FLOWFORGE_PIPELINE_ID": true,
	"FLOWFORGE_PROJECT_ID":  true,
	"FLOWFORGE_STEP_ID":     true,
	"FLOWFORGE_AGENT_ID":    true,
}

// InjectSecrets merges the provided secrets into the environment variable map.
// Existing env entries are preserved; secrets are added only when the key does
// not collide with a reserved variable name. An error is returned if any secret
// key matches a reserved name.
func InjectSecrets(env map[string]string, secrets map[string]string) (map[string]string, error) {
	if env == nil {
		env = make(map[string]string, len(secrets))
	}

	var violations []string
	for k := range secrets {
		upper := strings.ToUpper(k)
		if reservedEnvVars[upper] {
			violations = append(violations, k)
		}
	}
	if len(violations) > 0 {
		return nil, fmt.Errorf("secrets cannot overwrite reserved environment variables: %s", strings.Join(violations, ", "))
	}

	for k, v := range secrets {
		env[k] = v
	}
	return env, nil
}
