package queries

import "github.com/jmoiron/sqlx"

type Repositories struct {
	Users         *UserRepo
	Orgs          *OrgRepo
	Projects      *ProjectRepo
	Repos         *RepositoryRepo
	Pipelines     *PipelineRepo
	Runs          *RunRepo
	Logs          *LogRepo
	Agents        *AgentRepo
	Secrets       *SecretRepo
	Artifacts     *ArtifactRepo
	Notifications *NotificationRepo
	AuditLogs     *AuditLogRepo
	EnvVars       *EnvVarRepo
}

func NewRepositories(db *sqlx.DB) *Repositories {
	return &Repositories{
		Users:         &UserRepo{db: db},
		Orgs:          &OrgRepo{db: db},
		Projects:      &ProjectRepo{db: db},
		Repos:         &RepositoryRepo{db: db},
		Pipelines:     &PipelineRepo{db: db},
		Runs:          &RunRepo{db: db},
		Logs:          &LogRepo{db: db},
		Agents:        &AgentRepo{db: db},
		Secrets:       &SecretRepo{db: db},
		Artifacts:     &ArtifactRepo{db: db},
		Notifications: &NotificationRepo{db: db},
		AuditLogs:     &AuditLogRepo{db: db},
		EnvVars:       NewEnvVarRepo(db),
	}
}
