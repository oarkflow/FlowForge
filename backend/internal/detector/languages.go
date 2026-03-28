package detector

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
)

// languageDetector describes a single language detection strategy.
type languageDetector struct {
	language       string
	indicatorFiles []string   // presence of any of these files indicates the language
	extensions     []string   // source file extensions (e.g. ".go")
	buildTools     []string   // names of build tool config files
	detect         func(idx *fileIndex) *DetectionResult // optional custom detector
}

// builtinDetectors returns all built-in language detectors.
func builtinDetectors() []languageDetector {
	return []languageDetector{
		goDetector(),
		nodeDetector(),
		pythonDetector(),
		rubyDetector(),
		javaDetector(),
		kotlinDetector(),
		rustDetector(),
		phpDetector(),
		dotnetDetector(),
		swiftDetector(),
		scalaDetector(),
	}
}

// detectLanguages runs every language detector against the file index and
// returns the combined results.
func detectLanguages(idx *fileIndex) []DetectionResult {
	var results []DetectionResult
	for _, det := range builtinDetectors() {
		if det.detect != nil {
			if r := det.detect(idx); r != nil {
				results = append(results, *r)
			}
			continue
		}
		if r := runGenericDetector(idx, det); r != nil {
			results = append(results, *r)
		}
	}
	return results
}

// detectInfrastructure runs every infrastructure detector against the file
// index and returns the combined results.
func detectInfrastructure(idx *fileIndex) []DetectionResult {
	var results []DetectionResult
	for _, det := range infrastructureDetectors() {
		if det.detect != nil {
			if r := det.detect(idx); r != nil {
				results = append(results, *r)
			}
			continue
		}
		if r := runGenericDetector(idx, det); r != nil {
			results = append(results, *r)
		}
	}
	return results
}

// runGenericDetector uses indicator files and extension counts to determine
// whether a language is present and with what confidence.
func runGenericDetector(idx *fileIndex, det languageDetector) *DetectionResult {
	var confidence float64
	var depFile string
	var buildTool string

	// Check indicator / dependency files.
	for _, f := range det.indicatorFiles {
		if idx.hasFile(f) {
			confidence += 0.5
			if depFile == "" {
				depFile = idx.firstPathForName(f)
			}
			break
		}
	}

	// Check for build tool files.
	for _, bt := range det.buildTools {
		if idx.hasFile(bt) {
			if buildTool == "" {
				buildTool = bt
			}
			confidence += 0.1
		}
	}

	// Check source file extensions.
	for _, ext := range det.extensions {
		count := idx.countExt(ext)
		if count > 0 {
			// Scale confidence with file count, capping the contribution at 0.4.
			extConf := float64(count) * 0.02
			if extConf > 0.4 {
				extConf = 0.4
			}
			confidence += extConf
		}
	}

	if confidence <= 0 {
		return nil
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return &DetectionResult{
		Language:       det.language,
		DependencyFile: depFile,
		Confidence:     confidence,
		BuildTool:      buildTool,
	}
}

// ---------------------------------------------------------------------------
// Individual language detectors
// ---------------------------------------------------------------------------

func goDetector() languageDetector {
	return languageDetector{
		language: "Go",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile, buildTool, runtimeVersion string

			if idx.hasFile("go.mod") {
				confidence += 0.5
				depFile = idx.firstPathForName("go.mod")
				runtimeVersion = extractGoVersion(idx)
			}
			if idx.hasFile("go.sum") {
				confidence += 0.1
			}

			goCount := idx.countExt(".go")
			if goCount > 0 {
				c := float64(goCount) * 0.02
				if c > 0.4 {
					c = 0.4
				}
				confidence += c
			}

			if idx.hasFile("Makefile") {
				buildTool = "make"
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Go",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      buildTool,
				RuntimeVersion: runtimeVersion,
			}
		},
	}
}

