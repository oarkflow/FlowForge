package detector

import "strings"

// infrastructureDetectors returns detectors for infrastructure-as-code and
// deployment configuration files: Dockerfile, Docker Compose, Kubernetes
// manifests, Terraform, Nginx, and systemd units.
func infrastructureDetectors() []languageDetector {
	return []languageDetector{
		dockerfileDetector(),
		dockerComposeDetector(),
		kubernetesDetector(),
		terraformDetector(),
		nginxDetector(),
		systemdDetector(),
	}
}

func dockerfileDetector() languageDetector {
	return languageDetector{
		language: "Docker",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64

			if idx.hasFile("Dockerfile") {
				confidence += 0.6
			}
			// Also check for multi-stage / named Dockerfiles.
			for name := range idx.byName {
				lower := strings.ToLower(name)
				if strings.HasPrefix(lower, "dockerfile.") || strings.HasSuffix(lower, ".dockerfile") {
					confidence += 0.2
					break
				}
			}
			if idx.hasFile(".dockerignore") {
				confidence += 0.1
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Docker",
				DependencyFile: idx.firstPathForName("Dockerfile"),
				Confidence:     confidence,
				BuildTool:      "docker",
			}
		},
	}
}

func dockerComposeDetector() languageDetector {
	return languageDetector{
		language: "Docker Compose",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile string

			composeFiles := []string{
				"docker-compose.yml", "docker-compose.yaml",
				"compose.yml", "compose.yaml",
			}
			for _, f := range composeFiles {
				if idx.hasFile(f) {
					confidence += 0.7
					depFile = idx.firstPathForName(f)
					break
				}
			}
			// Check for variant files (docker-compose.dev.yml, etc.)
			for name := range idx.byName {
				lower := strings.ToLower(name)
				if strings.HasPrefix(lower, "docker-compose.") && (strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml")) {
					if depFile == "" {
						depFile = idx.firstPathForName(name)
					}
					if confidence == 0 {
						confidence = 0.5
					}
					break
				}
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Docker Compose",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      "docker-compose",
			}
		},
	}
}

func kubernetesDetector() languageDetector {
	return languageDetector{
		language: "Kubernetes",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile string

			// Check for common Kubernetes manifest directories / files.
			k8sIndicators := []string{
				"k8s", "kubernetes", "deploy", "manifests", "helm",
			}
			for _, dir := range k8sIndicators {
				for _, f := range idx.allFiles {
					if strings.HasPrefix(f, dir+"/") && (strings.HasSuffix(f, ".yml") || strings.HasSuffix(f, ".yaml")) {
						confidence += 0.3
						if depFile == "" {
							depFile = f
						}
						break
					}
				}
			}

			// Check for Chart.yaml (Helm).
			if idx.hasFile("Chart.yaml") {
				confidence += 0.5
				if depFile == "" {
					depFile = idx.firstPathForName("Chart.yaml")
				}
			}

			// Scan YAML files for Kubernetes API version markers.
			yamlFiles := append(idx.byExt[".yml"], idx.byExt[".yaml"]...)
			for _, f := range yamlFiles {
				data, err := idx.readFile(f)
				if err != nil {
					continue
				}
				content := string(data)
				if (strings.Contains(content, "apiVersion:") && strings.Contains(content, "kind:")) ||
					strings.Contains(content, "apps/v1") ||
					strings.Contains(content, "batch/v1") {
					confidence += 0.4
					if depFile == "" {
						depFile = f
					}
					break
				}
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Kubernetes",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      "kubectl",
			}
		},
	}
}

func terraformDetector() languageDetector {
	return languageDetector{
		language: "Terraform",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64

			tfCount := idx.countExt(".tf")
			if tfCount > 0 {
				confidence += 0.6
				c := float64(tfCount) * 0.05
				if c > 0.3 {
					c = 0.3
				}
				confidence += c
			}

			if idx.hasFile(".terraform.lock.hcl") {
				confidence += 0.2
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}

			var depFile string
			if paths := idx.byExt[".tf"]; len(paths) > 0 {
				depFile = paths[0]
			}

			return &DetectionResult{
				Language:       "Terraform",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      "terraform",
			}
		},
	}
}

func nginxDetector() languageDetector {
	return languageDetector{
		language: "Nginx",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile string

			// Check for nginx config files by name.
			nginxFiles := []string{"nginx.conf", "site.conf", "default.conf"}
			for _, f := range nginxFiles {
				if idx.hasFile(f) {
					confidence += 0.5
					if depFile == "" {
						depFile = idx.firstPathForName(f)
					}
					break
				}
			}

			// Check for nginx directory.
			for _, f := range idx.allFiles {
				if strings.Contains(f, "nginx/") && strings.HasSuffix(f, ".conf") {
					confidence += 0.3
					if depFile == "" {
						depFile = f
					}
					break
				}
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Nginx",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      "nginx",
			}
		},
	}
}

func systemdDetector() languageDetector {
	return languageDetector{
		language: "Systemd",
		detect: func(idx *fileIndex) *DetectionResult {
			var confidence float64
			var depFile string

			serviceCount := idx.countExt(".service")
			if serviceCount > 0 {
				confidence += 0.5
				paths := idx.byExt[".service"]
				if len(paths) > 0 {
					depFile = paths[0]
				}
			}

			timerCount := idx.countExt(".timer")
			if timerCount > 0 {
				confidence += 0.2
			}

			// Check for systemd directory.
			for _, f := range idx.allFiles {
				if strings.Contains(f, "systemd/") {
					confidence += 0.2
					break
				}
			}

			if confidence <= 0 {
				return nil
			}
			if confidence > 1.0 {
				confidence = 1.0
			}
			return &DetectionResult{
				Language:       "Systemd",
				DependencyFile: depFile,
				Confidence:     confidence,
				BuildTool:      "systemd",
			}
		},
	}
}
