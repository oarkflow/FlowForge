package pipeline

import (
	"regexp"
	"sort"
	"strings"
)

// ExtractedVariable represents a single environment variable or secret
// reference found in a pipeline configuration.
type ExtractedVariable struct {
	Name     string `json:"name"`
	Type     string `json:"type"`      // "env_var" or "secret"
	Source   string `json:"source"`    // e.g., "pipeline env", "job deploy env", "step run"
	HasValue bool   `json:"has_value"` // true if a static (non-reference) value exists
}

// ExtractionResult holds the categorized variables extracted from a pipeline.
type ExtractionResult struct {
	EnvVars []ExtractedVariable `json:"env_vars"`
	Secrets []ExtractedVariable `json:"secrets"`
}

// shellBuiltins are common shell variables that should not be flagged as
// needing configuration.
var shellBuiltins = map[string]bool{
	"HOME": true, "PATH": true, "PWD": true, "USER": true, "SHELL": true,
	"TERM": true, "LANG": true, "HOSTNAME": true, "LOGNAME": true,
	"OLDPWD": true, "TMPDIR": true, "EDITOR": true, "VISUAL": true,
	"RANDOM": true, "SECONDS": true, "LINENO": true, "FUNCNAME": true,
	"BASH_SOURCE": true, "BASH_LINENO": true, "PIPESTATUS": true,
	"IFS": true, "PS1": true, "PS2": true, "UID": true, "EUID": true,
	"PPID": true, "SHLVL": true, "MACHTYPE": true, "OSTYPE": true,
	// CI-injected variables the system provides automatically
	"CI": true, "FLOWFORGE_ENV": true,
}

