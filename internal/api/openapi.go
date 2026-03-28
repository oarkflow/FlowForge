package api

import (
	"github.com/gofiber/fiber/v3"
)

// OpenAPISpec returns the complete OpenAPI 3.0 specification for the FlowForge API.
func OpenAPISpec() fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.JSON(openAPISpec)
	}
}

var openAPISpec = map[string]interface{}{
	"openapi": "3.0.3",
	"info": map[string]interface{}{
		"title":       "FlowForge CI/CD API",
		"description": "REST API for the FlowForge CI/CD platform",
		"version":     "1.0.0",
		"contact": map[string]string{
			"name": "FlowForge Team",
		},
		"license": map[string]string{
			"name": "MIT",
		},
	},
	"servers": []map[string]string{
		{"url": "/api/v1", "description": "API v1"},
	},
	"components": map[string]interface{}{
		"securitySchemes": map[string]interface{}{
			"bearerAuth": map[string]interface{}{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
				"description":  "JWT access token",
			},
			"apiKeyAuth": map[string]interface{}{
				"type": "apiKey",
				"in":   "header",
				"name": "Authorization",
			},
		},
		"schemas": map[string]interface{}{
			"Error": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"error": map[string]string{"type": "string"},
				},
			},
			"User": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":           map[string]string{"type": "string"},
					"email":        map[string]string{"type": "string", "format": "email"},
					"username":     map[string]string{"type": "string"},
					"display_name": map[string]string{"type": "string"},
					"role":         map[string]string{"type": "string"},
					"is_active":    map[string]string{"type": "integer"},
					"created_at":   map[string]string{"type": "string", "format": "date-time"},
				},
			},
			"Project": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":          map[string]string{"type": "string"},
					"name":        map[string]string{"type": "string"},
					"slug":        map[string]string{"type": "string"},
					"description": map[string]string{"type": "string"},
					"visibility":  map[string]string{"type": "string"},
					"created_at":  map[string]string{"type": "string", "format": "date-time"},
				},
			},
			"Pipeline": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":             map[string]string{"type": "string"},
					"project_id":     map[string]string{"type": "string"},
					"name":           map[string]string{"type": "string"},
					"config_content": map[string]string{"type": "string"},
					"is_active":      map[string]string{"type": "integer"},
					"created_at":     map[string]string{"type": "string", "format": "date-time"},
				},
			},
			"PipelineRun": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":           map[string]string{"type": "string"},
					"pipeline_id":  map[string]string{"type": "string"},
					"number":       map[string]string{"type": "integer"},
					"status":       map[string]string{"type": "string"},
					"trigger_type": map[string]string{"type": "string"},
					"branch":       map[string]string{"type": "string"},
					"started_at":   map[string]string{"type": "string", "format": "date-time"},
					"finished_at":  map[string]string{"type": "string", "format": "date-time"},
				},
			},
			"Agent": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":       map[string]string{"type": "string"},
					"name":     map[string]string{"type": "string"},
					"status":   map[string]string{"type": "string"},
					"executor": map[string]string{"type": "string"},
					"labels":   map[string]string{"type": "string"},
				},
			},
			"Secret": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":    map[string]string{"type": "string"},
					"key":   map[string]string{"type": "string"},
					"scope": map[string]string{"type": "string"},
				},
			},
			"Template": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":          map[string]string{"type": "string"},
					"name":        map[string]string{"type": "string"},
					"description": map[string]string{"type": "string"},
					"category":    map[string]string{"type": "string"},
					"config":      map[string]string{"type": "string"},
					"is_builtin":  map[string]string{"type": "integer"},
					"downloads":   map[string]string{"type": "integer"},
				},
			},
			"FeatureFlag": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":                  map[string]string{"type": "string"},
					"name":                map[string]string{"type": "string"},
					"enabled":             map[string]string{"type": "integer"},
					"rollout_percentage":  map[string]string{"type": "integer"},
				},
			},
			"ScanResult": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":             map[string]string{"type": "string"},
					"run_id":         map[string]string{"type": "string"},
					"scanner_type":   map[string]string{"type": "string"},
					"critical_count": map[string]string{"type": "integer"},
					"high_count":     map[string]string{"type": "integer"},
					"status":         map[string]string{"type": "string"},
				},
			},
			"DeadLetterItem": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":              map[string]string{"type": "string"},
					"job_id":          map[string]string{"type": "string"},
					"failure_reason":  map[string]string{"type": "string"},
					"retry_count":     map[string]string{"type": "integer"},
					"status":          map[string]string{"type": "string"},
				},
			},
			"BackupInfo": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":         map[string]string{"type": "string"},
					"filename":   map[string]string{"type": "string"},
					"size_bytes": map[string]string{"type": "integer"},
					"created_at": map[string]string{"type": "string", "format": "date-time"},
				},
			},
		},
	},
	"paths": generatePaths(),
}