func nodeDetector() languageDetector {
	return languageDetector{
		language: "Node.js",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile, buildTool, runtimeVersion string

			if idx.hasFile("package.json") {
				confidence += 0.5
				depFile = idx.firstPathForName("package.json")
				runtimeVersion = extractNodeVersion(idx)
			}
			if idx.hasFile("package-lock.json") || idx.hasFile("yarn.lock") || idx.hasFile("pnpm-lock.yaml") {
				confidence += 0.1
			}

			jsCount := idx.countExt(".js") + idx.countExt(".jsx")
			tsCount := idx.countExt(".ts") + idx.countExt(".tsx")
			totalCount := jsCount + tsCount
			if totalCount > 0 {
				c := float64(totalCount) * 0.02
				if c > 0.4 {
					c = 0.4
				}
				confidence += c
			}

			// Determine package manager as build tool.
			if idx.hasFile("pnpm-lock.yaml") {
				buildTool = "pnpm"
			} else if idx.hasFile("yarn.lock") {
				buildTool = "yarn"
			} else if idx.hasFile("package-lock.json") {
				buildTool = "npm"
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Node.js",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      buildTool,
				RuntimeVersion: runtimeVersion,
			}
		},
	}
}

func pythonDetector() languageDetector {
	return languageDetector{
		language: "Python",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile, buildTool, runtimeVersion string

			pyIndicators := []struct {
				file string
				tool string
			}{
				{"requirements.txt", "pip"},
				{"setup.py", "setuptools"},
				{"pyproject.toml", ""},
				{"Pipfile", "pipenv"},
				{"setup.cfg", "setuptools"},
				{"poetry.lock", "poetry"},
			}
			for _, ind := range pyIndicators {
				if idx.hasFile(ind.file) {
					confidence += 0.4
					if depFile == "" {
						depFile = idx.firstPathForName(ind.file)
					}
					if buildTool == "" && ind.tool != "" {
						buildTool = ind.tool
					}
					break
				}
			}

			// Check pyproject.toml for build tool.
			if idx.hasFile("pyproject.toml") {
				if data, err := idx.readFileByName("pyproject.toml"); err == nil {
					content := string(data)
					if strings.Contains(content, "[tool.poetry]") {
						buildTool = "poetry"
					} else if strings.Contains(content, "[build-system]") {
						if buildTool == "" {
							buildTool = "pip"
						}
					}
				}
			}

			pyCount := idx.countExt(".py")
			if pyCount > 0 {
				c := float64(pyCount) * 0.02
				if c > 0.4 {
					c = 0.4
				}
				confidence += c
			}

			// Try to find a runtime version from .python-version or runtime.txt.
			if idx.hasFile(".python-version") {
				if data, err := idx.readFileByName(".python-version"); err == nil {
					runtimeVersion = strings.TrimSpace(string(data))
				}
			} else if idx.hasFile("runtime.txt") {
				if data, err := idx.readFileByName("runtime.txt"); err == nil {
					s := strings.TrimSpace(string(data))
					if strings.HasPrefix(s, "python-") {
						runtimeVersion = strings.TrimPrefix(s, "python-")
					}
				}
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Python",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      buildTool,
				RuntimeVersion: runtimeVersion,
			}
		},
	}
}

func rubyDetector() languageDetector {
	return languageDetector{
		language:       "Ruby",
		indicatorFiles: []string{"Gemfile", "Rakefile"},
		extensions:     []string{".rb", ".erb"},
		buildTools:     []string{"Rakefile"},
	}
}

func javaDetector() languageDetector {
	return languageDetector{
		language: "Java",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile, buildTool string

			if idx.hasFile("pom.xml") {
				confidence += 0.5
				depFile = idx.firstPathForName("pom.xml")
				buildTool = "maven"
			} else if idx.hasFile("build.gradle") {
				confidence += 0.5
				depFile = idx.firstPathForName("build.gradle")
				buildTool = "gradle"
			}

			javaCount := idx.countExt(".java")
			if javaCount > 0 {
				c := float64(javaCount) * 0.02
				if c > 0.4 {
					c = 0.4
				}
				confidence += c
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Java",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      buildTool,
			}
		},
	}
}

