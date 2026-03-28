package queries

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// setupTestDB creates an in-memory SQLite database with all required tables.
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
CREATE TABLE users (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    username    TEXT NOT NULL UNIQUE,
    password_hash TEXT,
    display_name TEXT,
    avatar_url  TEXT,
    role        TEXT NOT NULL DEFAULT 'viewer',
    totp_secret TEXT,
    totp_enabled INTEGER NOT NULL DEFAULT 0,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME
);

CREATE TABLE organizations (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    logo_url    TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE org_members (
    org_id      TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'developer',
    joined_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (org_id, user_id)
);

CREATE TABLE projects (
    id          TEXT PRIMARY KEY,
    org_id      TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    description TEXT,
    visibility  TEXT NOT NULL DEFAULT 'private',
    log_retention_days INTEGER NOT NULL DEFAULT 0,
    created_by  TEXT REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME,
    UNIQUE(org_id, slug)
);

CREATE TABLE repositories (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_id     TEXT NOT NULL,
    full_name       TEXT NOT NULL,
    clone_url       TEXT NOT NULL,
    ssh_url         TEXT,
    default_branch  TEXT NOT NULL DEFAULT 'main',
    webhook_id      TEXT,
    webhook_secret  TEXT,
    access_token_enc TEXT,
    ssh_key_enc     TEXT,
    is_active       INTEGER NOT NULL DEFAULT 1,
    last_sync_at    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pipelines (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repository_id   TEXT REFERENCES repositories(id),
    name            TEXT NOT NULL,
    description     TEXT,
    config_source   TEXT NOT NULL DEFAULT 'db',
    config_path     TEXT DEFAULT '.flowforge.yml',
    config_content  TEXT,
    config_version  INTEGER NOT NULL DEFAULT 1,
    triggers        TEXT NOT NULL DEFAULT '{}',
    is_active       INTEGER NOT NULL DEFAULT 1,
    path_filters    TEXT NOT NULL DEFAULT '',
    ignore_paths    TEXT NOT NULL DEFAULT '',
    created_by      TEXT REFERENCES users(id),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      DATETIME
);

CREATE TABLE pipeline_versions (
    id          TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL,
    config      TEXT NOT NULL,
    message     TEXT,
    created_by  TEXT REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pipeline_id, version)
);

CREATE TABLE pipeline_runs (
    id              TEXT PRIMARY KEY,
    pipeline_id     TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    number          INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'queued',
    trigger_type    TEXT NOT NULL,
    trigger_data    TEXT,
    commit_sha      TEXT,
    commit_message  TEXT,
    branch          TEXT,
    tag             TEXT,
    author          TEXT,
    started_at      DATETIME,
    finished_at     DATETIME,
    duration_ms     INTEGER,
    error_summary   TEXT,
    deploy_url      TEXT,
    created_by      TEXT REFERENCES users(id),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pipeline_id, number)
);

CREATE TABLE stage_runs (
    id          TEXT PRIMARY KEY,
    run_id      TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    position    INTEGER NOT NULL,
    started_at  DATETIME,
    finished_at DATETIME
);

CREATE TABLE job_runs (
    id              TEXT PRIMARY KEY,
    stage_run_id    TEXT NOT NULL REFERENCES stage_runs(id) ON DELETE CASCADE,
    run_id          TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    agent_id        TEXT,
    executor_type   TEXT NOT NULL DEFAULT 'local',
    started_at      DATETIME,
    finished_at     DATETIME
);

