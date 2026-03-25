package detector

import (
	"fmt"
	"strings"
)

// GenerateStarterPipeline produces a flowforge.yml pipeline configuration
// as a YAML string based on the detection results. It selects the primary
// language (highest confidence) and generates an appropriate CI pipeline.
func GenerateStarterPipeline(results []DetectionResult) string {
	if len(results) == 0 {
		return generateGenericPipeline()
	}

	primary := results[0] // already sorted by confidence descending

	switch primary.Language {
	case "Go":
		return generateGoPipeline(primary)
	case "Node.js":
		return generateNodePipeline(primary)
	case "Python":
		return generatePythonPipeline(primary)
	case "Ruby":
		return generateRubyPipeline(primary)
	case "Java":
		return generateJavaPipeline(primary)
	case "Kotlin":
		return generateKotlinPipeline(primary)
	case "Rust":
		return generateRustPipeline(primary)
	case "PHP":
		return generatePHPPipeline(primary)
	case ".NET":
		return generateDotNetPipeline(primary)
	case "Swift":
		return generateSwiftPipeline(primary)
	case "Scala":
		return generateScalaPipeline(results, primary)
	default:
		return generateGenericPipeline()
	}
}

// ---------------------------------------------------------------------------
// Language-specific pipeline generators
// ---------------------------------------------------------------------------

func generateGoPipeline(r DetectionResult) string {
	goVersion := r.RuntimeVersion
	if goVersion == "" {
		goVersion = "1.22"
	}
	image := fmt.Sprintf("golang:%s-alpine", goVersion)

	framework := ""
	if r.Framework != "" {
		framework = fmt.Sprintf("  # Detected framework: %s\n", r.Framework)
	}

	return renderYAML(pipelineTemplate{
		Name:    "Go CI",
		Comment: framework,
		Image:   image,
		Env: map[string]string{
			"CGO_ENABLED": "0",
		},
		CacheKey:   `go-mod-{{ hash "go.sum" }}`,
		CachePaths: []string{"/go/pkg/mod"},
		Stages: []stageTemplate{
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "lint",
						Steps: []stepTemplate{
							{Name: "Install golangci-lint", Run: "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"},
							{Name: "Run linter", Run: "golangci-lint run ./..."},
						},
					},
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Run tests", Run: "go test -race -coverprofile=coverage.out ./..."},
							{Name: "Upload coverage", Uses: "flowforge/upload-artifact@v1", With: map[string]string{"name": "coverage-report", "path": "coverage.out"}},
						},
					},
				},
			},
			{
				Name: "build",
				Jobs: []jobTemplate{
					{
						Name: "build",
						Steps: []stepTemplate{
							{Name: "Build binary", Run: "go build -ldflags=\"-w -s\" -o dist/app ./..."},
							{Name: "Upload artifact", Uses: "flowforge/upload-artifact@v1", With: map[string]string{"name": "app-binary", "path": "dist/app"}},
						},
					},
				},
			},
		},
	})
}

func generateNodePipeline(r DetectionResult) string {
	nodeVersion := r.RuntimeVersion
	if nodeVersion == "" {
		nodeVersion = "20"
	}
	image := fmt.Sprintf("node:%s-alpine", nodeVersion)

	installCmd := "npm ci"
	testCmd := "npm test"
	buildCmd := "npm run build"
	lintCmd := "npm run lint"

	switch r.BuildTool {
	case "pnpm":
		installCmd = "corepack enable && pnpm install --frozen-lockfile"
		testCmd = "pnpm test"
		buildCmd = "pnpm build"
		lintCmd = "pnpm lint"
	case "yarn":
		installCmd = "yarn install --frozen-lockfile"
		testCmd = "yarn test"
		buildCmd = "yarn build"
		lintCmd = "yarn lint"
	}

	cacheKey := `node-modules-{{ hash "package-lock.json" }}`
	if r.BuildTool == "pnpm" {
		cacheKey = `node-modules-{{ hash "pnpm-lock.yaml" }}`
	} else if r.BuildTool == "yarn" {
		cacheKey = `node-modules-{{ hash "yarn.lock" }}`
	}

	framework := ""
	if r.Framework != "" {
		framework = fmt.Sprintf("  # Detected framework: %s\n", r.Framework)
	}

	return renderYAML(pipelineTemplate{
		Name:       "Node.js CI",
		Comment:    framework,
		Image:      image,
		CacheKey:   cacheKey,
		CachePaths: []string{"node_modules"},
		Stages: []stageTemplate{
			{
				Name: "install",
				Jobs: []jobTemplate{
					{
						Name: "install",
						Steps: []stepTemplate{
							{Name: "Install dependencies", Run: installCmd},
						},
					},
				},
			},
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "lint",
						Steps: []stepTemplate{
							{Name: "Run linter", Run: lintCmd},
						},
					},
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Run tests", Run: testCmd},
						},
					},
				},
			},
			{
				Name: "build",
				Jobs: []jobTemplate{
					{
						Name: "build",
						Steps: []stepTemplate{
							{Name: "Build project", Run: buildCmd},
						},
					},
				},
			},
		},
	})
}

