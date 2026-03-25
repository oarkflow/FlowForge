package detector

import (
	"encoding/json"
	"strings"
)

// frameworkMatcher defines how to detect a framework for a given language.
type frameworkMatcher struct {
	language  string
	matchFunc func(idx *fileIndex) (framework string, found bool)
}

// detectFrameworks enriches existing DetectionResults with framework information.
// It modifies results in place and may add new results if a framework implies
// a language that was not yet detected.
func detectFrameworks(idx *fileIndex, results []DetectionResult) []DetectionResult {
	matchers := allFrameworkMatchers()

	// Index existing results by language for quick lookup.
	byLang := make(map[string]int, len(results))
	for i, r := range results {
		byLang[r.Language] = i
	}

	for _, m := range matchers {
		framework, found := m.matchFunc(idx)
		if !found {
			continue
		}
		if i, ok := byLang[m.language]; ok {
			// Enrich existing result.
			if results[i].Framework == "" {
				results[i].Framework = framework
			} else if !strings.Contains(results[i].Framework, framework) {
				results[i].Framework = results[i].Framework + ", " + framework
			}
		} else {
			// Framework detected but language was not. Add a new result with moderate confidence.
			r := DetectionResult{
				Language:   m.language,
				Framework:  framework,
				Confidence: 0.4,
			}
			results = append(results, r)
			byLang[m.language] = len(results) - 1
		}
	}

	return results
}

func allFrameworkMatchers() []frameworkMatcher {
	return []frameworkMatcher{
		// Node.js frameworks — detected from package.json dependencies.
		{language: "Node.js", matchFunc: detectNodeFramework},
		// Python frameworks — detected from requirements.txt or pyproject.toml.
		{language: "Python", matchFunc: detectPythonFramework},
		// Ruby frameworks — detected from Gemfile.
		{language: "Ruby", matchFunc: detectRubyFramework},
		// Java frameworks — detected from pom.xml or build.gradle.
		{language: "Java", matchFunc: detectJavaFramework},
		// PHP frameworks — detected from composer.json.
		{language: "PHP", matchFunc: detectPHPFramework},
		// Go frameworks — detected from go.mod.
		{language: "Go", matchFunc: detectGoFramework},
		// Scala frameworks — detected from build.sbt.
		{language: "Scala", matchFunc: detectScalaFramework},
	}
}

// ---------------------------------------------------------------------------
// Node.js framework detection
// ---------------------------------------------------------------------------

// packageJSON is a minimal representation of package.json for dependency extraction.
type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func detectNodeFramework(idx *fileIndex) (string, bool) {
	data, err := idx.readFileByName("package.json")
	if err != nil {
		return "", false
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", false
	}

	allDeps := mergeStringMaps(pkg.Dependencies, pkg.DevDependencies)

	// Order matters: check more specific frameworks first.
	nodeFrameworks := []struct {
		dep       string
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
		{"@hono/node-server", "Hono"},
		{"hono", "Hono"},
	}

	var detected []string
	for _, nf := range nodeFrameworks {
		if _, ok := allDeps[nf.dep]; ok {
			detected = append(detected, nf.framework)
		}
	}

	if len(detected) == 0 {
		return "", false
	}
	return strings.Join(detected, ", "), true
}

// ---------------------------------------------------------------------------
// Python framework detection
// ---------------------------------------------------------------------------

