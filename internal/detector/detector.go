// Package detector provides automatic language, framework, and dependency
// detection for project repositories. It walks a directory tree, identifies
// the technology stack, and can auto-generate a starter pipeline config.
package detector

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DetectionResult holds the outcome of detecting a single language/framework
// combination in a repository.
type DetectionResult struct {
	Language       string  `json:"language"`
	Framework      string  `json:"framework,omitempty"`
	DependencyFile string  `json:"dependency_file,omitempty"`
	Confidence     float64 `json:"confidence"`
	BuildTool      string  `json:"build_tool,omitempty"`
	RuntimeVersion string  `json:"runtime_version,omitempty"`
}

// fileIndex is a pre-built index of files found in the directory tree,
// used to avoid repeated filesystem walks.
type fileIndex struct {
	rootDir string

	// allFiles is every relative file path encountered.
	allFiles []string

	// byName maps a base filename (e.g. "go.mod") to its relative paths.
	byName map[string][]string

	// byExt maps a file extension (e.g. ".go") to its relative paths.
	byExt map[string][]string

	// fileContents caches file contents that have already been read.
	fileContents map[string][]byte
}

// buildFileIndex walks rootDir and builds the index.
func buildFileIndex(rootDir string) (*fileIndex, error) {
	idx := &fileIndex{
		rootDir:      rootDir,
		byName:       make(map[string][]string),
		byExt:        make(map[string][]string),
		fileContents: make(map[string][]byte),
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}

		// Skip hidden directories and common non-source directories.
		name := info.Name()
		if info.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}

		idx.allFiles = append(idx.allFiles, rel)
		idx.byName[name] = append(idx.byName[name], rel)

		ext := filepath.Ext(name)
		if ext != "" {
			idx.byExt[ext] = append(idx.byExt[ext], rel)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return idx, nil
}

// hasFile returns true if a file with the given base name exists.
func (idx *fileIndex) hasFile(name string) bool {
	return len(idx.byName[name]) > 0
}

// hasExt returns true if any file with the given extension exists.
func (idx *fileIndex) hasExt(ext string) bool {
	return len(idx.byExt[ext]) > 0
}

// countExt returns how many files have the given extension.
func (idx *fileIndex) countExt(ext string) int {
	return len(idx.byExt[ext])
}

// readFile reads and caches the contents of the file at the given relative path.
func (idx *fileIndex) readFile(relPath string) ([]byte, error) {
	if data, ok := idx.fileContents[relPath]; ok {
		return data, nil
	}
	absPath := filepath.Join(idx.rootDir, relPath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	idx.fileContents[relPath] = data
	return data, nil
}

// readFileByName reads the first file with the given base name.
func (idx *fileIndex) readFileByName(name string) ([]byte, error) {
	paths := idx.byName[name]
	if len(paths) == 0 {
		return nil, os.ErrNotExist
	}
	return idx.readFile(paths[0])
}

// firstPathForName returns the first relative path for a file with the given name.
func (idx *fileIndex) firstPathForName(name string) string {
	paths := idx.byName[name]
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

// shouldSkipDir returns true for directories that should not be traversed.
func shouldSkipDir(name string) bool {
	skip := map[string]bool{
		".git":         true,
		".hg":          true,
		".svn":         true,
		"node_modules": true,
		"vendor":       true,
		".vendor":      true,
		"__pycache__":  true,
		".venv":        true,
		"venv":         true,
		".tox":         true,
		"dist":         true,
		"build":        true,
		"target":       true,
		".gradle":      true,
		".idea":        true,
		".vscode":      true,
		".cache":       true,
		"bin":          true,
		"obj":          true,
	}
	return strings.HasPrefix(name, ".") && len(name) > 1 && !skip[name] || skip[name]
}

// Detect walks the directory tree at rootDir and returns all detected
// languages, frameworks, and dependency information sorted by confidence
// (highest first).
func Detect(rootDir string) ([]DetectionResult, error) {
	inspection, err := Inspect(rootDir)
	if err != nil {
		return nil, err
	}
	return inspection.Detections, nil
}

func detectWithIndex(idx *fileIndex) []DetectionResult {
	var results []DetectionResult

	// Run all language detectors.
	langResults := detectLanguages(idx)
	results = append(results, langResults...)

	// Run infrastructure detectors.
	infraResults := detectInfrastructure(idx)
	results = append(results, infraResults...)

	// Enrich with framework detection.
	results = detectFrameworks(idx, results)

	// Sort by confidence descending.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	return results
}

// DetectLanguage is the main entry point — an alias for Detect that
// returns the detection results for a given directory.
func DetectLanguage(rootDir string) ([]DetectionResult, error) {
	return Detect(rootDir)
}