// Regex patterns for extracting variable references.
var (
	// ${{ secrets.NAME }} — matches the expression template syntax.
	secretExprRe = regexp.MustCompile(`\$\{\{\s*secrets\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
	// ${{ env.NAME }} — matches env expression template syntax.
	envExprRe = regexp.MustCompile(`\$\{\{\s*env\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
	// ${VAR_NAME} or ${VAR_NAME:-default} — shell-style variable deref.
	shellDerefRe = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)(?::-[^}]*)?\}`)
	// Pure shell deref value: the entire value is "${VAR_NAME}" (possibly with quotes/whitespace).
	pureShellDerefRe = regexp.MustCompile(`^\s*"\$\{([A-Z_][A-Z0-9_]*)\}"\s*$|^\s*\$\{([A-Z_][A-Z0-9_]*)\}\s*$`)
	// LOCAL_VAR=... pattern inside run scripts (locally defined variables).
	localDefRe = regexp.MustCompile(`(?m)^[[:space:]]*([A-Z_][A-Z0-9_]*)=`)
)

// ExtractVariables parses a pipeline YAML/JSON string and returns all
// environment variable and secret references it contains. It returns an empty
// result (not an error) when the input is empty or unparseable so callers can
// use it non-fatally.
func ExtractVariables(data string) *ExtractionResult {
	result := &ExtractionResult{}
	if strings.TrimSpace(data) == "" {
		return result
	}

	spec, err := Parse(data)
	if err != nil {
		return result
	}

	// Tracking maps: name → *ExtractedVariable (for dedup / promotion).
	vars := make(map[string]*ExtractedVariable)

	// Track all statically-defined env var keys (to mark has_value).
	staticEnv := make(map[string]bool)

	// --- Category A: Walk all env: blocks ---

	// Pipeline-level env.
	walkEnvMap(spec.Env, "pipeline env", vars, staticEnv)

	// Job-level env.
	for jobName, job := range spec.Jobs {
		walkEnvMap(job.Env, "job "+jobName+" env", vars, staticEnv)

		// Step-level env.
		for _, step := range job.Steps {
			src := "step"
			if step.Name != "" {
				src = "step " + step.Name
			}
			walkEnvMap(step.Env, src+" env", vars, staticEnv)
		}
	}

	// --- Category B: Scan for ${{ secrets.NAME }} and ${{ env.NAME }} ---

	// Collect all scannable strings from the spec.
	var scanStrings []struct {
		value  string
		source string
	}

	// Env map values at all levels (already partially handled above, but we
	// also need to scan the *values* for expression templates).
	for k, v := range spec.Env {
		scanStrings = append(scanStrings, struct {
			value  string
			source string
		}{v, "pipeline env " + k})
	}
	for jobName, job := range spec.Jobs {
		for k, v := range job.Env {
			scanStrings = append(scanStrings, struct {
				value  string
				source string
			}{v, "job " + jobName + " env " + k})
		}
		// Job-level string fields.
		scanStrings = append(scanStrings, struct {
			value  string
			source string
		}{job.Image, "job " + jobName + " image"})
		scanStrings = append(scanStrings, struct {
			value  string
			source string
		}{job.When, "job " + jobName + " when"})

		for _, step := range job.Steps {
			stepSrc := "step"
			if step.Name != "" {
				stepSrc = "step " + step.Name
			}
			scanStrings = append(scanStrings, struct {
				value  string
				source string
			}{step.Run, stepSrc + " run"})
			scanStrings = append(scanStrings, struct {
				value  string
				source string
			}{step.If, stepSrc + " if"})
			scanStrings = append(scanStrings, struct {
				value  string
				source string
			}{step.Uses, stepSrc + " uses"})
			for wk, wv := range step.With {
				scanStrings = append(scanStrings, struct {
					value  string
					source string
				}{wv, stepSrc + " with." + wk})
			}
			for ek, ev := range step.Env {
				scanStrings = append(scanStrings, struct {
					value  string
					source string
				}{ev, stepSrc + " env " + ek})
			}
		}
	}

	for _, s := range scanStrings {
		if s.value == "" {
			continue
		}
		// Extract ${{ secrets.NAME }}.
		for _, m := range secretExprRe.FindAllStringSubmatch(s.value, -1) {
			name := m[1]
			promoteOrAdd(vars, name, "secret", s.source, false)
		}
		// Extract ${{ env.NAME }}.
		for _, m := range envExprRe.FindAllStringSubmatch(s.value, -1) {
			name := m[1]
			addIfAbsent(vars, name, "env_var", s.source, staticEnv[name])
		}
	}

	// --- Category C: Scan step.Run for ${VAR_NAME} shell references ---

	for jobName, job := range spec.Jobs {
		for _, step := range job.Steps {
			if step.Run == "" {
				continue
			}
			stepSrc := "step"
			if step.Name != "" {
				stepSrc = "step " + step.Name
			}
			_ = jobName // used in env scan above

			// Find locally-defined variables in this run block.
			localDefs := make(map[string]bool)
			for _, m := range localDefRe.FindAllStringSubmatch(step.Run, -1) {
				localDefs[m[1]] = true
			}

			for _, m := range shellDerefRe.FindAllStringSubmatch(step.Run, -1) {
				name := m[1]
				if shellBuiltins[name] || localDefs[name] {
					continue
				}
				// If already known as secret, don't downgrade.
				if existing, ok := vars[name]; ok && existing.Type == "secret" {
					continue
				}
				addIfAbsent(vars, name, "env_var", stepSrc+" run", staticEnv[name])
			}
		}
	}

	// --- Build result ---
	for _, v := range vars {
		switch v.Type {
		case "secret":
			result.Secrets = append(result.Secrets, *v)
		default:
			result.EnvVars = append(result.EnvVars, *v)
		}
	}

	sort.Slice(result.EnvVars, func(i, j int) bool { return result.EnvVars[i].Name < result.EnvVars[j].Name })
	sort.Slice(result.Secrets, func(i, j int) bool { return result.Secrets[i].Name < result.Secrets[j].Name })

	return result
}

// walkEnvMap processes an env: map, classifying keys as static or needing
// configuration based on their values.
func walkEnvMap(envMap map[string]string, source string, vars map[string]*ExtractedVariable, staticEnv map[string]bool) {
	for key, value := range envMap {
		// Check if the value is a pure shell deref like "${VAR_NAME}".
		if m := pureShellDerefRe.FindStringSubmatch(value); m != nil {
			innerVar := m[1]
			if innerVar == "" {
				innerVar = m[2]
			}
			// The env key is defined, but its value comes from another variable.
			addIfAbsent(vars, innerVar, "env_var", source, false)
			staticEnv[key] = true // the key itself is "defined" in the pipeline
			continue
		}

		// Static value — mark as having a value.
		staticEnv[key] = true
		addIfAbsent(vars, key, "env_var", source, true)
	}
}

// addIfAbsent adds a variable only if it doesn't already exist in the map.
func addIfAbsent(vars map[string]*ExtractedVariable, name, typ, source string, hasValue bool) {
	if _, ok := vars[name]; !ok {
		vars[name] = &ExtractedVariable{
			Name:     name,
			Type:     typ,
			Source:   source,
			HasValue: hasValue,
		}
	}
}

// promoteOrAdd adds a variable, or promotes it to "secret" type if it
// already exists as "env_var".
func promoteOrAdd(vars map[string]*ExtractedVariable, name, typ, source string, hasValue bool) {
	if existing, ok := vars[name]; ok {
		// Promote env_var → secret.
		if typ == "secret" && existing.Type != "secret" {
			existing.Type = "secret"
			existing.Source = source
		}
		return
	}
	vars[name] = &ExtractedVariable{
		Name:     name,
		Type:     typ,
		Source:   source,
		HasValue: hasValue,
	}
}
