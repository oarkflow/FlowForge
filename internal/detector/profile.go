package detector

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Inspection combines the flat detection list with a richer project profile.
type Inspection struct {
	Detections []DetectionResult `json:"detections"`
	Profile    ProjectProfile    `json:"profile"`
}

// ProjectProfile captures the import-time understanding of a repository.
type ProjectProfile struct {
	HasFlowForgeConfig  bool               `json:"has_flowforge_config"`
	FlowForgeConfigPath string             `json:"flowforge_config_path,omitempty"`
	Monorepo            bool               `json:"monorepo"`
	PrimaryLanguage     string             `json:"primary_language,omitempty"`
	PrimaryFramework    string             `json:"primary_framework,omitempty"`
	PackageManagers     []string           `json:"package_managers,omitempty"`
	DependencyFiles     []string           `json:"dependency_files,omitempty"`
	ConfigFiles         []string           `json:"config_files,omitempty"`
	EnvFiles            []string           `json:"env_files,omitempty"`
	Commands            CommandHints       `json:"commands,omitempty"`
	Services            []ServiceProfile   `json:"services,omitempty"`
	DeploymentTargets   []DeploymentTarget `json:"deployment_targets,omitempty"`
}

type CommandHints struct {
	Install string `json:"install,omitempty"`
	Lint    string `json:"lint,omitempty"`
	Test    string `json:"test,omitempty"`
	Build   string `json:"build,omitempty"`
	Run     string `json:"run,omitempty"`
}

type ServiceProfile struct {
	Name            string       `json:"name"`
	Path            string       `json:"path"`
	Language        string       `json:"language,omitempty"`
	Framework       string       `json:"framework,omitempty"`
	PackageManager  string       `json:"package_manager,omitempty"`
	BuildTool       string       `json:"build_tool,omitempty"`
	RuntimeVersion  string       `json:"runtime_version,omitempty"`
	DependencyFiles []string     `json:"dependency_files,omitempty"`
	Dependencies    []string     `json:"dependencies,omitempty"`
	ConfigFiles     []string     `json:"config_files,omitempty"`
	Commands        CommandHints `json:"commands,omitempty"`
}

type DeploymentTarget struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Tool string `json:"tool,omitempty"`
}

