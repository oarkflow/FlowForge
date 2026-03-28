package detector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTempProject creates a temporary directory with the given file structure.
// Files maps relative path -> content.
func createTempProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for relPath, content := range files {
		fullPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// --- Language Detection ---

func TestDetect_GoProject(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"go.mod":      "module example.com/myapp\n\ngo 1.22\n",
		"go.sum":      "",
		"main.go":     "package main\nfunc main() {}\n",
		"handler.go":  "package main\n",
		"Makefile":    "build:\n\tgo build ./...\n",
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("should detect at least one language")
	}
	found := false
	for _, r := range results {
		if r.Language == "Go" {
			found = true
			if r.Confidence < 0.5 {
				t.Errorf("Go confidence = %f, want >= 0.5", r.Confidence)
			}
			if r.DependencyFile == "" {
				t.Error("DependencyFile should point to go.mod")
			}
			if r.RuntimeVersion != "1.22" {
				t.Errorf("RuntimeVersion = %q, want %q", r.RuntimeVersion, "1.22")
			}
		}
	}
	if !found {
		t.Error("Go should be detected")
	}
}

func TestDetect_NodeProject(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"package.json":      `{"name":"myapp","dependencies":{"express":"^4.0"}}`,
		"package-lock.json": "{}",
		"src/index.js":      "console.log('hello');",
		"src/app.ts":        "const x: number = 1;",
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if r.Language == "Node.js" {
			found = true
			if r.Confidence < 0.5 {
				t.Errorf("Node.js confidence = %f, want >= 0.5", r.Confidence)
			}
			if r.BuildTool != "npm" {
				t.Errorf("BuildTool = %q, want %q", r.BuildTool, "npm")
			}
			if !strings.Contains(r.Framework, "Express") {
				t.Errorf("Framework should contain Express: %q", r.Framework)
			}
		}
	}
	if !found {
		t.Error("Node.js should be detected")
	}
}

func TestDetect_PythonProject(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"requirements.txt": "django==4.2\ncelery\n",
		"manage.py":        "#!/usr/bin/env python\n",
		"app/views.py":     "from django.views import View\n",
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if r.Language == "Python" {
			found = true
			if r.BuildTool != "pip" {
				t.Errorf("BuildTool = %q, want %q", r.BuildTool, "pip")
			}
			if !strings.Contains(r.Framework, "Django") {
				t.Errorf("Framework should contain Django: %q", r.Framework)
			}
		}
	}
	if !found {
		t.Error("Python should be detected")
	}
}

func TestDetect_RustProject(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"Cargo.toml": "[package]\nname = \"myapp\"\n",
		"Cargo.lock": "",
		"src/main.rs": "fn main() {}\n",
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if r.Language == "Rust" {
			found = true
			if r.BuildTool != "cargo" {
				t.Errorf("BuildTool = %q, want %q", r.BuildTool, "cargo")
			}
		}
	}
	if !found {
		t.Error("Rust should be detected")
	}
}

func TestDetect_JavaMavenProject(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"pom.xml":              `<project><groupId>com.example</groupId><dependencies><dependency><groupId>org.springframework.boot</groupId></dependency></dependencies></project>`,
		"src/main/java/App.java": "public class App {}",
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if r.Language == "Java" {
			found = true
			if r.BuildTool != "maven" {
				t.Errorf("BuildTool = %q, want %q", r.BuildTool, "maven")
			}
			if !strings.Contains(r.Framework, "Spring Boot") {
				t.Errorf("Framework should contain Spring Boot: %q", r.Framework)
			}
		}
	}
	if !found {
		t.Error("Java should be detected")
	}
}

func TestDetect_PHPProject(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"composer.json": `{"require":{"laravel/framework":"^10.0"}}`,
		"composer.lock": "{}",
		"artisan":       "#!/usr/bin/env php",
		"app/Http/Controllers/HomeController.php": "<?php\n",
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if r.Language == "PHP" {
			found = true
			if !strings.Contains(r.Framework, "Laravel") {
				t.Errorf("Framework should contain Laravel: %q", r.Framework)
			}
		}
	}
	if !found {
		t.Error("PHP should be detected")
	}
}

func TestDetect_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("empty directory should have 0 detections, got %d", len(results))
	}
}

func TestDetect_SortedByConfidence(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"go.mod":       "module example.com/app\n\ngo 1.22\n",
		"main.go":      "package main\n",
		"package.json": `{"name":"test","dependencies":{"react":"^18"}}`,
		"index.js":     "console.log('hi');",
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Skip("need at least 2 detections for sort check")
	}
	for i := 1; i < len(results); i++ {
		if results[i].Confidence > results[i-1].Confidence {
			t.Errorf("results not sorted by confidence descending: [%d]=%f > [%d]=%f",
				i, results[i].Confidence, i-1, results[i-1].Confidence)
		}
	}
}

