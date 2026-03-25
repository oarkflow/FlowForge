import type { Component } from 'solid-js';
import { createStore } from 'solid-js/store';
import { Show, Switch, Match } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { toast } from '../ui/Toast';
import { api, ApiRequestError } from '../../api/client';
import type {
  DetectionResult, ProviderRepo,
  ImportCreateProjectRequest,
} from '../../types';

import StepSourceType from './StepSourceType';
import StepSourceDetails from './StepSourceDetails';
import StepDetectionResults from './StepDetectionResults';
import StepProjectCreate from './StepProjectCreate';

export type SourceType = 'git' | 'github' | 'gitlab' | 'bitbucket' | 'local' | 'upload' | null;

export interface WizardState {
  step: 1 | 2 | 3 | 4;
  sourceType: SourceType;
  gitUrl: string;
  branch: string;
  sshKey: string;
  selectedRepo: ProviderRepo | null;
  localPath: string;
  uploadId: string;
  uploadFilename: string;
  providerToken: string;
  sessionId: string;
  detections: DetectionResult[];
  generatedPipeline: string;
  editedPipeline: string;
  defaultBranch: string;
  cloneUrl: string;
  projectName: string;
  projectSlug: string;
  projectDescription: string;
  visibility: 'private' | 'internal' | 'public';
  orgId: string;
  setupWebhook: boolean;
  loading: boolean;
  error: string;
}

const initialState: WizardState = {
  step: 1,
  sourceType: null,
  gitUrl: '',
  branch: '',
  sshKey: '',
  selectedRepo: null,
  localPath: '',
  uploadId: '',
  uploadFilename: '',
  providerToken: '',
  sessionId: '',
  detections: [],
  generatedPipeline: '',
  editedPipeline: '',
  defaultBranch: '',
  cloneUrl: '',
  projectName: '',
  projectSlug: '',
  projectDescription: '',
  visibility: 'private',
  orgId: '',
  setupWebhook: true,
  loading: false,
  error: '',
};