CREATE TABLE step_runs (
    id              TEXT PRIMARY KEY,
    job_run_id      TEXT NOT NULL REFERENCES job_runs(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    exit_code       INTEGER,
    error_message   TEXT,
    started_at      DATETIME,
    finished_at     DATETIME,
    duration_ms     INTEGER
);

CREATE TABLE run_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id      TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    step_run_id TEXT REFERENCES step_runs(id) ON DELETE CASCADE,
    stream      TEXT NOT NULL DEFAULT 'stdout',
    content     TEXT NOT NULL,
    ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agents (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,
    labels      TEXT NOT NULL DEFAULT '{}',
    executor    TEXT NOT NULL DEFAULT 'local',
    status      TEXT NOT NULL DEFAULT 'offline',
    version     TEXT,
    os          TEXT,
    arch        TEXT,
    cpu_cores   INTEGER,
    memory_mb   INTEGER,
    ip_address  TEXT,
    last_seen_at DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE secrets (
    id          TEXT PRIMARY KEY,
    project_id  TEXT REFERENCES projects(id) ON DELETE CASCADE,
    org_id      TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    scope       TEXT NOT NULL,
    key         TEXT NOT NULL,
    value_enc   TEXT NOT NULL,
    masked      INTEGER NOT NULL DEFAULT 1,
    created_by  TEXT REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rotation_interval TEXT,
    last_rotated_at DATETIME,
    provider_type TEXT NOT NULL DEFAULT 'local'
);

CREATE TABLE audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_id    TEXT REFERENCES users(id),
    actor_ip    TEXT,
    action      TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id TEXT,
    changes     TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE environments (
    id                      TEXT PRIMARY KEY,
    project_id              TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name                    TEXT NOT NULL,
    slug                    TEXT NOT NULL,
    description             TEXT NOT NULL DEFAULT '',
    url                     TEXT NOT NULL DEFAULT '',
    is_production           INTEGER NOT NULL DEFAULT 0,
    auto_deploy_branch      TEXT NOT NULL DEFAULT '',
    required_approvers      TEXT NOT NULL DEFAULT '[]',
    protection_rules        TEXT NOT NULL DEFAULT '{}',
    deploy_freeze           INTEGER NOT NULL DEFAULT 0,
    lock_owner_id           TEXT REFERENCES users(id),
    lock_reason             TEXT NOT NULL DEFAULT '',
    locked_at               DATETIME,
    current_deployment_id   TEXT,
    strategy                TEXT NOT NULL DEFAULT 'recreate',
    strategy_config         TEXT NOT NULL DEFAULT '{}',
    health_check_url        TEXT NOT NULL DEFAULT '',
    health_check_interval   INTEGER NOT NULL DEFAULT 10,
    health_check_timeout    INTEGER NOT NULL DEFAULT 30,
    health_check_retries    INTEGER NOT NULL DEFAULT 3,
    health_check_path       TEXT NOT NULL DEFAULT '/health',
    health_check_expected_status INTEGER NOT NULL DEFAULT 200,
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, slug)
);

CREATE TABLE deployments (
    id                    TEXT PRIMARY KEY,
    environment_id        TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    pipeline_run_id       TEXT REFERENCES pipeline_runs(id) ON DELETE SET NULL,
    version               TEXT NOT NULL,
    status                TEXT NOT NULL,
    commit_sha            TEXT NOT NULL DEFAULT '',
    image_tag             TEXT NOT NULL DEFAULT '',
    deployed_by           TEXT NOT NULL DEFAULT '',
    started_at            DATETIME,
    finished_at           DATETIME,
    health_check_status   TEXT NOT NULL DEFAULT 'unknown',
    rollback_from_id      TEXT REFERENCES deployments(id) ON DELETE SET NULL,
    metadata              TEXT NOT NULL DEFAULT '{}',
    strategy              TEXT NOT NULL DEFAULT 'recreate',
    canary_weight         INTEGER NOT NULL DEFAULT 0,
    health_check_results  TEXT NOT NULL DEFAULT '[]',
    strategy_state        TEXT NOT NULL DEFAULT '{}',
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE project_deployment_providers (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    provider_type TEXT NOT NULL,
    config_enc    TEXT NOT NULL,
    is_active     INTEGER NOT NULL DEFAULT 1,
    created_by    TEXT REFERENCES users(id),
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, name)
);

CREATE TABLE project_environment_chain (
    id                    TEXT PRIMARY KEY,
    project_id            TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    target_environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    position              INTEGER NOT NULL DEFAULT 0,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, source_environment_id, target_environment_id)
);

