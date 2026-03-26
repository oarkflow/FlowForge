package detector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupScalaProject creates a temporary directory with the given files
// and returns the path. Caller should defer os.RemoveAll(path).
func setupScalaProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func findResult(results []DetectionResult, lang string) *DetectionResult {
	for i := range results {
		if results[i].Language == lang {
			return &results[i]
		}
	}
	return nil
}

func TestScalaDetector_BasicDetection(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `
name := "my-app"
scalaVersion := "2.11.7"
libraryDependencies ++= Seq(
  "org.scalatest" %% "scalatest" % "3.2.15" % Test
)
`,
		"src/main/scala/Main.scala": `object Main extends App { println("Hello") }`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r == nil {
		t.Fatal("expected Scala to be detected")
	}
	if r.Confidence < 0.5 {
		t.Errorf("expected confidence >= 0.5, got %f", r.Confidence)
	}
	if r.RuntimeVersion != "2.11.7" {
		t.Errorf("expected version 2.11.7, got %q", r.RuntimeVersion)
	}
	if r.BuildTool != "sbt" {
		t.Errorf("expected build tool sbt, got %q", r.BuildTool)
	}
}

func TestScalaDetector_ThisBuildVersionStyle(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `
ThisBuild / scalaVersion := "3.3.1"
name := "my-app"
`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r == nil {
		t.Fatal("expected Scala to be detected")
	}
	if r.RuntimeVersion != "3.3.1" {
		t.Errorf("expected version 3.3.1, got %q", r.RuntimeVersion)
	}
}

func TestScalaDetector_InThisBuildVersionStyle(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `
scalaVersion in ThisBuild := "2.13.12"
name := "my-app"
`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r == nil {
		t.Fatal("expected Scala to be detected")
	}
	if r.RuntimeVersion != "2.13.12" {
		t.Errorf("expected version 2.13.12, got %q", r.RuntimeVersion)
	}
}

func TestScalaDetector_ConfidenceScaling(t *testing.T) {
	files := map[string]string{
		"build.sbt": `scalaVersion := "2.12.18"`,
	}
	// Add multiple .scala files to boost confidence.
	for i := 0; i < 25; i++ {
		name := filepath.Join("src", "main", "scala", strings.Replace(
			"File%d.scala", "%d", string(rune('A'+i)), 1))
		files[name] = `object Dummy`
	}
	dir := setupScalaProject(t, files)

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r == nil {
		t.Fatal("expected Scala to be detected")
	}
	// build.sbt = 0.5, 25 .scala files * 0.02 = 0.5, capped at 0.4 → 0.9
	if r.Confidence < 0.9 {
		t.Errorf("expected confidence >= 0.9, got %f", r.Confidence)
	}
}

func TestScalaFrameworkDetector_PlayFramework(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `
scalaVersion := "2.13.12"
libraryDependencies ++= Seq(
  "com.typesafe.play" %% "play-scala" % "2.8.19"
)
`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r == nil {
		t.Fatal("expected Scala to be detected")
	}
	if !strings.Contains(r.Framework, "Play Framework") {
		t.Errorf("expected Play Framework, got %q", r.Framework)
	}
}

func TestScalaFrameworkDetector_Akka(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `
scalaVersion := "2.13.12"
libraryDependencies ++= Seq(
  "com.typesafe.akka" %% "akka-actor" % "2.8.5"
)
`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r == nil {
		t.Fatal("expected Scala to be detected")
	}
	if !strings.Contains(r.Framework, "Akka") {
		t.Errorf("expected Akka, got %q", r.Framework)
	}
}

func TestScalaFrameworkDetector_ZIO(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `
scalaVersion := "3.3.1"
libraryDependencies ++= Seq(
  "dev.zio" %% "zio" % "2.0.19"
)
`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r == nil {
		t.Fatal("expected Scala to be detected")
	}
	if !strings.Contains(r.Framework, "ZIO") {
		t.Errorf("expected ZIO, got %q", r.Framework)
	}
}

func TestScalaFrameworkDetector_Http4s(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `
scalaVersion := "2.13.12"
libraryDependencies ++= Seq(
  "org.http4s" %% "http4s-dsl" % "0.23.24"
)
`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r == nil {
		t.Fatal("expected Scala to be detected")
	}
	if !strings.Contains(r.Framework, "http4s") {
		t.Errorf("expected http4s, got %q", r.Framework)
	}
}

func TestScalaFrameworkDetector_Finagle(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `
scalaVersion := "2.13.12"
libraryDependencies ++= Seq(
  "com.twitter" %% "finagle-http" % "23.11.0"
)
`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r == nil {
		t.Fatal("expected Scala to be detected")
	}
	if !strings.Contains(r.Framework, "Finagle") {
		t.Errorf("expected Finagle, got %q", r.Framework)
	}
}