func generatePythonPipeline(r DetectionResult) string {
	pyVersion := r.RuntimeVersion
	if pyVersion == "" {
		pyVersion = "3.12"
	}
	image := fmt.Sprintf("python:%s-slim", pyVersion)

	installCmd := "pip install -r requirements.txt"
	switch r.BuildTool {
	case "poetry":
		installCmd = "pip install poetry && poetry install --no-interaction"
	case "pipenv":
		installCmd = "pip install pipenv && pipenv install --dev"
	}

	framework := ""
	if r.Framework != "" {
		framework = fmt.Sprintf("  # Detected framework: %s\n", r.Framework)
	}

	return renderYAML(pipelineTemplate{
		Name:       "Python CI",
		Comment:    framework,
		Image:      image,
		CacheKey:   `pip-cache-{{ hash "requirements.txt" }}`,
		CachePaths: []string{"/root/.cache/pip"},
		Stages: []stageTemplate{
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "lint",
						Steps: []stepTemplate{
							{Name: "Install dependencies", Run: installCmd},
							{Name: "Run linter", Run: "pip install ruff && ruff check ."},
						},
					},
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Install dependencies", Run: installCmd},
							{Name: "Run tests", Run: "pip install pytest pytest-cov && pytest --cov=. --cov-report=xml"},
							{Name: "Upload coverage", Uses: "flowforge/upload-artifact@v1", With: map[string]string{"name": "coverage-report", "path": "coverage.xml"}},
						},
					},
				},
			},
		},
	})
}

func generateRubyPipeline(r DetectionResult) string {
	framework := ""
	if r.Framework != "" {
		framework = fmt.Sprintf("  # Detected framework: %s\n", r.Framework)
	}

	testCmd := "bundle exec rake test"
	if strings.Contains(r.Framework, "Rails") {
		testCmd = "bundle exec rails test"
	}

	return renderYAML(pipelineTemplate{
		Name:       "Ruby CI",
		Comment:    framework,
		Image:      "ruby:3.3-slim",
		CacheKey:   `bundle-{{ hash "Gemfile.lock" }}`,
		CachePaths: []string{"/usr/local/bundle"},
		Stages: []stageTemplate{
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Install dependencies", Run: "bundle install --jobs 4 --retry 3"},
							{Name: "Run tests", Run: testCmd},
						},
					},
				},
			},
		},
	})
}

func generateJavaPipeline(r DetectionResult) string {
	framework := ""
	if r.Framework != "" {
		framework = fmt.Sprintf("  # Detected framework: %s\n", r.Framework)
	}

	var buildCmd, testCmd, cachePaths []string
	var cacheKey string

	switch r.BuildTool {
	case "gradle":
		buildCmd = []string{"./gradlew build -x test"}
		testCmd = []string{"./gradlew test"}
		cacheKey = `gradle-{{ hash "build.gradle" }}`
		cachePaths = []string{"/root/.gradle/caches"}
	default: // maven
		buildCmd = []string{"mvn package -DskipTests"}
		testCmd = []string{"mvn test"}
		cacheKey = `maven-{{ hash "pom.xml" }}`
		cachePaths = []string{"/root/.m2/repository"}
	}

	return renderYAML(pipelineTemplate{
		Name:       "Java CI",
		Comment:    framework,
		Image:      "eclipse-temurin:21-jdk",
		CacheKey:   cacheKey,
		CachePaths: cachePaths,
		Stages: []stageTemplate{
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Run tests", Run: testCmd[0]},
						},
					},
				},
			},
			{
				Name: "build",
				Jobs: []jobTemplate{
					{
						Name: "build",
						Steps: []stepTemplate{
							{Name: "Build project", Run: buildCmd[0]},
						},
					},
				},
			},
		},
	})
}

