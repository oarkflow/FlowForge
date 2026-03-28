package pipeline

import (
	"fmt"
	"sort"
)

// MatrixCombination represents a single combination of matrix values.
type MatrixCombination map[string]string

// ExpandMatrix takes a JobSpec with a MatrixConfig and returns all expanded
// job copies, one per Cartesian-product combination. Each returned job has
// its Matrix set to nil and its name suffixed with the combination values.
// The returned map keys use the format "originalName (k1=v1, k2=v2)".
func ExpandMatrix(jobName string, job JobSpec) map[string]JobSpec {
	if job.Matrix == nil || len(job.Matrix.Entries) == 0 {
		return map[string]JobSpec{jobName: job}
	}

	combos := cartesianProduct(job.Matrix.Entries)
	result := make(map[string]JobSpec, len(combos))

	for _, combo := range combos {
		expanded := copyJob(job)
		expanded.Matrix = nil

		// Merge matrix values into the job's environment so that
		// ${{ matrix.* }} expressions can reference them.
		if expanded.Env == nil {
			expanded.Env = make(map[string]string)
		}
		for k, v := range combo {
			expanded.Env["MATRIX_"+toEnvKey(k)] = v
		}

		// Build a human-readable suffix from sorted keys for determinism.
		suffix := comboSuffix(combo)
		name := fmt.Sprintf("%s (%s)", jobName, suffix)
		result[name] = expanded
	}

	return result
}

// ExpandAllMatrices expands all matrix jobs in the spec, returning a new Jobs
// map with matrix jobs replaced by their expanded variants.
func ExpandAllMatrices(spec *PipelineSpec) map[string]JobSpec {
	result := make(map[string]JobSpec)
	for name, job := range spec.Jobs {
		for expandedName, expandedJob := range ExpandMatrix(name, job) {
			result[expandedName] = expandedJob
		}
	}
	return result
}

// cartesianProduct computes the Cartesian product of a map of string slices.
// Keys are iterated in sorted order to produce deterministic results.
func cartesianProduct(entries map[string][]string) []MatrixCombination {
	keys := sortedKeys(entries)
	if len(keys) == 0 {
		return nil
	}

	// Start with a single empty combination.
	combos := []MatrixCombination{{}}

	for _, key := range keys {
		values := entries[key]
		var next []MatrixCombination
		for _, combo := range combos {
			for _, val := range values {
				c := make(MatrixCombination, len(combo)+1)
				for k, v := range combo {
					c[k] = v
				}
				c[key] = val
				next = append(next, c)
			}
		}
		combos = next
	}

	return combos
}

// copyJob creates a shallow copy of a JobSpec with copied slices and maps.
func copyJob(job JobSpec) JobSpec {
	cp := job

	if job.Needs != nil {
		cp.Needs = make([]string, len(job.Needs))
		copy(cp.Needs, job.Needs)
	}

	if job.Env != nil {
		cp.Env = make(map[string]string, len(job.Env))
		for k, v := range job.Env {
			cp.Env[k] = v
		}
	}

	if job.Steps != nil {
		cp.Steps = make([]StepSpec, len(job.Steps))
		copy(cp.Steps, job.Steps)
	}

	if job.Cache != nil {
		cp.Cache = make([]CacheConfig, len(job.Cache))
		copy(cp.Cache, job.Cache)
	}

	return cp
}

// comboSuffix builds a deterministic "k1=v1, k2=v2" string from a combination.
func comboSuffix(combo MatrixCombination) string {
	keys := make([]string, 0, len(combo))
	for k := range combo {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := ""
	for i, k := range keys {
		if i > 0 {
			result += ", "
		}
		result += k + "=" + combo[k]
	}
	return result
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// toEnvKey converts a matrix key to an environment variable suffix.
// For example "go_version" stays as "GO_VERSION".
func toEnvKey(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			result = append(result, c-32) // uppercase
		} else if c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