func generatePaths() map[string]interface{} {
	return map[string]interface{}{
		"/auth/login":    pathItem("post", "Authentication", "Login with email and password", "LoginRequest", "LoginResponse"),
		"/auth/register": pathItem("post", "Authentication", "Register a new user", "RegisterRequest", "User"),
		"/auth/refresh":  pathItem("post", "Authentication", "Refresh access token", "RefreshRequest", "LoginResponse"),
		"/auth/logout":   pathItem("post", "Authentication", "Logout", nil, nil),
		"/users/me":      pathItem("get", "Users", "Get current user profile", nil, "User"),
		"/orgs":          pathItem("get", "Organizations", "List organizations", nil, nil),
		"/projects":      pathItem("get", "Projects", "List projects", nil, nil),
		"/projects/{id}": pathItem("get", "Projects", "Get project by ID", nil, "Project"),
		"/projects/{id}/pipelines":                    pathItem("get", "Pipelines", "List pipelines for a project", nil, nil),
		"/projects/{id}/pipelines/{pid}":              pathItem("get", "Pipelines", "Get pipeline by ID", nil, "Pipeline"),
		"/projects/{id}/pipelines/{pid}/trigger":      pathItem("post", "Pipelines", "Trigger a pipeline run", nil, "PipelineRun"),
		"/projects/{id}/pipelines/{pid}/runs":         pathItem("get", "Pipeline Runs", "List runs for a pipeline", nil, nil),
		"/projects/{id}/pipelines/{pid}/runs/{rid}":   pathItem("get", "Pipeline Runs", "Get run details", nil, "PipelineRun"),
		"/projects/{id}/pipelines/{pid}/runs/{rid}/cancel": pathItem("post", "Pipeline Runs", "Cancel a running pipeline", nil, nil),
		"/projects/{id}/pipelines/{pid}/runs/{rid}/logs":   pathItem("get", "Pipeline Runs", "Get run logs", nil, nil),
		"/projects/{id}/secrets":                pathItem("get", "Secrets", "List project secrets", nil, nil),
		"/projects/{id}/notifications":          pathItem("get", "Notifications", "List notification channels", nil, nil),
		"/projects/{id}/environments":           pathItem("get", "Environments", "List deployment environments", nil, nil),
		"/agents":                               pathItem("get", "Agents", "List build agents", nil, nil),
		"/agents/{id}":                          pathItem("get", "Agents", "Get agent by ID", nil, "Agent"),
		"/artifacts/{id}":                       pathItem("get", "Artifacts", "Get artifact metadata", nil, nil),
		"/artifacts/{id}/download":              pathItem("get", "Artifacts", "Download artifact", nil, nil),
		"/audit-logs":                           pathItem("get", "Audit", "List audit log entries", nil, nil),
		"/templates":                            pathItem("get", "Templates", "List pipeline templates", nil, nil),
		"/templates/{id}":                       pathItem("get", "Templates", "Get template by ID", nil, "Template"),
		"/runs/{rid}/security":                  pathItem("get", "Security", "Get security scan results for a run", nil, nil),
		"/admin/features":                       pathItem("get", "Admin", "List feature flags", nil, nil),
		"/admin/features/{id}":                  pathItem("put", "Admin", "Update feature flag", "FeatureFlag", "FeatureFlag"),
		"/admin/dlq":                            pathItem("get", "Admin", "List dead-letter queue items", nil, nil),
		"/admin/dlq/{id}/retry":                 pathItem("post", "Admin", "Retry a DLQ item", nil, nil),
		"/admin/dlq/{id}":                       pathItem("delete", "Admin", "Purge a DLQ item", nil, nil),
		"/admin/backup":                         pathItem("post", "Admin", "Create a database backup", nil, "BackupInfo"),
		"/admin/backups":                        pathItem("get", "Admin", "List available backups", nil, nil),
		"/admin/restore":                        pathItem("post", "Admin", "Restore from a backup", nil, nil),
		"/system/health":                        pathItem("get", "System", "Health check", nil, nil),
		"/system/metrics":                       pathItem("get", "System", "System metrics", nil, nil),
		"/system/info":                          pathItem("get", "System", "System information", nil, nil),
		"/openapi.json":                         pathItem("get", "System", "OpenAPI specification", nil, nil),
	}
}

func pathItem(method, tag, summary string, _ interface{}, _ interface{}) map[string]interface{} {
	op := map[string]interface{}{
		"tags":    []string{tag},
		"summary": summary,
		"responses": map[string]interface{}{
			"200": map[string]interface{}{
				"description": "Successful response",
			},
			"400": map[string]interface{}{
				"description": "Bad request",
			},
			"401": map[string]interface{}{
				"description": "Unauthorized",
			},
			"403": map[string]interface{}{
				"description": "Forbidden",
			},
			"404": map[string]interface{}{
				"description": "Not found",
			},
		},
	}

	// Add security requirement for non-auth endpoints
	if tag != "Authentication" && tag != "System" {
		op["security"] = []map[string]interface{}{
			{"bearerAuth": []string{}},
		}
	}

	return map[string]interface{}{
		method: op,
	}
}