func generateKotlinPipeline(r DetectionResult) string {
	framework := ""
	if r.Framework != "" {
		framework = fmt.Sprintf("  # Detected framework: %s\n", r.Framework)
	}

	return renderYAML(pipelineTemplate{
		Name:       "Kotlin CI",
		Comment:    framework,
		Image:      "eclipse-temurin:21-jdk",
		CacheKey:   `gradle-{{ hash "build.gradle.kts" }}`,
		CachePaths: []string{"/root/.gradle/caches"},
		Stages: []stageTemplate{
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Run tests", Run: "./gradlew test"},
						},
					},
				},
			},
			{
				Name: "build",
				Jobs: []jobTemplate{
					{
						Name: "build",
						Steps: []stepTemplate{
							{Name: "Build project", Run: "./gradlew build -x test"},
						},
					},
				},
			},
		},
	})
}

func generateRustPipeline(r DetectionResult) string {
	return renderYAML(pipelineTemplate{
		Name:       "Rust CI",
		Image:      "rust:1.77-slim",
		CacheKey:   `cargo-{{ hash "Cargo.lock" }}`,
		CachePaths: []string{"/usr/local/cargo/registry", "target"},
		Stages: []stageTemplate{
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "lint",
						Steps: []stepTemplate{
							{Name: "Check formatting", Run: "rustfmt --check src/**/*.rs || true"},
							{Name: "Run clippy", Run: "cargo clippy -- -D warnings"},
						},
					},
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Run tests", Run: "cargo test"},
						},
					},
				},
			},
			{
				Name: "build",
				Jobs: []jobTemplate{
					{
						Name: "build",
						Steps: []stepTemplate{
							{Name: "Build release", Run: "cargo build --release"},
						},
					},
				},
			},
		},
	})
}

func generatePHPPipeline(r DetectionResult) string {
	framework := ""
	if r.Framework != "" {
		framework = fmt.Sprintf("  # Detected framework: %s\n", r.Framework)
	}

	testCmd := "vendor/bin/phpunit"
	if strings.Contains(r.Framework, "Laravel") {
		testCmd = "php artisan test"
	}

	return renderYAML(pipelineTemplate{
		Name:       "PHP CI",
		Comment:    framework,
		Image:      "php:8.3-cli",
		CacheKey:   `composer-{{ hash "composer.lock" }}`,
		CachePaths: []string{"vendor"},
		Stages: []stageTemplate{
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Install dependencies", Run: "composer install --no-interaction --prefer-dist"},
							{Name: "Run tests", Run: testCmd},
						},
					},
				},
			},
		},
	})
}

func generateDotNetPipeline(r DetectionResult) string {
	return renderYAML(pipelineTemplate{
		Name:       ".NET CI",
		Image:      "mcr.microsoft.com/dotnet/sdk:8.0",
		CacheKey:   `nuget-{{ hash "**/*.csproj" }}`,
		CachePaths: []string{"/root/.nuget/packages"},
		Stages: []stageTemplate{
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Restore packages", Run: "dotnet restore"},
							{Name: "Run tests", Run: "dotnet test --no-restore --verbosity normal"},
						},
					},
				},
			},
			{
				Name: "build",
				Jobs: []jobTemplate{
					{
						Name: "build",
						Steps: []stepTemplate{
							{Name: "Build project", Run: "dotnet build --no-restore --configuration Release"},
							{Name: "Publish", Run: "dotnet publish --no-build --configuration Release --output dist/"},
						},
					},
				},
			},
		},
	})
}

func generateSwiftPipeline(r DetectionResult) string {
	return renderYAML(pipelineTemplate{
		Name:       "Swift CI",
		Image:      "swift:5.10",
		CacheKey:   `swift-{{ hash "Package.resolved" }}`,
		CachePaths: []string{".build"},
		Stages: []stageTemplate{
			{
				Name: "test",
				Jobs: []jobTemplate{
					{
						Name: "test",
						Steps: []stepTemplate{
							{Name: "Resolve dependencies", Run: "swift package resolve"},
							{Name: "Run tests", Run: "swift test"},
						},
					},
				},
			},
			{
				Name: "build",
				Jobs: []jobTemplate{
					{
						Name: "build",
						Steps: []stepTemplate{
							{Name: "Build release", Run: "swift build -c release"},
						},
					},
				},
			},
		},
	})
}