type packageManifest struct {
	Name            string            `json:"name"`
	PackageManager  string            `json:"packageManager"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Engines         map[string]string `json:"engines"`
	Workspaces      any               `json:"workspaces"`
}

// Inspect returns both the flat detection results and a richer project profile.
func Inspect(rootDir string) (*Inspection, error) {
	idx, err := buildFileIndex(rootDir)
	if err != nil {
		return nil, err
	}

	detections := detectWithIndex(idx)
	profile := analyzeProject(idx, detections)

	return &Inspection{
		Detections: detections,
		Profile:    profile,
	}, nil
}

func analyzeProject(idx *fileIndex, detections []DetectionResult) ProjectProfile {
	profile := ProjectProfile{}
	profile.FlowForgeConfigPath = findFirstPath(idx, "flowforge.yml", ".flowforge.yml", "flowforge.yaml", ".flowforge.yaml")
	profile.HasFlowForgeConfig = profile.FlowForgeConfigPath != ""
	profile.EnvFiles = collectEnvFiles(idx)
	profile.ConfigFiles = collectConfigFiles(idx)
	profile.DeploymentTargets = collectDeploymentTargets(idx)

	services := collectServices(idx)
	sort.Slice(services, func(i, j int) bool {
		if services[i].Path == services[j].Path {
			return services[i].Name < services[j].Name
		}
		if services[i].Path == "." {
			return true
		}
		if services[j].Path == "." {
			return false
		}
		return services[i].Path < services[j].Path
	})
	profile.Services = services

	for _, svc := range services {
		if svc.PackageManager != "" {
			profile.PackageManagers = appendUnique(profile.PackageManagers, svc.PackageManager)
		}
		profile.DependencyFiles = appendUnique(profile.DependencyFiles, svc.DependencyFiles...)
	}
	sort.Strings(profile.PackageManagers)
	sort.Strings(profile.DependencyFiles)

	profile.Monorepo = len(services) > 1 || idx.hasFile("go.work") || idx.hasFile("pnpm-workspace.yaml") || idx.hasFile("turbo.json") || idx.hasFile("nx.json") || hasNodeWorkspace(idx)

	if len(detections) > 0 {
		profile.PrimaryLanguage = detections[0].Language
		profile.PrimaryFramework = detections[0].Framework
	}
	if primary := choosePrimaryService(services, detections); primary != nil {
		profile.Commands = primary.Commands
		if profile.PrimaryLanguage == "" {
			profile.PrimaryLanguage = primary.Language
		}
		if profile.PrimaryFramework == "" {
			profile.PrimaryFramework = primary.Framework
		}
	} else {
		profile.Commands = fallbackCommands(detections)
	}

	sort.Strings(profile.ConfigFiles)
	sort.Strings(profile.EnvFiles)
	sort.Slice(profile.DeploymentTargets, func(i, j int) bool {
		if profile.DeploymentTargets[i].Type == profile.DeploymentTargets[j].Type {
			return profile.DeploymentTargets[i].Path < profile.DeploymentTargets[j].Path
		}
		return profile.DeploymentTargets[i].Type < profile.DeploymentTargets[j].Type
	})

	return profile
}

func collectServices(idx *fileIndex) []ServiceProfile {
	services := map[string]*ServiceProfile{}

	for _, manifest := range idx.byName["package.json"] {
		enrichNodeService(idx, services, manifest)
	}
	for _, manifest := range idx.byName["go.mod"] {
		enrichGoService(idx, services, manifest)
	}
	for _, manifest := range idx.byName["pyproject.toml"] {
		enrichPythonService(idx, services, manifest)
	}
	for _, manifest := range idx.byName["requirements.txt"] {
		enrichPythonService(idx, services, manifest)
	}
	for _, manifest := range idx.byName["Pipfile"] {
		enrichPythonService(idx, services, manifest)
	}
	for _, manifest := range idx.byName["pom.xml"] {
		enrichJavaService(idx, services, manifest)
	}
	for _, manifest := range idx.byName["build.gradle"] {
		enrichJavaService(idx, services, manifest)
	}
	for _, manifest := range idx.byName["build.gradle.kts"] {
		enrichJavaService(idx, services, manifest)
	}
	for _, manifest := range idx.byName["Cargo.toml"] {
		enrichRustService(idx, services, manifest)
	}
	for _, manifest := range idx.byName["composer.json"] {
		enrichPHPService(idx, services, manifest)
	}
	for _, manifest := range idx.byExt[".csproj"] {
		enrichDotNetService(idx, services, manifest)
	}

	var result []ServiceProfile
	for _, svc := range services {
		sort.Strings(svc.DependencyFiles)
		sort.Strings(svc.Dependencies)
		sort.Strings(svc.ConfigFiles)
		result = append(result, *svc)
	}
	return result
}

func enrichNodeService(idx *fileIndex, services map[string]*ServiceProfile, manifestPath string) {
	dir := pathDir(manifestPath)
	svc := ensureService(services, dir)

	data, err := idx.readFile(manifestPath)
	if err != nil {
		return
	}
	var manifest packageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return
	}

	if manifest.Name != "" {
		svc.Name = manifest.Name
	}
	svc.Path = dir
	svc.Language = "Node.js"
	svc.RuntimeVersion = normalizeSemver(manifest.Engines["node"])
	svc.PackageManager = detectNodePackageManager(idx, dir, manifest.PackageManager)
	svc.BuildTool = svc.PackageManager
	svc.Framework = detectNodeFrameworkFromMaps(manifest.Dependencies, manifest.DevDependencies)
	svc.DependencyFiles = appendUnique(svc.DependencyFiles, append([]string{manifestPath}, filesByNameInDir(idx, dir, "package-lock.json", "pnpm-lock.yaml", "yarn.lock")...)...)
	svc.Dependencies = appendUnique(svc.Dependencies, mapKeys(manifest.Dependencies)...)
	svc.ConfigFiles = appendUnique(svc.ConfigFiles, filesByNameInDir(idx, dir, "tsconfig.json", "vite.config.ts", "vite.config.js", "next.config.js", "next.config.mjs", "astro.config.mjs", "astro.config.ts", "nuxt.config.ts", "nuxt.config.js")...)
	svc.Commands = buildNodeCommands(svc.PackageManager, manifest.Scripts)
}

func enrichGoService(idx *fileIndex, services map[string]*ServiceProfile, manifestPath string) {
	dir := pathDir(manifestPath)
	svc := ensureService(services, dir)
	content, err := idx.readFile(manifestPath)
	if err != nil {
		return
	}

	svc.Path = dir
	if svc.Name == "" {
		svc.Name = filepath.Base(dir)
		if svc.Name == "." || svc.Name == string(filepath.Separator) {
			svc.Name = "root"
		}
	}
	svc.Language = "Go"
	svc.PackageManager = "go"
	svc.BuildTool = "go"
	svc.RuntimeVersion = extractGoVersionFromData(content)
	svc.Framework = detectGoFrameworkFromContent(string(content))
	svc.DependencyFiles = appendUnique(svc.DependencyFiles, manifestPath)
	if sumPath := firstFileByNameInDir(idx, dir, "go.sum"); sumPath != "" {
		svc.DependencyFiles = appendUnique(svc.DependencyFiles, sumPath)
	}
	svc.Dependencies = appendUnique(svc.Dependencies, parseGoDependencies(string(content))...)
	svc.ConfigFiles = appendUnique(svc.ConfigFiles, filesByNameInDir(idx, dir, "Makefile", "Dockerfile")...)
	svc.Commands = buildGoCommands(idx, dir)
}

func enrichPythonService(idx *fileIndex, services map[string]*ServiceProfile, manifestPath string) {
	dir := pathDir(manifestPath)
	svc := ensureService(services, dir)

	svc.Path = dir
	if svc.Name == "" {
		svc.Name = filepath.Base(dir)
		if svc.Name == "." || svc.Name == string(filepath.Separator) {
			svc.Name = "root"
		}
	}
	svc.Language = "Python"
	svc.DependencyFiles = appendUnique(svc.DependencyFiles, append([]string{manifestPath}, filesByNameInDir(idx, dir, "requirements.txt", "pyproject.toml", "Pipfile", "poetry.lock")...)...)
	svc.ConfigFiles = appendUnique(svc.ConfigFiles, filesByNameInDir(idx, dir, ".python-version", "runtime.txt", "pytest.ini", "tox.ini")...)
	svc.RuntimeVersion = detectPythonRuntimeInDir(idx, dir)

	dependencies := parsePythonDependencies(idx, dir)
	svc.Dependencies = appendUnique(svc.Dependencies, dependencies...)
	svc.Framework = detectPythonFrameworkFromDeps(dependencies)
	svc.PackageManager = detectPythonPackageManager(idx, dir)
	svc.BuildTool = svc.PackageManager
	svc.Commands = buildPythonCommands(svc.PackageManager, idx, dir)
}

func enrichJavaService(idx *fileIndex, services map[string]*ServiceProfile, manifestPath string) {
	dir := pathDir(manifestPath)
	svc := ensureService(services, dir)
	content, err := idx.readFile(manifestPath)
	if err != nil {
		return
	}

	svc.Path = dir
	if svc.Name == "" {
		svc.Name = filepath.Base(dir)
		if svc.Name == "." || svc.Name == string(filepath.Separator) {
			svc.Name = "root"
		}
	}
	svc.Language = "Java"
	svc.DependencyFiles = appendUnique(svc.DependencyFiles, manifestPath)
	svc.Framework = detectJavaFrameworkFromContent(string(content))
	switch filepath.Base(manifestPath) {
	case "pom.xml":
		svc.PackageManager = "maven"
		svc.BuildTool = "maven"
		svc.Commands = CommandHints{
			Install: "mvn -B dependency:resolve",
			Test:    "mvn -B test",
			Build:   "mvn -B package",
		}
	default:
		svc.PackageManager = "gradle"
		svc.BuildTool = "gradle"
		svc.Commands = CommandHints{
			Install: "./gradlew dependencies",
			Test:    "./gradlew test",
			Build:   "./gradlew build",
		}
	}
}

func enrichRustService(idx *fileIndex, services map[string]*ServiceProfile, manifestPath string) {
	dir := pathDir(manifestPath)
	svc := ensureService(services, dir)
	svc.Path = dir
	if svc.Name == "" {
		svc.Name = filepath.Base(dir)
		if svc.Name == "." || svc.Name == string(filepath.Separator) {
			svc.Name = "root"
		}
	}
	svc.Language = "Rust"
	svc.PackageManager = "cargo"
	svc.BuildTool = "cargo"
	svc.DependencyFiles = appendUnique(svc.DependencyFiles, append([]string{manifestPath}, filesByNameInDir(idx, dir, "Cargo.lock")...)...)
	svc.Commands = CommandHints{
		Install: "cargo fetch",
		Test:    "cargo test",
		Build:   "cargo build --release",
		Run:     "cargo run",
	}
}

func enrichPHPService(idx *fileIndex, services map[string]*ServiceProfile, manifestPath string) {
	dir := pathDir(manifestPath)
	svc := ensureService(services, dir)
	data, err := idx.readFile(manifestPath)
	if err != nil {
		return
	}
	var manifest struct {
		Name    string            `json:"name"`
		Require map[string]string `json:"require"`
	}
	_ = json.Unmarshal(data, &manifest)

	svc.Path = dir
	if manifest.Name != "" {
		svc.Name = manifest.Name
	}
	if svc.Name == "" {
		svc.Name = filepath.Base(dir)
	}
	svc.Language = "PHP"
	svc.PackageManager = "composer"
	svc.BuildTool = "composer"
	svc.DependencyFiles = appendUnique(svc.DependencyFiles, append([]string{manifestPath}, filesByNameInDir(idx, dir, "composer.lock")...)...)
	svc.Dependencies = appendUnique(svc.Dependencies, mapKeys(manifest.Require)...)
	svc.Framework = detectPHPFrameworkFromDeps(svc.Dependencies)
	svc.Commands = CommandHints{
		Install: "composer install --no-interaction",
		Test:    "vendor/bin/phpunit",
		Build:   "composer dump-autoload --optimize",
	}
}

func enrichDotNetService(idx *fileIndex, services map[string]*ServiceProfile, manifestPath string) {
	dir := pathDir(manifestPath)
	svc := ensureService(services, dir)
	svc.Path = dir
	if svc.Name == "" {
		svc.Name = strings.TrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))
	}
	svc.Language = ".NET"
	svc.PackageManager = "nuget"
	svc.BuildTool = "dotnet"
	svc.DependencyFiles = appendUnique(svc.DependencyFiles, manifestPath)
	svc.Commands = CommandHints{
		Install: "dotnet restore",
		Test:    "dotnet test",
		Build:   "dotnet build --configuration Release",
		Run:     "dotnet run",
	}
}

func ensureService(services map[string]*ServiceProfile, dir string) *ServiceProfile {
	dir = normalizeDir(dir)
	if svc, ok := services[dir]; ok {
		return svc
	}
	name := filepath.Base(dir)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "root"
	}
	svc := &ServiceProfile{Name: name, Path: dir}
	services[dir] = svc
	return svc
}

func choosePrimaryService(services []ServiceProfile, detections []DetectionResult) *ServiceProfile {
	if len(services) == 0 {
		return nil
	}
	wantLanguage := ""
	if len(detections) > 0 {
		wantLanguage = detections[0].Language
	}
	bestIdx := 0
	bestScore := -1
	for i, svc := range services {
		score := len(svc.DependencyFiles) * 5
		if svc.Path == "." {
			score += 25
		}
		if svc.Framework != "" {
			score += 10
		}
		if svc.Commands.Build != "" || svc.Commands.Test != "" {
			score += 10
		}
		if wantLanguage != "" && svc.Language == wantLanguage {
			score += 40
		}
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	return &services[bestIdx]
}

func fallbackCommands(detections []DetectionResult) CommandHints {
	if len(detections) == 0 {
		return CommandHints{}
	}
	switch detections[0].Language {
	case "Go":
		return CommandHints{Install: "go mod download", Lint: "go vet ./...", Test: "go test ./...", Build: "go build ./..."}
	case "Node.js":
		return CommandHints{Install: "npm ci", Test: "npm test", Build: "npm run build"}
	case "Python":
		return CommandHints{Install: "pip install -r requirements.txt", Test: "pytest"}
	default:
		return CommandHints{}
	}
}

func buildNodeCommands(packageManager string, scripts map[string]string) CommandHints {
	install := "npm ci"
	switch packageManager {
	case "pnpm":
		install = "corepack enable && pnpm install --frozen-lockfile"
	case "yarn":
		install = "yarn install --frozen-lockfile"
	case "bun":
		install = "bun install"
	}

	cmds := CommandHints{Install: install}
	if hasUsefulScript(scripts, "lint") {
		cmds.Lint = scriptRunner(packageManager, "lint")
	}
	if hasUsefulScript(scripts, "test") {
		cmds.Test = buildNodeTestCommand(packageManager, scripts["test"])
	}
	if hasUsefulScript(scripts, "build") {
		cmds.Build = scriptRunner(packageManager, "build")
	}
	if hasUsefulScript(scripts, "dev") {
		cmds.Run = scriptRunner(packageManager, "dev")
	} else if hasUsefulScript(scripts, "start") {
		cmds.Run = scriptRunner(packageManager, "start")
	}
	return cmds
}

func buildGoCommands(idx *fileIndex, dir string) CommandHints {
	cmds := CommandHints{
		Install: "go mod download",
		Lint:    "go vet ./...",
		Test:    "go test ./...",
		Build:   "go build ./...",
	}
	runTarget := detectGoRunTarget(idx, dir)
	if runTarget != "" {
		cmds.Run = "go run " + runTarget
	}
	return cmds
}

func buildPythonCommands(packageManager string, idx *fileIndex, dir string) CommandHints {
	cmds := CommandHints{}
	switch packageManager {
	case "poetry":
		cmds.Install = "pip install poetry && poetry install --no-interaction"
		cmds.Lint = "poetry run ruff check ."
		cmds.Test = "poetry run pytest"
		if hasPythonEntrypoint(idx, dir) {
			cmds.Run = "poetry run python ."
		}
	case "pipenv":
		cmds.Install = "pip install pipenv && pipenv install --dev"
		cmds.Lint = "pipenv run ruff check ."
		cmds.Test = "pipenv run pytest"
		if hasPythonEntrypoint(idx, dir) {
			cmds.Run = "pipenv run python ."
		}
	default:
		reqFile := "requirements.txt"
		if firstFileByNameInDir(idx, dir, "requirements.txt") == "" && firstFileByNameInDir(idx, dir, "pyproject.toml") != "" {
			reqFile = ""
		}
		if reqFile != "" {
			cmds.Install = "pip install -r requirements.txt"
		} else {
			cmds.Install = "pip install ."
		}
		cmds.Lint = "ruff check ."
		cmds.Test = "pytest"
		if hasPythonEntrypoint(idx, dir) {
			cmds.Run = "python ."
		}
	}
	return cmds
}

func collectEnvFiles(idx *fileIndex) []string {
	var files []string
	for _, name := range []string{".env", ".env.example", ".env.local", ".env.development", ".env.production", "env.example"} {
		files = append(files, idx.byName[name]...)
	}
	return uniqueStrings(files)
}

func collectConfigFiles(idx *fileIndex) []string {
	var files []string
	for _, name := range []string{
		"Dockerfile", "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml",
		"Chart.yaml", "terraform.tfvars", "Makefile", "go.work", "package.json", "pyproject.toml",
		"vite.config.ts", "vite.config.js", "next.config.js", "next.config.mjs", "nuxt.config.ts",
		"tsconfig.json", "pom.xml", "build.gradle", "build.gradle.kts", "Cargo.toml",
	} {
		files = append(files, idx.byName[name]...)
	}
	return uniqueStrings(files)
}

func collectDeploymentTargets(idx *fileIndex) []DeploymentTarget {
	var targets []DeploymentTarget
	for name, paths := range idx.byName {
		lower := strings.ToLower(name)
		switch {
		case name == "Dockerfile" || strings.HasPrefix(lower, "dockerfile.") || strings.HasSuffix(lower, ".dockerfile"):
			for _, p := range paths {
				targets = append(targets, DeploymentTarget{Type: "docker", Path: p, Tool: "docker"})
			}
		case name == "docker-compose.yml" || name == "docker-compose.yaml" || name == "compose.yml" || name == "compose.yaml":
			for _, p := range paths {
				targets = append(targets, DeploymentTarget{Type: "compose", Path: p, Tool: "docker-compose"})
			}
		case name == "Chart.yaml":
			for _, p := range paths {
				targets = append(targets, DeploymentTarget{Type: "helm", Path: p, Tool: "helm"})
			}
		}
	}
	for _, p := range idx.byExt[".tf"] {
		targets = append(targets, DeploymentTarget{Type: "terraform", Path: p, Tool: "terraform"})
	}
	for _, p := range append(idx.byExt[".yaml"], idx.byExt[".yml"]...) {
		if strings.Contains(p, "k8s/") || strings.Contains(p, "kubernetes/") || strings.Contains(p, "manifests/") || strings.Contains(p, "deploy/") {
			targets = append(targets, DeploymentTarget{Type: "kubernetes", Path: p, Tool: "kubectl"})
		}
	}
	return uniqueDeploymentTargets(targets)
}

func hasNodeWorkspace(idx *fileIndex) bool {
	if !idx.hasFile("package.json") {
		return false
	}
	data, err := idx.readFileByName("package.json")
	if err != nil {
		return false
	}
	var manifest packageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return false
	}
	return manifest.Workspaces != nil
}

func detectNodePackageManager(idx *fileIndex, dir, explicit string) string {
	if explicit != "" {
		switch {
		case strings.HasPrefix(explicit, "pnpm@"):
			return "pnpm"
		case strings.HasPrefix(explicit, "yarn@"):
			return "yarn"
		case strings.HasPrefix(explicit, "bun@"):
			return "bun"
		case strings.HasPrefix(explicit, "npm@"):
			return "npm"
		}
	}
	if firstFileByNameInDir(idx, dir, "pnpm-lock.yaml") != "" || idx.hasFile("pnpm-workspace.yaml") {
		return "pnpm"
	}
	if firstFileByNameInDir(idx, dir, "yarn.lock") != "" {
		return "yarn"
	}
	if firstFileByNameInDir(idx, dir, "package-lock.json") != "" {
		return "npm"
	}
	return "npm"
}

func detectPythonPackageManager(idx *fileIndex, dir string) string {
	if firstFileByNameInDir(idx, dir, "poetry.lock") != "" {
		return "poetry"
	}
	if data, err := readFirstNamedFileInDir(idx, dir, "pyproject.toml"); err == nil && strings.Contains(string(data), "[tool.poetry]") {
		return "poetry"
	}
	if firstFileByNameInDir(idx, dir, "Pipfile") != "" {
		return "pipenv"
	}
	return "pip"
}

func detectPythonRuntimeInDir(idx *fileIndex, dir string) string {
	for _, name := range []string{".python-version", "runtime.txt"} {
		data, err := readFirstNamedFileInDir(idx, dir, name)
		if err != nil {
			continue
		}
		value := strings.TrimSpace(string(data))
		if name == "runtime.txt" {
			value = strings.TrimPrefix(value, "python-")
		}
		if value != "" {
			return value
		}
	}
	return ""
}

func detectGoRunTarget(idx *fileIndex, dir string) string {
	if hasFileInDir(idx, dir, "main.go") {
		return "."
	}
	if dir == "." {
		var candidates []string
		for _, file := range idx.byName["main.go"] {
			if strings.HasPrefix(file, "cmd/") && strings.Count(file, "/") == 2 {
				candidates = append(candidates, strings.TrimSuffix(file, "/main.go"))
			}
		}
		sort.Strings(candidates)
		if len(candidates) > 0 {
			return "./" + candidates[0]
		}
	}
	return ""
}

func hasPythonEntrypoint(idx *fileIndex, dir string) bool {
	for _, file := range idx.byExt[".py"] {
		if dir != "." && !strings.HasPrefix(file, dir+"/") {
			continue
		}
		base := filepath.Base(file)
		if base == "main.py" || base == "manage.py" || base == "__main__.py" || base == "app.py" {
			return true
		}
	}
	return false
}

func parsePythonDependencies(idx *fileIndex, dir string) []string {
	var deps []string
	for _, name := range []string{"requirements.txt", "requirements.in", "pyproject.toml", "Pipfile"} {
		data, err := readFirstNamedFileInDir(idx, dir, name)
		if err != nil {
			continue
		}
		for _, line := range splitLines(string(data)) {
			normalized := normalizeDependencyName(line)
			if normalized != "" {
				deps = append(deps, normalized)
			}
		}
	}
	return uniqueStrings(deps)
}

func parseGoDependencies(content string) []string {
	var deps []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	inBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "require (") {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		if strings.HasPrefix(line, "require ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		}
		if !inBlock && !strings.HasPrefix(line, "require ") && !strings.Contains(content, "require (") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && !strings.HasPrefix(fields[0], "//") && fields[0] != ")" {
			deps = append(deps, fields[0])
		}
	}
	return uniqueStrings(deps)
}

func detectNodeFrameworkFromMaps(deps ...map[string]string) string {
	all := map[string]string{}
	for _, depMap := range deps {
		for k, v := range depMap {
			all[k] = v
		}
	}

	frameworks := []struct {
		name      string
		framework string
	}{
		{"next", "Next.js"},
		{"nuxt", "Nuxt.js"},
		{"@angular/core", "Angular"},
		{"vue", "Vue.js"},
		{"svelte", "Svelte"},
		{"solid-js", "SolidJS"},
		{"react", "React"},
		{"express", "Express"},
		{"fastify", "Fastify"},
		{"koa", "Koa"},
		{"@nestjs/core", "NestJS"},
		{"gatsby", "Gatsby"},
		{"remix", "Remix"},
		{"astro", "Astro"},
		{"hono", "Hono"},
	}

	var detected []string
	for _, framework := range frameworks {
		if _, ok := all[framework.name]; ok {
			detected = append(detected, framework.framework)
		}
	}
	return strings.Join(uniqueStrings(detected), ", ")
}

func detectPythonFrameworkFromDeps(deps []string) string {
	mapping := map[string]string{
		"django":    "Django",
		"flask":     "Flask",
		"fastapi":   "FastAPI",
		"starlette": "Starlette",
		"streamlit": "Streamlit",
		"celery":    "Celery",
		"sanic":     "Sanic",
	}
	var frameworks []string
	for _, dep := range deps {
		if framework, ok := mapping[strings.ToLower(dep)]; ok {
			frameworks = append(frameworks, framework)
		}
	}
	return strings.Join(uniqueStrings(frameworks), ", ")
}

func detectGoFrameworkFromContent(content string) string {
	mapping := map[string]string{
		"github.com/gofiber/fiber":    "Fiber",
		"github.com/gofiber/fiber/v2": "Fiber",
		"github.com/gofiber/fiber/v3": "Fiber",
		"github.com/gin-gonic/gin":    "Gin",
		"github.com/labstack/echo":    "Echo",
		"github.com/gorilla/mux":      "Gorilla Mux",
		"github.com/go-chi/chi":       "Chi",
		"github.com/valyala/fasthttp": "FastHTTP",
	}
	var frameworks []string
	for module, name := range mapping {
		if strings.Contains(content, module) {
			frameworks = append(frameworks, name)
		}
	}
	return strings.Join(uniqueStrings(frameworks), ", ")
}

func detectJavaFrameworkFromContent(content string) string {
	mapping := map[string]string{
		"spring-boot":   "Spring Boot",
		"micronaut":     "Micronaut",
		"quarkus":       "Quarkus",
		"jakarta.ws.rs": "Jakarta REST",
		"io.dropwizard": "Dropwizard",
		"vertx":         "Vert.x",
	}
	var frameworks []string
	lower := strings.ToLower(content)
	for marker, name := range mapping {
		if strings.Contains(lower, marker) {
			frameworks = append(frameworks, name)
		}
	}
	return strings.Join(uniqueStrings(frameworks), ", ")
}

func detectPHPFrameworkFromDeps(deps []string) string {
	mapping := map[string]string{
		"laravel/framework":        "Laravel",
		"symfony/framework-bundle": "Symfony",
		"codeigniter4/framework":   "CodeIgniter",
		"slim/slim":                "Slim",
	}
	var frameworks []string
	for _, dep := range deps {
		if framework, ok := mapping[dep]; ok {
			frameworks = append(frameworks, framework)
		}
	}
	return strings.Join(uniqueStrings(frameworks), ", ")
}

func normalizeDependencyName(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
		return ""
	}
	line = strings.Trim(line, `"' ,`)
	for _, sep := range []string{"==", ">=", "<=", "~=", "!=", ">", "<", "[", "{", " ", "="} {
		if i := strings.Index(line, sep); i >= 0 {
			line = line[:i]
		}
	}
	line = strings.TrimSpace(strings.Trim(line, `"' ,`))
	if line == "" || strings.Contains(line, "/") && !strings.Contains(line, ".") && !strings.Contains(line, "-") {
		return ""
	}
	return line
}

