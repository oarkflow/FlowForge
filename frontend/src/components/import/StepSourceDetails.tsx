import type { Component } from 'solid-js';
import { createSignal, createResource, Show, For } from 'solid-js';
import type { SetStoreFunction } from 'solid-js/store';
import Button from '../ui/Button';
import Input from '../ui/Input';
import Badge from '../ui/Badge';
import { api, ApiRequestError } from '../../api/client';
import type { WizardState } from './ImportWizard';
import type { ProviderRepo } from '../../types';

interface Props {
  state: WizardState;
  setState: SetStoreFunction<WizardState>;
  onDetect: () => void;
  onBack: () => void;
}

// --------------------------------------------------------------------------
// Git URL form
// --------------------------------------------------------------------------
const GitURLForm: Component<{ state: WizardState; setState: SetStoreFunction<WizardState> }> = (props) => (
  <div class="space-y-4">
    <Input
      label="Git URL"
      placeholder="https://github.com/user/repo.git or git@github.com:user/repo.git"
      value={props.state.gitUrl}
      onInput={(e) => props.setState('gitUrl', e.currentTarget.value)}
    />
    <Input
      label="Branch (optional)"
      placeholder="main"
      value={props.state.branch}
      onInput={(e) => props.setState('branch', e.currentTarget.value)}
    />
    <details class="group">
      <summary class="text-xs text-[var(--color-text-tertiary)] cursor-pointer hover:text-[var(--color-text-secondary)]">
        Advanced: SSH key authentication
      </summary>
      <div class="mt-2">
        <label class="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">SSH Private Key (PEM)</label>
        <textarea
          class="w-full h-24 px-3 py-2 rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-sm text-[var(--color-text-primary)] font-mono resize-none focus:outline-none focus:border-indigo-500"
          placeholder="-----BEGIN OPENSSH PRIVATE KEY-----..."
          value={props.state.sshKey}
          onInput={(e) => props.setState('sshKey', e.currentTarget.value)}
        />
      </div>
    </details>
  </div>
);