// scalaJDKImage maps a Scala version to an appropriate JDK Docker image.
func scalaJDKImage(scalaVersion string) string {
	if scalaVersion == "" {
		return "eclipse-temurin:17-jdk"
	}
	if strings.HasPrefix(scalaVersion, "2.10.") || strings.HasPrefix(scalaVersion, "2.11.") {
		return "eclipse-temurin:8-jdk"
	}
	if strings.HasPrefix(scalaVersion, "2.12.") {
		return "eclipse-temurin:11-jdk"
	}
	if strings.HasPrefix(scalaVersion, "2.13.") {
		return "eclipse-temurin:17-jdk"
	}
	if strings.HasPrefix(scalaVersion, "3.") {
		return "eclipse-temurin:21-jdk"
	}
	return "eclipse-temurin:17-jdk"
}

func generateScalaPipeline(results []DetectionResult, r DetectionResult) string {
	jdkImage := scalaJDKImage(r.RuntimeVersion)

	framework := ""
	if r.Framework != "" {
		framework = fmt.Sprintf("  # Detected framework: %s\n", r.Framework)
	}
	if r.RuntimeVersion != "" {
		framework += fmt.Sprintf("  # Scala version: %s\n", r.RuntimeVersion)
	}

	// Determine build command based on framework.
	buildCmd := "sbt clean assembly"
	if strings.Contains(r.Framework, "Play") {
		buildCmd = "sbt dist"
	}

	stages := []stageTemplate{
		{
			Name: "test",
			Jobs: []jobTemplate{
				{
					Name:       "test",
					Image:      jdkImage,
					CacheKey:   `sbt-{{ hash "build.sbt" }}`,
					CachePaths: []string{"~/.sbt", "~/.ivy2/cache", "~/.cache/coursier", "target"},
					Steps: []stepTemplate{
						{Name: "Run tests", Run: "sbt test"},
					},
				},
			},
		},
		{
			Name: "build",
			Jobs: []jobTemplate{
				{
					Name:       "build",
					Image:      jdkImage,
					CacheKey:   `sbt-{{ hash "build.sbt" }}`,
					CachePaths: []string{"~/.sbt", "~/.ivy2/cache", "~/.cache/coursier", "target"},
					Steps: []stepTemplate{
						{Name: "Build project", Run: buildCmd},
						{Name: "Upload artifact", Uses: "flowforge/upload-artifact@v1", With: map[string]string{"name": "scala-build", "path": "target/"}},
					},
				},
			},
		},
	}

	// Check if a frontend is present in the detection results (any Node.js
	// co-detection alongside Scala triggers the frontend build stage).
	hasFrontend := false
	for _, res := range results {
		if res.Language == "Node.js" {
			hasFrontend = true
			break
		}
	}

	if hasFrontend {
		stages = append(stages, stageTemplate{
			Name: "build-frontend",
			Jobs: []jobTemplate{
				{
					Name:       "build-frontend",
					Image:      "node:20-alpine",
					CacheKey:   `node-modules-{{ hash "package-lock.json" }}`,
					CachePaths: []string{"node_modules"},
					Steps: []stepTemplate{
						{Name: "Install dependencies", Run: "cd frontend && npm ci"},
						{Name: "Build frontend", Run: "cd frontend && npm run build"},
						{Name: "Upload frontend build", Uses: "flowforge/upload-artifact@v1", With: map[string]string{"name": "frontend-build", "path": "frontend/build"}},
					},
				},
			},
		})
	}

	// Package stage.
	stages = append(stages, stageTemplate{
		Name: "package",
		Jobs: []jobTemplate{
			{
				Name:  "docker-build",
				Image: "docker:26-dind",
				Steps: []stepTemplate{
					{Name: "Build Docker image", Run: "docker build -t app:latest ."},
				},
			},
		},
	})

	// Deploy stage.
	stages = append(stages, stageTemplate{
		Name: "deploy",
		Jobs: []jobTemplate{
			{
				Name:  "deploy",
				Image: jdkImage,
				Steps: []stepTemplate{
					{Name: "Deploy", Run: "echo 'Configure deployment target in flowforge.yml'"},
				},
			},
		},
	})

	return renderYAML(pipelineTemplate{
		Name:    "Scala CI/CD",
		Comment: framework,
		Image:   jdkImage,
		Stages:  stages,
	})
}