func TestDetect_SkipsHiddenDirs(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		".git/config":        "gitconfig",
		"node_modules/x.js":  "module",
		"vendor/v.go":         "package vendor",
		"src/main.go":        "package main\nfunc main() {}\n",
		"go.mod":             "module test\n\ngo 1.22\n",
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should detect Go, not be confused by vendor/node_modules
	found := false
	for _, r := range results {
		if r.Language == "Go" {
			found = true
		}
	}
	if !found {
		t.Error("Go should be detected even with skipped dirs present")
	}
}

// --- fileIndex Tests ---

func TestFileIndex_HasFile(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"go.mod": "",
		"main.go": "",
	})
	idx, err := buildFileIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !idx.hasFile("go.mod") {
		t.Error("should find go.mod")
	}
	if idx.hasFile("nonexistent") {
		t.Error("should not find nonexistent file")
	}
}

func TestFileIndex_CountExt(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"a.go": "", "b.go": "", "c.go": "",
		"x.js": "", "y.js": "",
	})
	idx, err := buildFileIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	if idx.countExt(".go") != 3 {
		t.Errorf("countExt(.go) = %d, want 3", idx.countExt(".go"))
	}
	if idx.countExt(".js") != 2 {
		t.Errorf("countExt(.js) = %d, want 2", idx.countExt(".js"))
	}
	if idx.countExt(".rs") != 0 {
		t.Errorf("countExt(.rs) = %d, want 0", idx.countExt(".rs"))
	}
}

func TestFileIndex_ReadFileByName(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"go.mod": "module test\n\ngo 1.21\n",
	})
	idx, err := buildFileIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := idx.readFileByName("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "module test") {
		t.Error("should read file content")
	}
}

func TestFileIndex_ReadFileByName_NotFound(t *testing.T) {
	dir := t.TempDir()
	idx, _ := buildFileIndex(dir)
	_, err := idx.readFileByName("nonexistent")
	if err == nil {
		t.Error("should return error for nonexistent file")
	}
}

// --- Pipeline Auto-Generation ---

func TestGenerateStarterPipeline_Go(t *testing.T) {
	yaml := GenerateStarterPipeline([]DetectionResult{
		{Language: "Go", RuntimeVersion: "1.22", Framework: "Fiber"},
	})
	if !strings.Contains(yaml, "Go CI") {
		t.Error("Go pipeline should have Go CI name")
	}
	if !strings.Contains(yaml, "golang:1.22-alpine") {
		t.Error("should use detected Go version in image")
	}
	if !strings.Contains(yaml, "Fiber") {
		t.Error("should mention detected framework")
	}
	if !strings.Contains(yaml, "go test") {
		t.Error("should include go test step")
	}
}

func TestGenerateStarterPipeline_Node(t *testing.T) {
	yaml := GenerateStarterPipeline([]DetectionResult{
		{Language: "Node.js", BuildTool: "pnpm", RuntimeVersion: "20"},
	})
	if !strings.Contains(yaml, "Node.js CI") {
		t.Error("should have Node.js CI name")
	}
	if !strings.Contains(yaml, "pnpm") {
		t.Error("should use pnpm build tool")
	}
}

func TestGenerateStarterPipeline_Python(t *testing.T) {
	yaml := GenerateStarterPipeline([]DetectionResult{
		{Language: "Python", RuntimeVersion: "3.11", BuildTool: "pip"},
	})
	if !strings.Contains(yaml, "Python CI") {
		t.Error("should have Python CI name")
	}
	if !strings.Contains(yaml, "python:3.11-slim") {
		t.Error("should use detected Python version")
	}
}

func TestGenerateStarterPipeline_Empty(t *testing.T) {
	yaml := GenerateStarterPipeline(nil)
	if !strings.Contains(yaml, "CI Pipeline") {
		t.Error("empty results should generate generic pipeline")
	}
}

func TestGenerateStarterPipeline_Unknown(t *testing.T) {
	yaml := GenerateStarterPipeline([]DetectionResult{
		{Language: "UnknownLang"},
	})
	if !strings.Contains(yaml, "CI Pipeline") {
		t.Error("unknown language should generate generic pipeline")
	}
}

// --- shouldSkipDir ---

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{".git", true},
		{"node_modules", true},
		{"vendor", true},
		{"__pycache__", true},
		{".venv", true},
		{"target", true},
		{"src", false},
		{"app", false},
		{"internal", false},
		{"cmd", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipDir(tt.name)
			if got != tt.want {
				t.Errorf("shouldSkipDir(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// --- NodeEngineVersion ---

func TestExtractNodeEngineVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"simple",
			"{\n  \"engines\": {\n    \"node\": \">=18.0.0\"\n  }\n}",
			"18.0.0",
		},
		{
			"multiline",
			`{
  "name": "app",
  "engines": {
    "node": "20.x"
  }
}`,
			"20.x",
		},
		{
			"no engines",
			`{"name":"app","version":"1.0.0"}`,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNodeEngineVersion([]byte(tt.content))
			if got != tt.want {
				t.Errorf("extractNodeEngineVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// scalaJDKImage tests live in scala_test.go