// --------------------------------------------------------------------------
// Provider repo browser
// --------------------------------------------------------------------------
const RepoBrowser: Component<{
  provider: string;
  token: string;
  onSelect: (repo: ProviderRepo) => void;
  selected: ProviderRepo | null;
  onTokenChange: (token: string) => void;
}> = (props) => {
  const [search, setSearch] = createSignal('');
  const [page, setPage] = createSignal(1);
  let debounceTimer: number | undefined;

  const fetchRepos = async () => {
    if (!props.token) return { repos: [], total: 0, page: 1 };
    return api.import.listRepos(props.provider, { search: search(), page: page(), per_page: 20 }, props.token);
  };

  const [repos, { refetch }] = createResource(
    () => ({ token: props.token, search: search(), page: page() }),
    fetchRepos,
  );

  const handleSearch = (value: string) => {
    clearTimeout(debounceTimer);
    debounceTimer = window.setTimeout(() => {
      setSearch(value);
      setPage(1);
    }, 300);
  };

  return (
    <div class="space-y-4">
      <Show when={!props.token} fallback={
        <div>
          <Input
            placeholder={`Search ${props.provider} repositories...`}
            onInput={(e) => handleSearch(e.currentTarget.value)}
            icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clip-rule="evenodd" /></svg>}
          />

          <Show when={repos.loading}>
            <div class="mt-4 space-y-2">
              <For each={[1, 2, 3]}>{() => (
                <div class="p-3 rounded-lg bg-[var(--color-bg-tertiary)] animate-pulse">
                  <div class="h-4 w-48 bg-[var(--color-bg-secondary)] rounded mb-1" />
                  <div class="h-3 w-64 bg-[var(--color-bg-secondary)] rounded" />
                </div>
              )}</For>
            </div>
          </Show>

          <Show when={!repos.loading && repos()}>
            <div class="mt-4 max-h-80 overflow-y-auto space-y-1">
              <For each={repos()?.repos || []}>{(repo) => (
                <button
                  type="button"
                  onClick={() => props.onSelect(repo)}
                  class={`w-full text-left p-3 rounded-lg border transition-colors ${
                    props.selected?.full_name === repo.full_name
                      ? 'border-indigo-500 bg-indigo-500/10'
                      : 'border-transparent hover:bg-[var(--color-bg-tertiary)]'
                  }`}
                >
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-medium text-[var(--color-text-primary)]">{repo.full_name}</span>
                    <Show when={repo.private}>
                      <Badge variant="warning" size="sm">private</Badge>
                    </Show>
                  </div>
                  <Show when={repo.description}>
                    <p class="text-xs text-[var(--color-text-tertiary)] mt-0.5 line-clamp-1">{repo.description}</p>
                  </Show>
                </button>
              )}</For>
              <Show when={(repos()?.repos || []).length === 0}>
                <p class="text-sm text-[var(--color-text-tertiary)] text-center py-4">No repositories found</p>
              </Show>
            </div>

            {/* Pagination */}
            <Show when={(repos()?.total || 0) > 20}>
              <div class="flex justify-center gap-2 mt-3">
                <Button size="sm" variant="ghost" disabled={page() <= 1} onClick={() => setPage(p => p - 1)}>Previous</Button>
                <span class="text-xs text-[var(--color-text-tertiary)] self-center">Page {page()}</span>
                <Button size="sm" variant="ghost" disabled={(repos()?.repos || []).length < 20} onClick={() => setPage(p => p + 1)}>Next</Button>
              </div>
            </Show>
          </Show>
        </div>
      }>
        <div class="space-y-3">
          <Input
            label={`${props.provider.charAt(0).toUpperCase() + props.provider.slice(1)} Personal Access Token`}
            type="password"
            placeholder="Enter your access token..."
            value={props.token}
            onInput={(e) => props.onTokenChange(e.currentTarget.value)}
          />
          <p class="text-xs text-[var(--color-text-tertiary)]">
            Your token is used in-memory only and is never stored.
          </p>
        </div>
      </Show>
    </div>
  );
};

// --------------------------------------------------------------------------
// Local directory form
// --------------------------------------------------------------------------
const LocalPathForm: Component<{ state: WizardState; setState: SetStoreFunction<WizardState> }> = (props) => (
  <div class="space-y-4">
    <Input
      label="Directory Path"
      placeholder="/home/user/projects/my-app"
      value={props.state.localPath}
      onInput={(e) => props.setState('localPath', e.currentTarget.value)}
    />
    <p class="text-xs text-[var(--color-text-tertiary)]">
      Must be an absolute path accessible by the FlowForge server.
    </p>
  </div>
);