func generateGenericPipeline() string {
	return renderYAML(pipelineTemplate{
		Name:  "CI Pipeline",
		Image: "ubuntu:22.04",
		Stages: []stageTemplate{
			{
				Name: "build",
				Jobs: []jobTemplate{
					{
						Name: "build",
						Steps: []stepTemplate{
							{Name: "Show environment", Run: "uname -a && pwd && ls -la"},
							{Name: "Echo", Run: "echo 'Configure your pipeline in flowforge.yml'"},
						},
					},
				},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// YAML rendering helpers
// ---------------------------------------------------------------------------

// pipelineTemplate is the intermediate structure used to generate YAML output.
type pipelineTemplate struct {
	Name       string
	Comment    string // optional comment lines injected near the top
	Image      string
	Env        map[string]string
	CacheKey   string
	CachePaths []string
	Stages     []stageTemplate
}

type stageTemplate struct {
	Name string
	Jobs []jobTemplate
}

type jobTemplate struct {
	Name       string
	Image      string   // per-job image override (empty = use pipeline default)
	CacheKey   string   // per-job cache key override
	CachePaths []string // per-job cache paths override
	Steps      []stepTemplate
}

type stepTemplate struct {
	Name string
	Run  string
	Uses string
	With map[string]string
}

// renderYAML generates YAML output from a pipelineTemplate using string
// building. This avoids pulling in a YAML marshal dependency and gives
// full control over formatting.
func renderYAML(t pipelineTemplate) string {
	var b strings.Builder

	b.WriteString("version: \"1\"\n")
	b.WriteString(fmt.Sprintf("name: %q\n", t.Name))
	b.WriteString("\n")

	if t.Comment != "" {
		b.WriteString(t.Comment)
	}

	// Triggers.
	b.WriteString("on:\n")
	b.WriteString("  push:\n")
	b.WriteString("    branches:\n")
	b.WriteString("      - main\n")
	b.WriteString("      - develop\n")
	b.WriteString("  pull_request:\n")
	b.WriteString("    branches:\n")
	b.WriteString("      - main\n")
	b.WriteString("\n")

	// Defaults.
	b.WriteString("defaults:\n")
	b.WriteString("  executor: docker\n")
	b.WriteString(fmt.Sprintf("  image: %s\n", t.Image))
	b.WriteString("  timeout: 30m\n")
	b.WriteString("\n")

	// Environment variables.
	if len(t.Env) > 0 {
		b.WriteString("env:\n")
		for k, v := range t.Env {
			b.WriteString(fmt.Sprintf("  %s: %q\n", k, v))
		}
		b.WriteString("\n")
	}

	// Stage list.
	b.WriteString("stages:\n")
	for _, stage := range t.Stages {
		b.WriteString(fmt.Sprintf("  - %s\n", stage.Name))
	}
	b.WriteString("\n")

	// Jobs.
	b.WriteString("jobs:\n")
	for _, stage := range t.Stages {
		for _, job := range stage.Jobs {
			b.WriteString(fmt.Sprintf("  %s:\n", job.Name))
			b.WriteString(fmt.Sprintf("    stage: %s\n", stage.Name))
			b.WriteString("    executor: docker\n")

			// Per-job image override, otherwise use pipeline default.
			jobImage := t.Image
			if job.Image != "" {
				jobImage = job.Image
			}
			b.WriteString(fmt.Sprintf("    image: %s\n", jobImage))

			// Per-job cache override, otherwise use pipeline default.
			cacheKey := t.CacheKey
			cachePaths := t.CachePaths
			if job.CacheKey != "" {
				cacheKey = job.CacheKey
			}
			if len(job.CachePaths) > 0 {
				cachePaths = job.CachePaths
			}
			if cacheKey != "" && len(cachePaths) > 0 {
				b.WriteString("    cache:\n")
				b.WriteString(fmt.Sprintf("      key: %s\n", cacheKey))
				b.WriteString("      paths:\n")
				for _, p := range cachePaths {
					b.WriteString(fmt.Sprintf("        - %s\n", p))
				}
			}

			b.WriteString("    steps:\n")
			// Always start with checkout.
			b.WriteString("      - name: Checkout\n")
			b.WriteString("        uses: flowforge/checkout@v1\n")

			for _, step := range job.Steps {
				b.WriteString(fmt.Sprintf("      - name: %s\n", step.Name))
				if step.Uses != "" {
					b.WriteString(fmt.Sprintf("        uses: %s\n", step.Uses))
					if len(step.With) > 0 {
						b.WriteString("        with:\n")
						for k, v := range step.With {
							b.WriteString(fmt.Sprintf("          %s: %s\n", k, v))
						}
					}
				} else if step.Run != "" {
					if strings.Contains(step.Run, "\n") {
						b.WriteString("        run: |\n")
						for _, line := range strings.Split(step.Run, "\n") {
							b.WriteString(fmt.Sprintf("          %s\n", line))
						}
					} else {
						b.WriteString(fmt.Sprintf("        run: %s\n", step.Run))
					}
				}
			}
		}
	}

	return b.String()
}