func hasUsefulScript(scripts map[string]string, name string) bool {
	value := strings.TrimSpace(scripts[name])
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	return !strings.Contains(lower, "no test specified")
}

func scriptRunner(packageManager, script string) string {
	switch packageManager {
	case "pnpm":
		return "corepack enable && pnpm " + script
	case "yarn":
		return "yarn " + script
	case "bun":
		return "bun run " + script
	default:
		return "npm run " + script
	}
}

func buildNodeTestCommand(packageManager, scriptBody string) string {
	base := scriptRunner(packageManager, "test")
	scriptBody = strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(scriptBody))), " ")
	extraArgs := ""

	switch {
	case strings.Contains(scriptBody, "react-scripts test"), strings.Contains(scriptBody, "craco test"):
		extraArgs = " -- --watch=false --passWithNoTests"
	case strings.Contains(scriptBody, "jest") &&
		!strings.Contains(scriptBody, "--watchall=false") &&
		!strings.Contains(scriptBody, "--watch=false"):
		extraArgs = " -- --watchAll=false --passWithNoTests"
	case strings.Contains(scriptBody, "vitest") && !strings.Contains(scriptBody, "vitest run"):
		extraArgs = " -- --run"
	}

	return "export CI=true\n" + base + extraArgs
}

func normalizeSemver(value string) string {
	return strings.TrimLeft(strings.TrimSpace(value), "^~>=<v ")
}