func detectPythonFramework(idx *fileIndex) (string, bool) {
	var lines []string

	// Collect dependency lines from all relevant files.
	for _, name := range []string{"requirements.txt", "requirements.in"} {
		if data, err := idx.readFileByName(name); err == nil {
			lines = append(lines, splitLines(string(data))...)
		}
	}

	// Also scan pyproject.toml for dependencies.
	if data, err := idx.readFileByName("pyproject.toml"); err == nil {
		lines = append(lines, splitLines(string(data))...)
	}

	// Also scan Pipfile.
	if data, err := idx.readFileByName("Pipfile"); err == nil {
		lines = append(lines, splitLines(string(data))...)
	}

	pyFrameworks := []struct {
		pkg       string
		framework string
	}{
		{"django", "Django"},
		{"Django", "Django"},
		{"flask", "Flask"},
		{"Flask", "Flask"},
		{"fastapi", "FastAPI"},
		{"FastAPI", "FastAPI"},
		{"tornado", "Tornado"},
		{"starlette", "Starlette"},
		{"sanic", "Sanic"},
		{"pyramid", "Pyramid"},
		{"aiohttp", "aiohttp"},
		{"celery", "Celery"},
		{"streamlit", "Streamlit"},
	}

	detected := make(map[string]bool)
	for _, line := range lines {
		// Normalize: take the package name before any version specifier.
		normalized := strings.ToLower(strings.TrimSpace(line))
		for _, sep := range []string{"==", ">=", "<=", "~=", "!=", ">", "<", "[", " "} {
			if i := strings.Index(normalized, sep); i >= 0 {
				normalized = normalized[:i]
			}
		}
		normalized = strings.TrimSpace(normalized)

		for _, pf := range pyFrameworks {
			if strings.EqualFold(normalized, pf.pkg) {
				detected[pf.framework] = true
			}
		}
	}

	if len(detected) == 0 {
		return "", false
	}

	var frameworks []string
	for f := range detected {
		frameworks = append(frameworks, f)
	}
	return strings.Join(frameworks, ", "), true
}

// ---------------------------------------------------------------------------
// Ruby framework detection
// ---------------------------------------------------------------------------

func detectRubyFramework(idx *fileIndex) (string, bool) {
	data, err := idx.readFileByName("Gemfile")
	if err != nil {
		return "", false
	}
	content := string(data)

	rubyFrameworks := []struct {
		gem       string
		framework string
	}{
		{"rails", "Rails"},
		{"sinatra", "Sinatra"},
		{"hanami", "Hanami"},
		{"padrino", "Padrino"},
		{"grape", "Grape"},
		{"roda", "Roda"},
	}

	var detected []string
	for _, rf := range rubyFrameworks {
		// Match gem declarations like: gem 'rails' or gem "rails"
		if strings.Contains(content, `"`+rf.gem+`"`) || strings.Contains(content, `'`+rf.gem+`'`) {
			detected = append(detected, rf.framework)
		}
	}

	if len(detected) == 0 {
		return "", false
	}
	return strings.Join(detected, ", "), true
}

// ---------------------------------------------------------------------------
// Java framework detection
// ---------------------------------------------------------------------------

func detectJavaFramework(idx *fileIndex) (string, bool) {
	var content string

	if data, err := idx.readFileByName("pom.xml"); err == nil {
		content = string(data)
	} else if data, err := idx.readFileByName("build.gradle"); err == nil {
		content = string(data)
	} else if data, err := idx.readFileByName("build.gradle.kts"); err == nil {
		content = string(data)
	} else {
		return "", false
	}

	javaFrameworks := []struct {
		indicator string
		framework string
	}{
		{"spring-boot", "Spring Boot"},
		{"org.springframework.boot", "Spring Boot"},
		{"spring-cloud", "Spring Cloud"},
		{"io.micronaut", "Micronaut"},
		{"io.quarkus", "Quarkus"},
		{"io.dropwizard", "Dropwizard"},
		{"play-java", "Play Framework"},
		{"io.vertx", "Vert.x"},
		{"org.apache.struts", "Struts"},
	}

	var detected []string
	for _, jf := range javaFrameworks {
		if strings.Contains(content, jf.indicator) {
			detected = append(detected, jf.framework)
		}
	}

	if len(detected) == 0 {
		return "", false
	}
	return strings.Join(detected, ", "), true
}

// ---------------------------------------------------------------------------
// PHP framework detection
// ---------------------------------------------------------------------------