func kotlinDetector() languageDetector {
	return languageDetector{
		language: "Kotlin",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile, buildTool string

			if idx.hasFile("build.gradle.kts") {
				confidence += 0.5
				depFile = idx.firstPathForName("build.gradle.kts")
				buildTool = "gradle"
			}

			ktCount := idx.countExt(".kt") + idx.countExt(".kts")
			if ktCount > 0 {
				c := float64(ktCount) * 0.02
				if c > 0.4 {
					c = 0.4
				}
				confidence += c
			}

			// If build.gradle.kts is not present but .kt files exist along with build.gradle,
			// still detect Kotlin.
			if confidence <= 0 && ktCount > 0 && idx.hasFile("build.gradle") {
				confidence = 0.3
				depFile = idx.firstPathForName("build.gradle")
				buildTool = "gradle"
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Kotlin",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      buildTool,
			}
		},
	}
}

func rustDetector() languageDetector {
	return languageDetector{
		language: "Rust",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile string

			if idx.hasFile("Cargo.toml") {
				confidence += 0.5
				depFile = idx.firstPathForName("Cargo.toml")
			}
			if idx.hasFile("Cargo.lock") {
				confidence += 0.1
			}

			rsCount := idx.countExt(".rs")
			if rsCount > 0 {
				c := float64(rsCount) * 0.02
				if c > 0.4 {
					c = 0.4
				}
				confidence += c
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Rust",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      "cargo",
			}
		},
	}
}

func phpDetector() languageDetector {
	return languageDetector{
		language: "PHP",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile string

			if idx.hasFile("composer.json") {
				confidence += 0.5
				depFile = idx.firstPathForName("composer.json")
			}
			if idx.hasFile("composer.lock") {
				confidence += 0.1
			}

			phpCount := idx.countExt(".php")
			if phpCount > 0 {
				c := float64(phpCount) * 0.02
				if c > 0.4 {
					c = 0.4
				}
				confidence += c
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "PHP",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      "composer",
			}
		},
	}
}

func dotnetDetector() languageDetector {
	return languageDetector{
		language: ".NET",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile, buildTool string

			// Check for .csproj files.
			csprojCount := idx.countExt(".csproj")
			if csprojCount > 0 {
				confidence += 0.5
				paths := idx.byExt[".csproj"]
				if len(paths) > 0 {
					depFile = paths[0]
				}
				buildTool = "dotnet"
			}

			// Check for .sln files.
			slnCount := idx.countExt(".sln")
			if slnCount > 0 {
				confidence += 0.2
				if depFile == "" {
					paths := idx.byExt[".sln"]
					if len(paths) > 0 {
						depFile = paths[0]
					}
				}
				buildTool = "dotnet"
			}

			// Check for .fsproj (F#) files.
			fsprojCount := idx.countExt(".fsproj")
			if fsprojCount > 0 {
				confidence += 0.3
			}

			csCount := idx.countExt(".cs")
			fsCount := idx.countExt(".fs")
			totalCount := csCount + fsCount
			if totalCount > 0 {
				c := float64(totalCount) * 0.02
				if c > 0.3 {
					c = 0.3
				}
				confidence += c
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       ".NET",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      buildTool,
			}
		},
	}
}

func swiftDetector() languageDetector {
	return languageDetector{
		language: "Swift",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile, buildTool string

			if idx.hasFile("Package.swift") {
				confidence += 0.5
				depFile = idx.firstPathForName("Package.swift")
				buildTool = "swift"
			}

			// Check for Xcode project files.
			xcodeprojCount := idx.countExt(".xcodeproj")
			xcworkspaceCount := idx.countExt(".xcworkspace")
			if xcodeprojCount > 0 || xcworkspaceCount > 0 {
				confidence += 0.2
				if buildTool == "" {
					buildTool = "xcode"
				}
			}

			swiftCount := idx.countExt(".swift")
			if swiftCount > 0 {
				c := float64(swiftCount) * 0.02
				if c > 0.4 {
					c = 0.4
				}
				confidence += c
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Swift",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      buildTool,
			}
		},
	}
}

