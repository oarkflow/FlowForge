package security

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BOM represents a CycloneDX Bill of Materials.
type BOM struct {
	BOMFormat    string       `json:"bomFormat"`
	SpecVersion  string       `json:"specVersion"`
	Version      int          `json:"version"`
	Metadata     BOMMetadata  `json:"metadata"`
	Components   []Component  `json:"components"`
}

// BOMMetadata contains metadata about the BOM generation.
type BOMMetadata struct {
	Timestamp string    `json:"timestamp"`
	Tools     []BOMTool `json:"tools"`
}

// BOMTool describes the tool that generated the BOM.
type BOMTool struct {
	Vendor  string `json:"vendor"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Component represents a single dependency in the BOM.
type Component struct {
	Type    string `json:"type"` // library, framework, application
	Name    string `json:"name"`
	Version string `json:"version"`
	Purl    string `json:"purl,omitempty"` // package URL
	Group   string `json:"group,omitempty"`
}

// GenerateSBOM scans the given directory for dependency files and produces
// a CycloneDX JSON BOM.
func GenerateSBOM(dir string) (*BOM, error) {
	bom := &BOM{
		BOMFormat:   "CycloneDX",
		SpecVersion: "1.5",
		Version:     1,
		Metadata: BOMMetadata{
			Timestamp: time.Now().Format(time.RFC3339),
			Tools: []BOMTool{
				{Vendor: "FlowForge", Name: "flowforge-sbom", Version: "1.0.0"},
			},
		},
	}

	// Parse go.mod
	gomod := filepath.Join(dir, "go.mod")
	if comps, err := parseGoMod(gomod); err == nil {
		bom.Components = append(bom.Components, comps...)
	}

	// Parse package.json
	pkgjson := filepath.Join(dir, "package.json")
	if comps, err := parsePackageJSON(pkgjson); err == nil {
		bom.Components = append(bom.Components, comps...)
	}

	// Parse requirements.txt
	reqtxt := filepath.Join(dir, "requirements.txt")
	if comps, err := parseRequirementsTxt(reqtxt); err == nil {
		bom.Components = append(bom.Components, comps...)
	}

	// Parse Cargo.toml
	cargo := filepath.Join(dir, "Cargo.toml")
	if comps, err := parseCargoToml(cargo); err == nil {
		bom.Components = append(bom.Components, comps...)
	}

	// Parse pom.xml (simplified)
	pom := filepath.Join(dir, "pom.xml")
	if comps, err := parsePomXml(pom); err == nil {
		bom.Components = append(bom.Components, comps...)
	}

	return bom, nil
}

// SBOMToJSON serializes the BOM to a JSON string.
func SBOMToJSON(bom *BOM) (string, error) {
	data, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseGoMod(path string) ([]Component, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var comps []Component
	inRequire := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "require (") || strings.HasPrefix(line, "require(") {
			inRequire = true
			continue
		}
		if line == ")" {
			inRequire = false
			continue
		}
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			// Single-line require
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				comps = append(comps, Component{
					Type:    "library",
					Name:    parts[1],
					Version: parts[2],
					Purl:    fmt.Sprintf("pkg:golang/%s@%s", parts[1], parts[2]),
				})
			}
			continue
		}
		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				comps = append(comps, Component{
					Type:    "library",
					Name:    parts[0],
					Version: parts[1],
					Purl:    fmt.Sprintf("pkg:golang/%s@%s", parts[0], parts[1]),
				})
			}
		}
	}
	return comps, nil
}

func parsePackageJSON(path string) ([]Component, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	var comps []Component
	for name, version := range pkg.Dependencies {
		comps = append(comps, Component{
			Type:    "library",
			Name:    name,
			Version: strings.TrimLeft(version, "^~>=<"),
			Purl:    fmt.Sprintf("pkg:npm/%s@%s", name, strings.TrimLeft(version, "^~>=<")),
		})
	}
	for name, version := range pkg.DevDependencies {
		comps = append(comps, Component{
			Type:    "library",
			Name:    name,
			Version: strings.TrimLeft(version, "^~>=<"),
			Purl:    fmt.Sprintf("pkg:npm/%s@%s", name, strings.TrimLeft(version, "^~>=<")),
		})
	}
	return comps, nil
}

func parseRequirementsTxt(path string) ([]Component, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var comps []Component
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		// Handle name==version, name>=version, name~=version
		var name, version string
		for _, sep := range []string{"==", ">=", "~=", "<=", "!="} {
			if idx := strings.Index(line, sep); idx != -1 {
				name = strings.TrimSpace(line[:idx])
				version = strings.TrimSpace(line[idx+len(sep):])
				break
			}
		}
		if name == "" {
			name = line
			version = "unknown"
		}
		comps = append(comps, Component{
			Type:    "library",
			Name:    name,
			Version: version,
			Purl:    fmt.Sprintf("pkg:pypi/%s@%s", name, version),
		})
	}
	return comps, nil
}

func parseCargoToml(path string) ([]Component, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var comps []Component
	inDeps := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[dependencies]" || line == "[dev-dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDeps = false
			continue
		}
		if inDeps && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			name := strings.TrimSpace(parts[0])
			version := strings.TrimSpace(parts[1])
			version = strings.Trim(version, `"'`)
			// Handle table format: name = { version = "x.y" }
			if strings.HasPrefix(version, "{") {
				if idx := strings.Index(version, "version"); idx != -1 {
					rest := version[idx:]
					if eqIdx := strings.Index(rest, "="); eqIdx != -1 {
						v := strings.TrimSpace(rest[eqIdx+1:])
						v = strings.Trim(v, `"' {},`)
						version = v
					}
				}
			}
			comps = append(comps, Component{
				Type:    "library",
				Name:    name,
				Version: version,
				Purl:    fmt.Sprintf("pkg:cargo/%s@%s", name, version),
			})
		}
	}
	return comps, nil
}

func parsePomXml(path string) ([]Component, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var comps []Component

	// Simple XML parsing for dependencies (not full XML parser)
	idx := 0
	for {
		start := strings.Index(content[idx:], "<dependency>")
		if start == -1 {
			break
		}
		end := strings.Index(content[idx+start:], "</dependency>")
		if end == -1 {
			break
		}
		dep := content[idx+start : idx+start+end+len("</dependency>")]
		idx = idx + start + end + len("</dependency>")

		groupID := extractXMLTag(dep, "groupId")
		artifactID := extractXMLTag(dep, "artifactId")
		version := extractXMLTag(dep, "version")

		if artifactID != "" {
			comps = append(comps, Component{
				Type:    "library",
				Group:   groupID,
				Name:    artifactID,
				Version: version,
				Purl:    fmt.Sprintf("pkg:maven/%s/%s@%s", groupID, artifactID, version),
			})
		}
	}
	return comps, nil
}

func extractXMLTag(xml, tag string) string {
	start := strings.Index(xml, "<"+tag+">")
	if start == -1 {
		return ""
	}
	start += len(tag) + 2
	end := strings.Index(xml[start:], "</"+tag+">")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(xml[start : start+end])
}