func TestScalaJDKImage(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"2.10.7", "eclipse-temurin:8-jdk"},
		{"2.11.7", "eclipse-temurin:8-jdk"},
		{"2.11.12", "eclipse-temurin:8-jdk"},
		{"2.12.18", "eclipse-temurin:11-jdk"},
		{"2.12.0", "eclipse-temurin:11-jdk"},
		{"2.13.12", "eclipse-temurin:17-jdk"},
		{"2.13.0", "eclipse-temurin:17-jdk"},
		{"3.3.1", "eclipse-temurin:21-jdk"},
		{"3.0.0", "eclipse-temurin:21-jdk"},
		{"", "eclipse-temurin:17-jdk"},
		{"unknown", "eclipse-temurin:17-jdk"},
	}

	for _, tc := range tests {
		t.Run("version_"+tc.version, func(t *testing.T) {
			got := scalaJDKImage(tc.version)
			if got != tc.expected {
				t.Errorf("scalaJDKImage(%q) = %q, want %q", tc.version, got, tc.expected)
			}
		})
	}
}

func TestGenerateScalaPipeline_Basic(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `scalaVersion := "2.11.7"`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	yaml := GenerateStarterPipeline(results)
	if yaml == "" {
		t.Fatal("expected non-empty pipeline YAML")
	}

	// Verify key elements are present.
	checks := []string{
		"Scala CI/CD",
		"sbtscala/scala-sbt:",
		"sbt test",
		"sbt clean assembly",
		"Scala version: 2.11.7",
		"deploy-local",
		"Application available at",
	}
	for _, check := range checks {
		if !strings.Contains(yaml, check) {
			t.Errorf("expected YAML to contain %q, got:\n%s", check, yaml)
		}
	}
}

func TestGenerateScalaPipeline_WithSolidJSFrontend(t *testing.T) {
	files := map[string]string{
		"build.sbt": `scalaVersion := "2.11.7"`,
		"frontend/package.json": `{
  "name": "my-frontend",
  "dependencies": {
    "solid-js": "^1.9.0"
  },
  "devDependencies": {
    "vite": "^6.0.0",
    "vite-plugin-solid": "^2.10.0"
  },
  "scripts": {
    "build": "vite build"
  }
}`,
	}
	// Add enough .scala files so Scala has higher confidence than Node.js.
	for i := 0; i < 20; i++ {
		name := filepath.Join("src", "main", "scala", "File"+string(rune('A'+i))+".scala")
		files[name] = `object Dummy`
	}
	dir := setupScalaProject(t, files)

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify Scala is the primary (highest confidence) result.
	if results[0].Language != "Scala" {
		t.Fatalf("expected Scala as primary language, got %q (confidence: %f)",
			results[0].Language, results[0].Confidence)
	}

	yaml := GenerateStarterPipeline(results)
	if yaml == "" {
		t.Fatal("expected non-empty pipeline YAML")
	}

	// When a frontend framework (SolidJS, React, etc.) is detected alongside
	// Scala, the pipeline should include a frontend build stage.
	checks := []string{
		"Scala CI/CD",
		"sbtscala/scala-sbt:",
		"sbt test",
		"build-frontend",
		"node:20-alpine",
		"npm install",
		"npm run build",
		"deploy-local",
		"Deploy to Docker",
		"Application available at",
	}
	for _, check := range checks {
		if !strings.Contains(yaml, check) {
			t.Errorf("expected YAML to contain %q, got:\n%s", check, yaml)
		}
	}
}

func TestGenerateScalaPipeline_PlayFramework(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"build.sbt": `
scalaVersion := "2.13.12"
libraryDependencies ++= Seq(
  "com.typesafe.play" %% "play-scala" % "2.8.19"
)
`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	yaml := GenerateStarterPipeline(results)

	// Play projects should use "sbt dist" instead of "sbt clean assembly".
	if !strings.Contains(yaml, "sbt dist") {
		t.Errorf("expected Play project to use 'sbt dist', got:\n%s", yaml)
	}
	if strings.Contains(yaml, "sbt clean assembly") {
		t.Errorf("expected Play project NOT to use 'sbt clean assembly', got:\n%s", yaml)
	}
}

func TestScalaDetector_NoScalaProject(t *testing.T) {
	dir := setupScalaProject(t, map[string]string{
		"package.json": `{"name": "not-scala"}`,
		"index.js":     `console.log("hello")`,
	})

	results, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := findResult(results, "Scala")
	if r != nil {
		t.Error("expected no Scala detection for a JS project")
	}
}