// --------------------------------------------------------------------------
// Upload form
// --------------------------------------------------------------------------
const UploadForm: Component<{
  state: WizardState;
  setState: SetStoreFunction<WizardState>;
}> = (props) => {
  const [uploading, setUploading] = createSignal(false);
  const [dragActive, setDragActive] = createSignal(false);

  const handleFile = async (file: File) => {
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append('file', file);
      const result = await api.import.upload(formData);
      props.setState({
        uploadId: result.upload_id,
        uploadFilename: result.filename,
      });
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Upload failed';
      props.setState('error', msg);
    } finally {
      setUploading(false);
    }
  };

  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    setDragActive(false);
    const file = e.dataTransfer?.files[0];
    if (file) handleFile(file);
  };

  const onFileInput = (e: Event) => {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (file) handleFile(file);
  };

  return (
    <div class="space-y-4">
      <Show when={!props.state.uploadId} fallback={
        <div class="p-4 rounded-xl border border-green-500/30 bg-green-500/10">
          <div class="flex items-center gap-2">
            <svg class="w-5 h-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
            </svg>
            <span class="text-sm text-green-400 font-medium">Uploaded: {props.state.uploadFilename}</span>
          </div>
        </div>
      }>
        <div
          class={`border-2 border-dashed rounded-xl p-8 text-center transition-colors ${
            dragActive()
              ? 'border-indigo-500 bg-indigo-500/5'
              : 'border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)]'
          }`}
          onDragOver={(e) => { e.preventDefault(); setDragActive(true); }}
          onDragLeave={() => setDragActive(false)}
          onDrop={onDrop}
        >
          <Show when={!uploading()} fallback={
            <div class="flex flex-col items-center gap-2">
              <svg class="animate-spin h-8 w-8 text-indigo-400" viewBox="0 0 24 24" fill="none">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              <span class="text-sm text-[var(--color-text-tertiary)]">Uploading...</span>
            </div>
          }>
            <svg class="w-10 h-10 mx-auto text-[var(--color-text-tertiary)] mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
            </svg>
            <p class="text-sm text-[var(--color-text-secondary)] mb-2">
              Drag and drop your archive here, or{' '}
              <label class="text-indigo-400 hover:text-indigo-300 cursor-pointer">
                browse
                <input type="file" class="hidden" accept=".zip,.tar.gz,.tgz" onChange={onFileInput} />
              </label>
            </p>
            <p class="text-xs text-[var(--color-text-tertiary)]">
              Supports .zip, .tar.gz, .tgz (max 500MB)
            </p>
          </Show>
        </div>
      </Show>
    </div>
  );
};

// --------------------------------------------------------------------------
// Main step
// --------------------------------------------------------------------------
const StepSourceDetails: Component<Props> = (props) => {
  const canDetect = () => {
    switch (props.state.sourceType) {
      case 'git': return !!props.state.gitUrl.trim();
      case 'github':
      case 'gitlab':
      case 'bitbucket': return !!props.state.selectedRepo;
      case 'local': return !!props.state.localPath.trim();
      case 'upload': return !!props.state.uploadId;
      default: return false;
    }
  };

  const sourceLabel = () => {
    const labels: Record<string, string> = {
      git: 'Git URL',
      github: 'GitHub',
      gitlab: 'GitLab',
      bitbucket: 'Bitbucket',
      local: 'Local Directory',
      upload: 'Upload Archive',
    };
    return labels[props.state.sourceType || ''] || '';
  };

  return (
    <div>
      <h2 class="text-xl font-semibold text-[var(--color-text-primary)] mb-2">
        {sourceLabel()} Details
      </h2>
      <p class="text-sm text-[var(--color-text-secondary)] mb-6">
        Provide the details for your project source.
      </p>

      <div class="mb-6">
        <Show when={props.state.sourceType === 'git'}>
          <GitURLForm state={props.state} setState={props.setState} />
        </Show>
        <Show when={props.state.sourceType === 'github' || props.state.sourceType === 'gitlab' || props.state.sourceType === 'bitbucket'}>
          <RepoBrowser
            provider={props.state.sourceType!}
            token={props.state.providerToken}
            onTokenChange={(token) => props.setState('providerToken', token)}
            selected={props.state.selectedRepo}
            onSelect={(repo) => {
              props.setState({
                selectedRepo: repo,
                cloneUrl: repo.clone_url,
                defaultBranch: repo.default_branch,
              });
            }}
          />
        </Show>
        <Show when={props.state.sourceType === 'local'}>
          <LocalPathForm state={props.state} setState={props.setState} />
        </Show>
        <Show when={props.state.sourceType === 'upload'}>
          <UploadForm state={props.state} setState={props.setState} />
        </Show>
      </div>

      <div class="flex justify-between">
        <Button variant="ghost" onClick={props.onBack}>Back</Button>
        <Button
          onClick={props.onDetect}
          loading={props.state.loading}
          disabled={!canDetect() || props.state.loading}
        >
          Detect Stack
        </Button>
      </div>
    </div>
  );
};

export default StepSourceDetails;
