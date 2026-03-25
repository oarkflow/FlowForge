package pipeline

import (
	"fmt"
	"sort"
	"strings"
)

// DAGNode represents a node in the pipeline DAG.
type DAGNode struct {
	Name         string   `json:"name"`
	Dependencies []string `json:"dependencies"`
	Dependents   []string `json:"dependents"`
	Level        int      `json:"level"`
}

// DAG represents a directed acyclic graph of pipeline stages.
type DAG struct {
	Nodes    map[string]*DAGNode `json:"nodes"`
	Levels   [][]string          `json:"levels"`
	HasCycle bool                `json:"has_cycle"`
}

// BuildStageDAG computes a DAG from stage names and a dependency map.
// The depsMap maps stage name → list of stage names it depends on (needs).
// If no stage has dependencies, all stages are placed at level 0 in their original order.
func BuildStageDAG(stageNames []string, depsMap map[string][]string) (*DAG, error) {
	dag := &DAG{
		Nodes: make(map[string]*DAGNode, len(stageNames)),
	}

	stageSet := make(map[string]bool, len(stageNames))
	for _, name := range stageNames {
		stageSet[name] = true
	}

	// Initialize nodes
	for _, name := range stageNames {
		dag.Nodes[name] = &DAGNode{
			Name:         name,
			Dependencies: nil,
			Dependents:   nil,
			Level:        -1,
		}
	}

	// Build adjacency
	hasDeps := false
	for _, name := range stageNames {
		deps, ok := depsMap[name]
		if !ok || len(deps) == 0 {
			continue
		}
		hasDeps = true
		dag.Nodes[name].Dependencies = deps
		for _, dep := range deps {
			if node, exists := dag.Nodes[dep]; exists {
				node.Dependents = append(node.Dependents, name)
			}
		}
	}

	// If no stage has explicit dependencies, run sequentially (backward compat):
	// each stage at its own level in order.
	if !hasDeps {
		for i, name := range stageNames {
			dag.Nodes[name].Level = i
		}
		dag.Levels = make([][]string, len(stageNames))
		for i, name := range stageNames {
			dag.Levels[i] = []string{name}
		}
		return dag, nil
	}

	// Topological sort via Kahn's algorithm to detect cycles and compute levels
	inDegree := make(map[string]int, len(stageNames))
	for _, name := range stageNames {
		inDegree[name] = 0
	}
	for _, name := range stageNames {
		for _, dep := range dag.Nodes[name].Dependencies {
			if stageSet[dep] {
				inDegree[name]++
			}
		}
	}

	// Start with nodes that have no dependencies
	var queue []string
	for _, name := range stageNames {
		if inDegree[name] == 0 {
			queue = append(queue, name)
		}
	}

	visited := 0
	levelMap := make(map[string]int, len(stageNames))

	// BFS-style level computation
	for len(queue) > 0 {
		// All items currently in queue are at the same level
		var nextQueue []string
		level := 0
		if len(queue) > 0 {
			// The level is max(dependency levels) + 1, but for root nodes it's 0
			for _, name := range queue {
				maxDepLevel := -1
				for _, dep := range dag.Nodes[name].Dependencies {
					if l, ok := levelMap[dep]; ok && l > maxDepLevel {
						maxDepLevel = l
					}
				}
				level = maxDepLevel + 1
				levelMap[name] = level
				dag.Nodes[name].Level = level
				visited++

				for _, dependent := range dag.Nodes[name].Dependents {
					inDegree[dependent]--
					if inDegree[dependent] == 0 {
						nextQueue = append(nextQueue, dependent)
					}
				}
			}
		}
		queue = nextQueue
	}

	if visited != len(stageNames) {
		dag.HasCycle = true
		var cycleStages []string
		for name, deg := range inDegree {
			if deg > 0 {
				cycleStages = append(cycleStages, name)
			}
		}
		sort.Strings(cycleStages)
		return dag, fmt.Errorf("dependency cycle detected involving stages: %s", strings.Join(cycleStages, ", "))
	}

	// Group by level
	maxLevel := 0
	for _, node := range dag.Nodes {
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}
	dag.Levels = make([][]string, maxLevel+1)
	for _, name := range stageNames {
		level := dag.Nodes[name].Level
		dag.Levels[level] = append(dag.Levels[level], name)
	}

	return dag, nil
}

// ValidateStageDAG checks stage dependency references for issues.
// Returns a list of error messages (empty if valid).
func ValidateStageDAG(stageNames []string, depsMap map[string][]string) []string {
	var errs []string
	stageSet := make(map[string]bool, len(stageNames))
	for _, s := range stageNames {
		stageSet[s] = true
	}

	for stageName, deps := range depsMap {
		for _, dep := range deps {
			if dep == stageName {
				errs = append(errs, fmt.Sprintf("stage %q has a self-reference in needs", stageName))
			}
			if !stageSet[dep] {
				errs = append(errs, fmt.Sprintf("stage %q references undefined stage %q in needs", stageName, dep))
			}
		}
	}

	// Check for cycles
	_, err := BuildStageDAG(stageNames, depsMap)
	if err != nil {
		errs = append(errs, err.Error())
	}

	return errs
}

// HasStageNeeds returns true if any entry in the deps map has dependencies.
func HasStageNeeds(depsMap map[string][]string) bool {
	for _, deps := range depsMap {
		if len(deps) > 0 {
			return true
		}
	}
	return false
}