const ImportWizard: Component = () => {
  const [state, setState] = createStore<WizardState>({ ...initialState });
  const navigate = useNavigate();

  const setStep = (step: WizardState['step']) => setState('step', step);
  const setError = (error: string) => setState('error', error);
  const setLoading = (loading: boolean) => setState('loading', loading);

  const goBack = () => {
    if (state.step > 1) {
      setStep((state.step - 1) as WizardState['step']);
      setError('');
    }
  };

  const selectSource = (type: SourceType) => {
    setState({ sourceType: type, error: '' });
    setStep(2);
  };

  const runDetection = async () => {
    setLoading(true);
    setError('');
    try {
      const detectReq: Record<string, string | undefined> = {
        source_type: state.sourceType!,
      };

      if (state.sourceType === 'git') {
        detectReq.git_url = state.gitUrl;
        detectReq.branch = state.branch || undefined;
        detectReq.ssh_key = state.sshKey || undefined;
      } else if (state.sourceType === 'github' || state.sourceType === 'gitlab' || state.sourceType === 'bitbucket') {
        if (!state.selectedRepo) {
          setError('Please select a repository');
          setLoading(false);
          return;
        }
        detectReq.source_type = state.sourceType;
        const parts = state.selectedRepo.full_name.split('/');
        detectReq.repo_owner = parts[0];
        detectReq.repo_name = parts.slice(1).join('/');
      } else if (state.sourceType === 'local') {
        detectReq.local_path = state.localPath;
      } else if (state.sourceType === 'upload') {
        detectReq.upload_id = state.uploadId;
      }

      const result = await api.import.detect(
        detectReq as any,
        state.providerToken || undefined,
      );

      setState({
        sessionId: result.session_id,
        detections: result.detections,
        generatedPipeline: result.generated_pipeline,
        editedPipeline: result.generated_pipeline,
        defaultBranch: result.default_branch || state.defaultBranch,
        cloneUrl: result.clone_url || state.cloneUrl,
      });

      // Auto-populate project name from repo name.
      if (!state.projectName) {
        let name = '';
        if (state.selectedRepo) {
          name = state.selectedRepo.full_name.split('/').pop() || '';
        } else if (state.gitUrl) {
          const match = state.gitUrl.match(/\/([^/]+?)(?:\.git)?$/);
          if (match) name = match[1];
        }
        if (name) {
          setState({
            projectName: name.replace(/[-_]/g, ' ').replace(/\b\w/g, c => c.toUpperCase()),
            projectSlug: name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, ''),
          });
        }
      }

      setStep(3);
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Detection failed';
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  const proceedToCreate = () => {
    setStep(4);
    setError('');
  };

  const createProject = async () => {
    setLoading(true);
    setError('');
    try {
      const slug = state.projectSlug ||
        state.projectName.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');

      const req: ImportCreateProjectRequest = {
        session_id: state.sessionId,
        project: {
          name: state.projectName,
          slug,
          description: state.projectDescription,
          visibility: state.visibility,
          org_id: state.orgId || undefined,
        },
        repository: {
          provider: state.sourceType === 'git' ? 'git' : (state.sourceType || 'git'),
          provider_id: state.selectedRepo?.id || '',
          full_name: state.selectedRepo?.full_name || state.gitUrl || state.localPath || state.uploadFilename || '',
          clone_url: state.cloneUrl || state.gitUrl || '',
          ssh_url: state.selectedRepo?.ssh_url || '',
          default_branch: state.defaultBranch || 'main',
        },
        pipeline_yaml: state.editedPipeline,
        setup_webhook: state.setupWebhook,
      };

      const result = await api.import.createProject(
        req,
        state.providerToken || undefined,
      );

      toast.success(`Project "${result.project.name}" created successfully`);
      navigate(`/projects/${result.project.id}`);
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to create project';
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  const stepLabels = ['Source', 'Details', 'Detection', 'Create'];

  return (
    <div class="max-w-4xl mx-auto">
      {/* Progress bar */}
      <div class="mb-8">
        <div class="flex items-center justify-between mb-2">
          {stepLabels.map((label, i) => (
            <div class="flex items-center">
              <div
                class={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors ${
                  i + 1 <= state.step
                    ? 'bg-indigo-600 text-white'
                    : 'bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)]'
                }`}
              >
                {i + 1}
              </div>
              <span
                class={`ml-2 text-sm ${
                  i + 1 <= state.step
                    ? 'text-[var(--color-text-primary)] font-medium'
                    : 'text-[var(--color-text-tertiary)]'
                }`}
              >
                {label}
              </span>
              <Show when={i < stepLabels.length - 1}>
                <div
                  class={`mx-4 h-px flex-1 min-w-[2rem] ${
                    i + 1 < state.step ? 'bg-indigo-600' : 'bg-[var(--color-border-primary)]'
                  }`}
                />
              </Show>
            </div>
          ))}
        </div>
      </div>

      {/* Error banner */}
      <Show when={state.error}>
        <div class="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30">
          <p class="text-sm text-red-400">{state.error}</p>
        </div>
      </Show>

      {/* Steps */}
      <Switch>
        <Match when={state.step === 1}>
          <StepSourceType onSelect={selectSource} />
        </Match>
        <Match when={state.step === 2}>
          <StepSourceDetails
            state={state}
            setState={setState}
            onDetect={runDetection}
            onBack={goBack}
          />
        </Match>
        <Match when={state.step === 3}>
          <StepDetectionResults
            detections={state.detections}
            generatedPipeline={state.generatedPipeline}
            editedPipeline={state.editedPipeline}
            onEditPipeline={(yaml) => setState('editedPipeline', yaml)}
            onNext={proceedToCreate}
            onBack={goBack}
          />
        </Match>
        <Match when={state.step === 4}>
          <StepProjectCreate
            state={state}
            setState={setState}
            onCreate={createProject}
            onBack={goBack}
          />
        </Match>
      </Switch>
    </div>
  );
};

export default ImportWizard;