func scalaDetector() languageDetector {
	return languageDetector{
		language: "Scala",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile, buildTool, runtimeVersion string

			if idx.hasFile("build.sbt") {
				confidence += 0.5
				depFile = idx.firstPathForName("build.sbt")
				buildTool = "sbt"
				runtimeVersion = extractScalaVersion(idx)
			}

			scalaCount := idx.countExt(".scala")
			if scalaCount > 0 {
				c := float64(scalaCount) * 0.02
				if c > 0.4 {
					c = 0.4
				}
				confidence += c
			}

			// Also check for sbt project structure without build.sbt at root.
			if confidence <= 0 && scalaCount > 0 && idx.hasFile("build.properties") {
				confidence = 0.3
				buildTool = "sbt"
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Scala",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      buildTool,
				RuntimeVersion: runtimeVersion,
			}
		},
	}
}

// ---------------------------------------------------------------------------
// Helpers for extracting runtime versions
// ---------------------------------------------------------------------------

var goVersionRe = regexp.MustCompile(`(?m)^go\s+([\d.]+)`)

func extractGoVersion(idx *fileIndex) string {
	data, err := idx.readFileByName("go.mod")
	if err != nil {
		return ""
	}
	m := goVersionRe.FindSubmatch(data)
	if len(m) >= 2 {
		return string(m[1])
	}
	return ""
}

func extractNodeVersion(idx *fileIndex) string {
	// Try .nvmrc first.
	if idx.hasFile(".nvmrc") {
		if data, err := idx.readFileByName(".nvmrc"); err == nil {
			v := strings.TrimSpace(string(data))
			if v != "" {
				return strings.TrimPrefix(v, "v")
			}
		}
	}
	// Try .node-version.
	if idx.hasFile(".node-version") {
		if data, err := idx.readFileByName(".node-version"); err == nil {
			v := strings.TrimSpace(string(data))
			if v != "" {
				return strings.TrimPrefix(v, "v")
			}
		}
	}
	// Try engines field in package.json.
	if data, err := idx.readFileByName("package.json"); err == nil {
		return extractNodeEngineVersion(data)
	}
	return ""
}

// extractNodeEngineVersion does a minimal JSON parse to find "engines": { "node": "..." }
// without importing encoding/json to keep allocations small. Falls back gracefully.
func extractNodeEngineVersion(data []byte) string {
	// Simple line-scanning approach.
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inEngines := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, `"engines"`) {
			inEngines = true
			continue
		}
		if inEngines {
			if strings.Contains(line, "}") && !strings.Contains(line, `"node"`) {
				break
			}
			if strings.Contains(line, `"node"`) {
				// Extract the version value.
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					v := strings.Trim(parts[1], ` ",`)
					v = strings.TrimSuffix(v, ",")
					v = strings.TrimSpace(v)
					// Strip leading comparators for a clean version.
					v = strings.TrimLeft(v, "><=~^")
					return v
				}
			}
		}
	}
	return ""
}

// scalaVersionRe matches common scalaVersion assignment styles in build.sbt:
//   scalaVersion := "2.11.7"
//   scalaVersion in ThisBuild := "2.13.12"
//   ThisBuild / scalaVersion := "3.3.1"
var scalaVersionRe = regexp.MustCompile(`(?m)(?:ThisBuild\s*/\s*)?scalaVersion\s+(?:in\s+ThisBuild\s+)?:=\s*"([^"]+)"`)

func extractScalaVersion(idx *fileIndex) string {
	data, err := idx.readFileByName("build.sbt")
	if err != nil {
		return ""
	}
	m := scalaVersionRe.FindSubmatch(data)
	if len(m) >= 2 {
		return string(m[1])
	}
	return ""
}
