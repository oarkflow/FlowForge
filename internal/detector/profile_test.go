package detector

import (
	"strings"
	"testing"
)

func TestInspectBuildsProjectProfileForNodeWorkspace(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"pnpm-workspace.yaml": "packages:\n  - apps/*\n",
		"apps/web/package.json": `{
			"name": "@acme/web",
			"packageManager": "pnpm@9.0.0",
			"scripts": {
				"lint": "eslint .",
				"test": "vitest run",
				"build": "vite build",
				"dev": "vite"
			},
			"dependencies": {
				"react": "^19.0.0",
				"vite": "^6.0.0"
			}
		}`,
		"apps/web/pnpm-lock.yaml": "lockfileVersion: '9.0'",
		"apps/web/vite.config.ts": "export default {}",
		"apps/web/Dockerfile":     "FROM node:20-alpine",
		"apps/web/src/index.tsx":  "console.log('hi')",
	})

	inspection, err := Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if !inspection.Profile.Monorepo {
		t.Fatalf("profile.monorepo = false, want true")
	}
	if inspection.Profile.PrimaryLanguage != "Node.js" {
		t.Fatalf("primary_language = %q, want Node.js", inspection.Profile.PrimaryLanguage)
	}
	if len(inspection.Profile.Services) == 0 {
		t.Fatal("expected at least one service")
	}

	service := inspection.Profile.Services[0]
	if service.Path != "apps/web" {
		t.Fatalf("service.path = %q, want apps/web", service.Path)
	}
	if service.PackageManager != "pnpm" {
		t.Fatalf("service.package_manager = %q, want pnpm", service.PackageManager)
	}
	if service.Framework != "React" {
		t.Fatalf("service.framework = %q, want React", service.Framework)
	}
	if service.Commands.Build != "corepack enable && pnpm build" {
		t.Fatalf("service.commands.build = %q", service.Commands.Build)
	}

	foundDocker := false
	for _, target := range inspection.Profile.DeploymentTargets {
		if target.Type == "docker" && target.Path == "apps/web/Dockerfile" {
			foundDocker = true
			break
		}
	}
	if !foundDocker {
		t.Fatal("expected docker deployment target for apps/web/Dockerfile")
	}
}

func TestInspectBuildsGoRunCommandFromCmdDirectory(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"go.mod":             "module example.com/flowforge\n\ngo 1.23\nrequire github.com/gofiber/fiber/v3 v3.0.0\n",
		"go.sum":             "",
		"cmd/server/main.go": "package main\nfunc main() {}\n",
		"internal/app.go":    "package internal\n",
	})

	inspection, err := Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if inspection.Profile.PrimaryLanguage != "Go" {
		t.Fatalf("primary_language = %q, want Go", inspection.Profile.PrimaryLanguage)
	}
	if inspection.Profile.Commands.Run != "go run ./cmd/server" {
		t.Fatalf("profile.commands.run = %q, want go run ./cmd/server", inspection.Profile.Commands.Run)
	}
	if inspection.Profile.PrimaryFramework != "Fiber" {
		t.Fatalf("primary_framework = %q, want Fiber", inspection.Profile.PrimaryFramework)
	}
}

func TestGenerateStarterPipelineForProfileSkipsDockerWithoutDockerfile(t *testing.T) {
	profile := &ProjectProfile{
		PrimaryLanguage: "Node.js",
		Services: []ServiceProfile{
			{
				Name:     "web",
				Path:     ".",
				Language: "Node.js",
				Commands: CommandHints{
					Install: "npm ci",
					Test:    "npm test",
					Build:   "npm run build",
				},
			},
		},
	}
	results := []DetectionResult{
		{
			Language:   "Node.js",
			Framework:  "React",
			BuildTool:  "npm",
			Confidence: 0.9,
		},
	}

	yaml := GenerateStarterPipelineForProfile(results, profile)
	if strings.Contains(yaml, "docker build") {
		t.Fatalf("expected generated pipeline to skip docker stage when no docker target exists:\n%s", yaml)
	}
	if strings.Contains(yaml, "deploy-local") {
		t.Fatalf("expected generated pipeline to skip deploy stage when no docker target exists:\n%s", yaml)
	}
}

func TestGenerateStarterPipelineForProfileSkipsNodeLintWhenScriptMissing(t *testing.T) {
	profile := &ProjectProfile{
		PrimaryLanguage: "Node.js",
		Services: []ServiceProfile{
			{
				Name:     "web",
				Path:     ".",
				Language: "Node.js",
				Commands: CommandHints{
					Install: "npm ci",
					Test:    "npm test",
					Build:   "npm run build",
				},
			},
		},
	}
	results := []DetectionResult{
		{
			Language:   "Node.js",
			Framework:  "React",
			BuildTool:  "npm",
			Confidence: 0.9,
		},
	}

	yaml := GenerateStarterPipelineForProfile(results, profile)
	if strings.Contains(yaml, "Run linter") || strings.Contains(yaml, "npm run lint") {
		t.Fatalf("expected generated pipeline to omit lint step when no lint command exists:\n%s", yaml)
	}
	if !strings.Contains(yaml, "npm run build") {
		t.Fatalf("expected generated pipeline to keep build step:\n%s", yaml)
	}
}

func TestInspectBuildsNodeTestCommandForCracoInCI(t *testing.T) {
	dir := createTempProject(t, map[string]string{
		"package.json": `{
			"name": "craco-app",
			"packageManager": "yarn@1.22.22",
			"scripts": {
				"test": "craco test"
			}
		}`,
		"yarn.lock": "",
	})

	inspection, err := Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if inspection.Profile.Commands.Test == "" {
		t.Fatal("expected test command to be detected")
	}
	if !strings.Contains(inspection.Profile.Commands.Test, "export CI=true") {
		t.Fatalf("expected test command to force CI mode, got %q", inspection.Profile.Commands.Test)
	}
	if !strings.Contains(inspection.Profile.Commands.Test, "yarn test -- --watch=false --passWithNoTests") {
		t.Fatalf("expected craco test command to disable watch mode, got %q", inspection.Profile.Commands.Test)
	}
}
