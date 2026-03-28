package queries

import "github.com/jmoiron/sqlx"

type Repositories struct {
	Users                    *UserRepo
	Orgs                     *OrgRepo
	Projects                 *ProjectRepo
	Repos                    *RepositoryRepo
	Pipelines                *PipelineRepo
	Runs                     *RunRepo
	Logs                     *LogRepo
	Agents                   *AgentRepo
	Secrets                  *SecretRepo
	Artifacts                *ArtifactRepo
	Notifications            *NotificationRepo
	AuditLogs                *AuditLogRepo
	EnvVars                  *EnvVarRepo
	Environments             *EnvironmentRepo
	Deployments              *DeploymentRepo
	EnvOverrides             *EnvOverrideRepo
	Registries               *RegistryRepo
	Approvals                *ApprovalRepo
	ApprovalResponses        *ApprovalResponseRepo
	Schedules                *ScheduleRepo
	ScalingPolicies          *ScalingPolicyRepo
	ScalingEvents            *ScalingEventRepo
	PipelineLinks            *PipelineLinkRepo
	InAppNotifications       *InAppNotificationRepo
	NotificationPrefs        *NotificationPrefRepo
	DashboardPrefs           *DashboardPrefRepo
	Templates                *TemplateRepo
	DeadLetters              *DeadLetterRepo
	FeatureFlags             *FeatureFlagRepo
	ScanResults              *ScanResultRepo
	SecretProviders          *SecretProviderRepo
	IPAllowlist              *IPAllowlistRepo
	DeploymentProviders      *ProjectDeploymentProviderRepo
	EnvironmentChain         *ProjectEnvironmentChainRepo
	StageEnvironmentMappings *PipelineStageEnvironmentMappingRepo
}

func NewRepositories(db *sqlx.DB) *Repositories {
	return &Repositories{
		Users:                    &UserRepo{db: db},
		Orgs:                     &OrgRepo{db: db},
		Projects:                 &ProjectRepo{db: db},
		Repos:                    &RepositoryRepo{db: db},
		Pipelines:                &PipelineRepo{db: db},
		Runs:                     &RunRepo{db: db},
		Logs:                     &LogRepo{db: db},
		Agents:                   &AgentRepo{db: db},
		Secrets:                  &SecretRepo{db: db},
		Artifacts:                &ArtifactRepo{db: db},
		Notifications:            &NotificationRepo{db: db},
		AuditLogs:                &AuditLogRepo{db: db},
		EnvVars:                  NewEnvVarRepo(db),
		Environments:             &EnvironmentRepo{db: db},
		Deployments:              &DeploymentRepo{db: db},
		EnvOverrides:             &EnvOverrideRepo{db: db},
		Registries:               &RegistryRepo{db: db},
		Approvals:                &ApprovalRepo{db: db},
		ApprovalResponses:        &ApprovalResponseRepo{db: db},
		Schedules:                &ScheduleRepo{db: db},
		ScalingPolicies:          &ScalingPolicyRepo{db: db},
		ScalingEvents:            &ScalingEventRepo{db: db},
		PipelineLinks:            &PipelineLinkRepo{db: db},
		InAppNotifications:       &InAppNotificationRepo{db: db},
		NotificationPrefs:        &NotificationPrefRepo{db: db},
		DashboardPrefs:           &DashboardPrefRepo{db: db},
		Templates:                &TemplateRepo{db: db},
		DeadLetters:              &DeadLetterRepo{db: db},
		FeatureFlags:             &FeatureFlagRepo{db: db},
		ScanResults:              &ScanResultRepo{db: db},
		SecretProviders:          &SecretProviderRepo{db: db},
		IPAllowlist:              &IPAllowlistRepo{db: db},
		DeploymentProviders:      &ProjectDeploymentProviderRepo{db: db},
		EnvironmentChain:         &ProjectEnvironmentChainRepo{db: db},
		StageEnvironmentMappings: &PipelineStageEnvironmentMappingRepo{db: db},
	}
}