func extractGoVersionFromData(data []byte) string {
	m := goVersionRe.FindSubmatch(data)
	if len(m) >= 2 {
		return string(m[1])
	}
	return ""
}

func pathDir(rel string) string {
	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." || dir == "" {
		return "."
	}
	return dir
}

func normalizeDir(dir string) string {
	if dir == "" {
		return "."
	}
	dir = filepath.ToSlash(dir)
	if dir == "." {
		return "."
	}
	return strings.TrimPrefix(dir, "./")
}

func firstFileByNameInDir(idx *fileIndex, dir, name string) string {
	dir = normalizeDir(dir)
	for _, candidate := range idx.byName[name] {
		if pathDir(candidate) == dir {
			return candidate
		}
	}
	return ""
}

func filesByNameInDir(idx *fileIndex, dir string, names ...string) []string {
	var files []string
	for _, name := range names {
		if path := firstFileByNameInDir(idx, dir, name); path != "" {
			files = append(files, path)
		}
	}
	return uniqueStrings(files)
}

func readFirstNamedFileInDir(idx *fileIndex, dir, name string) ([]byte, error) {
	path := firstFileByNameInDir(idx, dir, name)
	if path == "" {
		return nil, os.ErrNotExist
	}
	return idx.readFile(path)
}

func hasFileInDir(idx *fileIndex, dir, name string) bool {
	return firstFileByNameInDir(idx, dir, name) != ""
}

func findFirstPath(idx *fileIndex, names ...string) string {
	for _, name := range names {
		if path := idx.firstPathForName(name); path != "" {
			return path
		}
	}
	return ""
}

func mapKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func appendUnique(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	for _, value := range additions {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func uniqueStrings(values []string) []string {
	var result []string
	for _, value := range values {
		result = appendUnique(result, value)
	}
	sort.Strings(result)
	return result
}

func uniqueDeploymentTargets(values []DeploymentTarget) []DeploymentTarget {
	seen := map[string]struct{}{}
	var result []DeploymentTarget
	for _, value := range values {
		key := value.Type + "|" + value.Path + "|" + value.Tool
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}