CREATE TABLE pipeline_stage_environment_mappings (
    id             TEXT PRIMARY KEY,
    project_id     TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    pipeline_id    TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    stage_name     TEXT NOT NULL,
    environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pipeline_id, stage_name)
);
`
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return db
}

func strPtr(s string) *string { return &s }

// --- UserRepo ---

func TestUserRepo_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := &UserRepo{db: db}
	ctx := context.Background()

	user := &models.User{
		Email:    "alice@example.com",
		Username: "alice",
		Role:     "developer",
		IsActive: 1,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	if user.ID == "" {
		t.Error("ID should be assigned")
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", got.Email)
	}
}

func TestUserRepo_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := &UserRepo{db: db}
	ctx := context.Background()

	user := &models.User{Email: "bob@test.com", Username: "bob", Role: "viewer", IsActive: 1}
	repo.Create(ctx, user)

	got, err := repo.GetByEmail(ctx, "bob@test.com")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("user should be found by email")
	}
	if got.Username != "bob" {
		t.Errorf("username = %q, want bob", got.Username)
	}
}

func TestUserRepo_GetByUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := &UserRepo{db: db}
	ctx := context.Background()

	user := &models.User{Email: "carol@test.com", Username: "carol", Role: "admin", IsActive: 1}
	repo.Create(ctx, user)

	got, err := repo.GetByUsername(ctx, "carol")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("user should be found by username")
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := &UserRepo{db: db}
	ctx := context.Background()

	got, err := repo.GetByID(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("should return nil for nonexistent user")
	}
}

func TestUserRepo_List(t *testing.T) {
	db := setupTestDB(t)
	repo := &UserRepo{db: db}
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		u := &models.User{
			Email:    fmt.Sprintf("user%d@test.com", i),
			Username: fmt.Sprintf("user%d", i),
			Role:     "viewer",
			IsActive: 1,
		}
		repo.Create(ctx, u)
	}

	users, err := repo.List(ctx, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 3 {
		t.Errorf("List(3,0) = %d users, want 3", len(users))
	}
}

func TestUserRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := &UserRepo{db: db}
	ctx := context.Background()

	user := &models.User{Email: "update@test.com", Username: "updateme", Role: "viewer", IsActive: 1}
	repo.Create(ctx, user)

	user.Role = "admin"
	user.DisplayName = strPtr("Updated User")
	if err := repo.Update(ctx, user); err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetByID(ctx, user.ID)
	if got.Role != "admin" {
		t.Errorf("role after update = %q, want admin", got.Role)
	}
}

func TestUserRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := &UserRepo{db: db}
	ctx := context.Background()

	user := &models.User{Email: "delete@test.com", Username: "deleteme", Role: "viewer", IsActive: 1}
	repo.Create(ctx, user)

	if err := repo.SoftDelete(ctx, user.ID); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("soft-deleted user should not be returned by GetByID")
	}
}

// --- OrgRepo ---

func TestOrgRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := &OrgRepo{db: db}
	ctx := context.Background()

	org := &models.Organization{Name: "TestOrg", Slug: "test-org"}
	if err := repo.Create(ctx, org); err != nil {
		t.Fatal(err)
	}
	if org.ID == "" {
		t.Error("ID should be assigned")
	}

	got, err := repo.GetByID(ctx, org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "TestOrg" {
		t.Errorf("name = %q, want TestOrg", got.Name)
	}

	org.Name = "UpdatedOrg"
	if err := repo.Update(ctx, org); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetByID(ctx, org.ID)
	if got.Name != "UpdatedOrg" {
		t.Errorf("name after update = %q, want UpdatedOrg", got.Name)
	}

	if err := repo.Delete(ctx, org.ID); err != nil {
		t.Fatal(err)
	}
}

func TestOrgRepo_ListMembers(t *testing.T) {
	db := setupTestDB(t)
	orgRepo := &OrgRepo{db: db}
	userRepo := &UserRepo{db: db}
	ctx := context.Background()

	user := &models.User{Email: "member@test.com", Username: "member1", Role: "developer", IsActive: 1}
	userRepo.Create(ctx, user)

	org := &models.Organization{Name: "MemberOrg", Slug: "member-org"}
	orgRepo.Create(ctx, org)

	member := &models.OrgMember{OrgID: org.ID, UserID: user.ID, Role: "developer"}
	if err := orgRepo.AddMember(ctx, member); err != nil {
		t.Fatal(err)
	}

	members, err := orgRepo.ListMembers(ctx, org.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("members = %d, want 1", len(members))
	}

	if err := orgRepo.RemoveMember(ctx, org.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	members, _ = orgRepo.ListMembers(ctx, org.ID)
	if len(members) != 0 {
		t.Error("members should be 0 after removal")
	}
}

// --- ProjectRepo ---

func TestProjectRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	orgRepo := &OrgRepo{db: db}
	projRepo := &ProjectRepo{db: db}
	ctx := context.Background()

	org := &models.Organization{Name: "ProjOrg", Slug: "proj-org"}
	orgRepo.Create(ctx, org)

	project := &models.Project{
		OrgID:      &org.ID,
		Name:       "TestProject",
		Slug:       "test-project",
		Visibility: "private",
	}
	if err := projRepo.Create(ctx, project); err != nil {
		t.Fatal(err)
	}

	got, err := projRepo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "TestProject" {
		t.Errorf("name = %q, want TestProject", got.Name)
	}

	project.Name = "UpdatedProject"
	if err := projRepo.Update(ctx, project); err != nil {
		t.Fatal(err)
	}
	got, _ = projRepo.GetByID(ctx, project.ID)
	if got.Name != "UpdatedProject" {
		t.Errorf("name after update = %q", got.Name)
	}
}

func TestProjectRepo_ListByOrg(t *testing.T) {
	db := setupTestDB(t)
	orgRepo := &OrgRepo{db: db}
	projRepo := &ProjectRepo{db: db}
	ctx := context.Background()

	org := &models.Organization{Name: "ListOrg", Slug: "list-org"}
	orgRepo.Create(ctx, org)

	for i := 0; i < 3; i++ {
		p := &models.Project{
			OrgID:      &org.ID,
			Name:       fmt.Sprintf("proj%d", i),
			Slug:       fmt.Sprintf("proj-%d", i),
			Visibility: "private",
		}
		projRepo.Create(ctx, p)
	}

	projects, err := projRepo.ListByOrg(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 3 {
		t.Errorf("ListByOrg = %d projects, want 3", len(projects))
	}
}

func TestProjectRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	orgRepo := &OrgRepo{db: db}
	projRepo := &ProjectRepo{db: db}
	ctx := context.Background()

	org := &models.Organization{Name: "DelOrg", Slug: "del-org"}
	orgRepo.Create(ctx, org)

	project := &models.Project{OrgID: &org.ID, Name: "ToDelete", Slug: "to-delete", Visibility: "private"}
	projRepo.Create(ctx, project)

	if err := projRepo.SoftDelete(ctx, project.ID); err != nil {
		t.Fatal(err)
	}

	projects, _ := projRepo.List(ctx, 10, 0)
	for _, p := range projects {
		if p.ID == project.ID {
			t.Error("soft-deleted project should not appear in List")
		}
	}
}

// --- PipelineRepo ---

func createTestProjectAndOrg(t *testing.T, db *sqlx.DB) (*models.Organization, *models.Project) {
	t.Helper()
	ctx := context.Background()
	orgRepo := &OrgRepo{db: db}
	projRepo := &ProjectRepo{db: db}

	org := &models.Organization{Name: "PipeOrg", Slug: fmt.Sprintf("pipe-org-%d", time.Now().UnixNano())}
	orgRepo.Create(ctx, org)

	proj := &models.Project{OrgID: &org.ID, Name: "PipeProj", Slug: fmt.Sprintf("pipe-proj-%d", time.Now().UnixNano()), Visibility: "private"}
	projRepo.Create(ctx, proj)

	return org, proj
}

func TestPipelineRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := &PipelineRepo{db: db}
	ctx := context.Background()
	_, proj := createTestProjectAndOrg(t, db)

	pipeline := &models.Pipeline{
		ProjectID:    proj.ID,
		Name:         "Build Pipeline",
		ConfigSource: "db",
		Triggers:     "{}",
		IsActive:     1,
	}
	if err := repo.Create(ctx, pipeline); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID(ctx, pipeline.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Build Pipeline" {
		t.Errorf("name = %q", got.Name)
	}

	pipeline.Name = "Updated Pipeline"
	if err := repo.Update(ctx, pipeline); err != nil {
		t.Fatal(err)
	}
}

func TestPipelineRepo_ListByProject(t *testing.T) {
	db := setupTestDB(t)
	repo := &PipelineRepo{db: db}
	ctx := context.Background()
	_, proj := createTestProjectAndOrg(t, db)

	for i := 0; i < 3; i++ {
		p := &models.Pipeline{
			ProjectID:    proj.ID,
			Name:         fmt.Sprintf("pipeline-%d", i),
			ConfigSource: "db",
			Triggers:     "{}",
			IsActive:     1,
		}
		repo.Create(ctx, p)
	}

	pipelines, err := repo.ListByProject(ctx, proj.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pipelines) != 3 {
		t.Errorf("ListByProject = %d, want 3", len(pipelines))
	}
}

func TestPipelineRepo_Versioning(t *testing.T) {
	db := setupTestDB(t)
	repo := &PipelineRepo{db: db}
	ctx := context.Background()
	_, proj := createTestProjectAndOrg(t, db)

	pipeline := &models.Pipeline{ProjectID: proj.ID, Name: "Versioned", ConfigSource: "db", Triggers: "{}", IsActive: 1}
	repo.Create(ctx, pipeline)

	for i := 1; i <= 3; i++ {
		v := &models.PipelineVersion{
			PipelineID: pipeline.ID,
			Version:    i,
			Config:     fmt.Sprintf("version %d config", i),
		}
		if err := repo.CreateVersion(ctx, v); err != nil {
			t.Fatal(err)
		}
	}

	versions, err := repo.ListVersions(ctx, pipeline.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 3 {
		t.Errorf("versions = %d, want 3", len(versions))
	}
	// Should be ordered descending
	if versions[0].Version != 3 {
		t.Errorf("first version = %d, want 3", versions[0].Version)
	}
}

// --- RunRepo ---

func createTestPipeline(t *testing.T, db *sqlx.DB) *models.Pipeline {
	t.Helper()
	ctx := context.Background()
	_, proj := createTestProjectAndOrg(t, db)
	repo := &PipelineRepo{db: db}
	p := &models.Pipeline{ProjectID: proj.ID, Name: "RunPipeline", ConfigSource: "db", Triggers: "{}", IsActive: 1}
	repo.Create(ctx, p)
	return p
}

func TestRunRepo_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	runRepo := &RunRepo{db: db}
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)

	run := &models.PipelineRun{
		PipelineID:  pipeline.ID,
		Number:      1,
		Status:      "queued",
		TriggerType: "push",
		Branch:      strPtr("main"),
	}
	if err := runRepo.Create(ctx, run); err != nil {
		t.Fatal(err)
	}

	got, err := runRepo.GetByID(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "queued" {
		t.Errorf("status = %q, want queued", got.Status)
	}
}

func TestRunRepo_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	runRepo := &RunRepo{db: db}
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)

	run := &models.PipelineRun{PipelineID: pipeline.ID, Number: 1, Status: "queued", TriggerType: "push"}
	runRepo.Create(ctx, run)

	if err := runRepo.UpdateStatus(ctx, run.ID, "running"); err != nil {
		t.Fatal(err)
	}
	got, _ := runRepo.GetByID(ctx, run.ID)
	if got.Status != "running" {
		t.Errorf("status = %q, want running", got.Status)
	}
}

func TestRunRepo_SetStartedAndFinished(t *testing.T) {
	db := setupTestDB(t)
	runRepo := &RunRepo{db: db}
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)

	run := &models.PipelineRun{PipelineID: pipeline.ID, Number: 1, Status: "queued", TriggerType: "push"}
	runRepo.Create(ctx, run)

	if err := runRepo.SetStarted(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := runRepo.GetByID(ctx, run.ID)
	if got.Status != "running" {
		t.Errorf("status after SetStarted = %q", got.Status)
	}
	if got.StartedAt == nil {
		t.Error("started_at should be set")
	}

	if err := runRepo.SetFinished(ctx, run.ID, "success", 1500, ""); err != nil {
		t.Fatal(err)
	}
	got, _ = runRepo.GetByID(ctx, run.ID)
	if got.Status != "success" {
		t.Errorf("status after SetFinished = %q", got.Status)
	}
	if got.DurationMs == nil || *got.DurationMs != 1500 {
		t.Error("duration_ms should be 1500")
	}
}

func TestRunRepo_SetFinished_WithError(t *testing.T) {
	db := setupTestDB(t)
	runRepo := &RunRepo{db: db}
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)

	run := &models.PipelineRun{PipelineID: pipeline.ID, Number: 1, Status: "running", TriggerType: "push"}
	runRepo.Create(ctx, run)

	if err := runRepo.SetFinished(ctx, run.ID, "failure", 500, "step 3 failed"); err != nil {
		t.Fatal(err)
	}
	got, _ := runRepo.GetByID(ctx, run.ID)
	if got.ErrorSummary == nil || *got.ErrorSummary != "step 3 failed" {
		t.Error("error_summary should be set")
	}
}

func TestRunRepo_GetNextNumber(t *testing.T) {
	db := setupTestDB(t)
	runRepo := &RunRepo{db: db}
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)

	n, err := runRepo.GetNextNumber(ctx, pipeline.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("first number = %d, want 1", n)
	}

	run := &models.PipelineRun{PipelineID: pipeline.ID, Number: 1, Status: "queued", TriggerType: "push"}
	runRepo.Create(ctx, run)

	n, _ = runRepo.GetNextNumber(ctx, pipeline.ID)
	if n != 2 {
		t.Errorf("second number = %d, want 2", n)
	}
}

func TestRunRepo_SetDeployURL(t *testing.T) {
	db := setupTestDB(t)
	runRepo := &RunRepo{db: db}
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)

	run := &models.PipelineRun{PipelineID: pipeline.ID, Number: 1, Status: "queued", TriggerType: "push"}
	runRepo.Create(ctx, run)

	if err := runRepo.SetDeployURL(ctx, run.ID, "https://app.example.com"); err != nil {
		t.Fatal(err)
	}
	got, _ := runRepo.GetByID(ctx, run.ID)
	if got.DeployURL == nil || *got.DeployURL != "https://app.example.com" {
		t.Error("deploy_url should be set")
	}
}

func TestRunRepo_StageRuns(t *testing.T) {
	db := setupTestDB(t)
	runRepo := &RunRepo{db: db}
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)

	run := &models.PipelineRun{PipelineID: pipeline.ID, Number: 1, Status: "running", TriggerType: "push"}
	runRepo.Create(ctx, run)

	stage := &models.StageRun{RunID: run.ID, Name: "build", Status: "pending", Position: 1}
	if err := runRepo.CreateStageRun(ctx, stage); err != nil {
		t.Fatal(err)
	}

	stages, err := runRepo.ListStageRuns(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stages) != 1 {
		t.Fatalf("stages = %d, want 1", len(stages))
	}
	if stages[0].Name != "build" {
		t.Errorf("stage name = %q", stages[0].Name)
	}
}

func TestRunRepo_JobAndStepRuns(t *testing.T) {
	db := setupTestDB(t)
	runRepo := &RunRepo{db: db}
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)

	run := &models.PipelineRun{PipelineID: pipeline.ID, Number: 1, Status: "running", TriggerType: "push"}
	runRepo.Create(ctx, run)

	stage := &models.StageRun{RunID: run.ID, Name: "test", Status: "running", Position: 0}
	runRepo.CreateStageRun(ctx, stage)

	job := &models.JobRun{StageRunID: stage.ID, RunID: run.ID, Name: "unit-tests", Status: "pending", ExecutorType: "local"}
	if err := runRepo.CreateJobRun(ctx, job); err != nil {
		t.Fatal(err)
	}

	jobs, _ := runRepo.ListJobRuns(ctx, stage.ID)
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(jobs))
	}

	step := &models.StepRun{JobRunID: job.ID, Name: "go test", Status: "pending"}
	if err := runRepo.CreateStepRun(ctx, step); err != nil {
		t.Fatal(err)
	}

	steps, _ := runRepo.ListStepRuns(ctx, job.ID)
	if len(steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(steps))
	}
}

func TestRunRepo_ListByPipeline(t *testing.T) {
	db := setupTestDB(t)
	runRepo := &RunRepo{db: db}
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)

	for i := 1; i <= 5; i++ {
		run := &models.PipelineRun{PipelineID: pipeline.ID, Number: i, Status: "success", TriggerType: "push"}
		runRepo.Create(ctx, run)
	}

	runs, err := runRepo.ListByPipeline(ctx, pipeline.ID, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 {
		t.Errorf("ListByPipeline(3,0) = %d runs, want 3", len(runs))
	}
}

// --- AgentRepo ---

func TestAgentRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := &AgentRepo{db: db}
	ctx := context.Background()

	agent := &models.Agent{
		Name:      "agent-1",
		TokenHash: "hash123",
		Labels:    `["linux","docker"]`,
		Executor:  "docker",
		Status:    "online",
	}
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID(ctx, agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "agent-1" {
		t.Errorf("name = %q", got.Name)
	}

	if err := repo.UpdateStatus(ctx, agent.ID, "busy"); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetByID(ctx, agent.ID)
	if got.Status != "busy" {
		t.Errorf("status = %q, want busy", got.Status)
	}

	if err := repo.Delete(ctx, agent.ID); err != nil {
		t.Fatal(err)
	}
}

func TestAgentRepo_ListByStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := &AgentRepo{db: db}
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		a := &models.Agent{
			Name:      fmt.Sprintf("agent-%d", i),
			TokenHash: fmt.Sprintf("hash%d", i),
			Labels:    "{}",
			Executor:  "local",
			Status:    "online",
		}
		repo.Create(ctx, a)
	}
	offline := &models.Agent{Name: "agent-off", TokenHash: "hashoff", Labels: "{}", Executor: "local", Status: "offline"}
	repo.Create(ctx, offline)

	online, err := repo.ListByStatus(ctx, "online")
	if err != nil {
		t.Fatal(err)
	}
	if len(online) != 3 {
		t.Errorf("online agents = %d, want 3", len(online))
	}
}

// --- SecretRepo ---

func TestSecretRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := &SecretRepo{db: db}
	ctx := context.Background()

	secret := &models.Secret{
		Scope:        "global",
		Key:          "API_KEY",
		ValueEnc:     "encrypted-value",
		Masked:       1,
		ProviderType: "local",
	}
	if err := repo.Create(ctx, secret); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID(ctx, secret.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "API_KEY" {
		t.Errorf("key = %q", got.Key)
	}

	secret.ValueEnc = "new-encrypted"
	if err := repo.Update(ctx, secret); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetByID(ctx, secret.ID)
	if got.ValueEnc != "new-encrypted" {
		t.Error("value_enc should be updated")
	}

	if err := repo.Delete(ctx, secret.ID); err != nil {
		t.Fatal(err)
	}
}

// --- LogRepo ---

func createTestRun(t *testing.T, db *sqlx.DB) *models.PipelineRun {
	t.Helper()
	ctx := context.Background()
	pipeline := createTestPipeline(t, db)
	runRepo := &RunRepo{db: db}
	run := &models.PipelineRun{PipelineID: pipeline.ID, Number: 1, Status: "running", TriggerType: "push"}
	runRepo.Create(ctx, run)
	return run
}

func TestLogRepo_InsertAndGet(t *testing.T) {
	db := setupTestDB(t)
	logRepo := &LogRepo{db: db}
	ctx := context.Background()
	run := createTestRun(t, db)

	log := &models.RunLog{RunID: run.ID, Stream: "stdout", Content: "Building..."}
	if err := logRepo.Insert(ctx, log); err != nil {
		t.Fatal(err)
	}

	logs, err := logRepo.GetByRunID(ctx, run.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(logs))
	}
	if logs[0].Content != "Building..." {
		t.Errorf("content = %q", logs[0].Content)
	}
}

// --- AuditLogRepo ---

func TestAuditLogRepo_InsertAndList(t *testing.T) {
	db := setupTestDB(t)
	repo := &AuditLogRepo{db: db}
	ctx := context.Background()

	log := &models.AuditLog{
		Action:   "create",
		Resource: "project",
	}
	if err := repo.Insert(ctx, log); err != nil {
		t.Fatal(err)
	}

	logs, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(logs))
	}
	if logs[0].Action != "create" {
		t.Errorf("action = %q", logs[0].Action)
	}
}

func TestAuditLogRepo_ListByActor(t *testing.T) {
	db := setupTestDB(t)
	repo := &AuditLogRepo{db: db}
	userRepo := &UserRepo{db: db}
	ctx := context.Background()

	user := &models.User{Email: "audit@test.com", Username: "auditor", Role: "admin", IsActive: 1}
	userRepo.Create(ctx, user)

	for i := 0; i < 3; i++ {
		log := &models.AuditLog{ActorID: &user.ID, Action: fmt.Sprintf("action-%d", i), Resource: "project"}
		repo.Insert(ctx, log)
	}

	logs, err := repo.ListByActor(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 3 {
		t.Errorf("logs = %d, want 3", len(logs))
	}
}

// --- NewRepositories ---

func TestNewRepositories(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepositories(db)

	if repos.Users == nil {
		t.Error("Users repo should not be nil")
	}
	if repos.Projects == nil {
		t.Error("Projects repo should not be nil")
	}
	if repos.Pipelines == nil {
		t.Error("Pipelines repo should not be nil")
	}
	if repos.Runs == nil {
		t.Error("Runs repo should not be nil")
	}
	if repos.Agents == nil {
		t.Error("Agents repo should not be nil")
	}
	if repos.Secrets == nil {
		t.Error("Secrets repo should not be nil")
	}
	if repos.AuditLogs == nil {
		t.Error("AuditLogs repo should not be nil")
	}
}

func TestProjectDeploymentProviderRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := &ProjectDeploymentProviderRepo{db: db}
	ctx := context.Background()

	_, proj := createTestProjectAndOrg(t, db)
	provider := &models.ProjectDeploymentProvider{
		ProjectID:    proj.ID,
		Name:         "aws-primary",
		ProviderType: "aws",
		ConfigEnc:    "encrypted-json",
		IsActive:     1,
	}
	if err := repo.Create(ctx, provider); err != nil {
		t.Fatal(err)
	}
	if provider.ID == "" {
		t.Fatal("expected provider ID")
	}

	got, err := repo.GetByID(ctx, provider.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != provider.Name {
		t.Fatalf("name=%s want=%s", got.Name, provider.Name)
	}

	provider.Name = "aws-secondary"
	provider.ConfigEnc = "encrypted-json-2"
	if err := repo.Update(ctx, provider); err != nil {
		t.Fatal(err)
	}

	byName, err := repo.GetByName(ctx, proj.ID, "aws-secondary")
	if err != nil {
		t.Fatal(err)
	}
	if byName.ID != provider.ID {
		t.Fatalf("lookup by name returned wrong provider")
	}

	list, err := repo.ListByProject(ctx, proj.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list)=%d want=1", len(list))
	}

	if err := repo.Delete(ctx, provider.ID); err != nil {
		t.Fatal(err)
	}
}

func TestProjectEnvironmentChainRepo_ReplaceAndCheck(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	repo := &ProjectEnvironmentChainRepo{db: db}
	envRepo := &EnvironmentRepo{db: db}
	_, proj := createTestProjectAndOrg(t, db)

	beta := &models.Environment{ProjectID: proj.ID, Name: "Beta", Slug: "beta", RequiredApprovers: "[]", ProtectionRules: "{}", StrategyConfig: "{}"}
	stage := &models.Environment{ProjectID: proj.ID, Name: "Stage", Slug: "stage", RequiredApprovers: "[]", ProtectionRules: "{}", StrategyConfig: "{}"}
	prod := &models.Environment{ProjectID: proj.ID, Name: "Prod", Slug: "prod", RequiredApprovers: "[]", ProtectionRules: "{}", StrategyConfig: "{}"}
	if err := envRepo.Create(ctx, beta); err != nil {
		t.Fatal(err)
	}
	if err := envRepo.Create(ctx, stage); err != nil {
		t.Fatal(err)
	}
	if err := envRepo.Create(ctx, prod); err != nil {
		t.Fatal(err)
	}

	edges := []models.ProjectEnvironmentChainEdge{
		{SourceEnvironmentID: beta.ID, TargetEnvironmentID: stage.ID, Position: 0},
		{SourceEnvironmentID: stage.ID, TargetEnvironmentID: prod.ID, Position: 1},
	}
	if err := repo.ReplaceForProject(ctx, proj.ID, edges); err != nil {
		t.Fatal(err)
	}

	listed, err := repo.ListByProject(ctx, proj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("len(listed)=%d want=2", len(listed))
	}

	allowed, err := repo.IsPromotionAllowed(ctx, proj.ID, beta.ID, stage.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected beta->stage allowed")
	}

	notAllowed, err := repo.IsPromotionAllowed(ctx, proj.ID, beta.ID, prod.ID)
	if err != nil {
		t.Fatal(err)
	}
	if notAllowed {
		t.Fatal("expected beta->prod not allowed")
	}
}

func TestPipelineStageEnvironmentMappingRepo_ReplaceAndList(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	mappingRepo := &PipelineStageEnvironmentMappingRepo{db: db}
	envRepo := &EnvironmentRepo{db: db}
	pipelineRepo := &PipelineRepo{db: db}
	_, proj := createTestProjectAndOrg(t, db)

	env := &models.Environment{ProjectID: proj.ID, Name: "Stage", Slug: "stage", RequiredApprovers: "[]", ProtectionRules: "{}", StrategyConfig: "{}"}
	if err := envRepo.Create(ctx, env); err != nil {
		t.Fatal(err)
	}

	pipeline := &models.Pipeline{ProjectID: proj.ID, Name: "Deploy", ConfigSource: "db", Triggers: "{}", IsActive: 1}
	if err := pipelineRepo.Create(ctx, pipeline); err != nil {
		t.Fatal(err)
	}

	mappings := []models.PipelineStageEnvironmentMapping{
		{StageName: "deploy", EnvironmentID: env.ID},
	}
	if err := mappingRepo.ReplaceForPipeline(ctx, proj.ID, pipeline.ID, mappings); err != nil {
		t.Fatal(err)
	}

	listed, err := mappingRepo.ListByPipeline(ctx, proj.ID, pipeline.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("len(listed)=%d want=1", len(listed))
	}
	if listed[0].StageName != "deploy" {
		t.Fatalf("stage_name=%s want=deploy", listed[0].StageName)
	}
}