func detectPHPFramework(idx *fileIndex) (string, bool) {
	data, err := idx.readFileByName("composer.json")
	if err != nil {
		return "", false
	}

	var composerJSON struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if err := json.Unmarshal(data, &composerJSON); err != nil {
		return "", false
	}

	allDeps := mergeStringMaps(composerJSON.Require, composerJSON.RequireDev)

	phpFrameworks := []struct {
		dep       string
		framework string
	}{
		{"laravel/framework", "Laravel"},
		{"symfony/framework-bundle", "Symfony"},
		{"slim/slim", "Slim"},
		{"cakephp/cakephp", "CakePHP"},
		{"yiisoft/yii2", "Yii2"},
		{"codeigniter4/framework", "CodeIgniter"},
		{"hyperf/hyperf", "Hyperf"},
	}

	var detected []string
	for _, pf := range phpFrameworks {
		if _, ok := allDeps[pf.dep]; ok {
			detected = append(detected, pf.framework)
		}
	}

	// Also check for the artisan file as a Laravel indicator.
	if idx.hasFile("artisan") && !containsStr(detected, "Laravel") {
		detected = append(detected, "Laravel")
	}

	if len(detected) == 0 {
		return "", false
	}
	return strings.Join(detected, ", "), true
}

// ---------------------------------------------------------------------------
// Go framework detection
// ---------------------------------------------------------------------------

func detectGoFramework(idx *fileIndex) (string, bool) {
	data, err := idx.readFileByName("go.mod")
	if err != nil {
		return "", false
	}
	content := string(data)

	goFrameworks := []struct {
		module    string
		framework string
	}{
		{"github.com/gofiber/fiber", "Fiber"},
		{"github.com/gin-gonic/gin", "Gin"},
		{"github.com/labstack/echo", "Echo"},
		{"github.com/gorilla/mux", "Gorilla Mux"},
		{"github.com/go-chi/chi", "Chi"},
		{"github.com/beego/beego", "Beego"},
		{"github.com/valyala/fasthttp", "FastHTTP"},
		{"github.com/julienschmidt/httprouter", "httprouter"},
		{"github.com/gofiber/fiber/v3", "Fiber"},
		{"github.com/gofiber/fiber/v2", "Fiber"},
	}

	detected := make(map[string]bool)
	for _, gf := range goFrameworks {
		if strings.Contains(content, gf.module) {
			detected[gf.framework] = true
		}
	}

	if len(detected) == 0 {
		return "", false
	}

	var frameworks []string
	for f := range detected {
		frameworks = append(frameworks, f)
	}
	return strings.Join(frameworks, ", "), true
}

// ---------------------------------------------------------------------------
// Scala framework detection
// ---------------------------------------------------------------------------

func detectScalaFramework(idx *fileIndex) (string, bool) {
	data, err := idx.readFileByName("build.sbt")
	if err != nil {
		return "", false
	}
	content := string(data)

	scalaFrameworks := []struct {
		indicators []string
		framework  string
	}{
		{[]string{"com.typesafe.play", "play-scala"}, "Play Framework"},
		{[]string{"com.typesafe.akka", "akka-actor"}, "Akka"},
		{[]string{"org.http4s"}, "http4s"},
		{[]string{"dev.zio"}, "ZIO"},
		{[]string{"io.spray"}, "Spray"},
		{[]string{"com.twitter"}, "Finatra"},
		{[]string{"org.scalatra"}, "Scalatra"},
	}

	var detected []string
	for _, sf := range scalaFrameworks {
		for _, indicator := range sf.indicators {
			if strings.Contains(content, indicator) {
				if !containsStr(detected, sf.framework) {
					detected = append(detected, sf.framework)
				}
				break
			}
		}
	}

	// Disambiguate Finatra vs Finagle for Twitter libs.
	if containsStr(detected, "Finatra") {
		if strings.Contains(content, "finagle") && !strings.Contains(content, "finatra") {
			// Replace Finatra with Finagle.
			for i, f := range detected {
				if f == "Finatra" {
					detected[i] = "Finagle"
				}
			}
		}
	}

	if len(detected) == 0 {
		return "", false
	}
	return strings.Join(detected, ", "), true
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mergeStringMaps(maps ...map[string]string) map[string]string {
	merged := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines
}

func containsStr(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
