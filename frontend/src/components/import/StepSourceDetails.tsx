import type { Component } from 'solid-js';
import { createSignal, createResource, Show, For } from 'solid-js';
import type { SetStoreFunction } from 'solid-js/store';
import JSZip from 'jszip';
import Button from '../ui/Button';
import Input from '../ui/Input';
import Badge from '../ui/Badge';
import ConfirmDialog from '../ui/ConfirmDialog';
import { api, ApiRequestError } from '../../api/client';
import { createGitignoreMatcher, createDefaultMatcher } from '../../utils/gitignore';
import type { GitignoreMatcher } from '../../utils/gitignore';
import type { WizardState } from './ImportWizard';
import type { LocalMode } from './ImportWizard';
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
// Local directory form (tabbed: Server Path / Browse File / Browse Folder)
// --------------------------------------------------------------------------
const LocalPathForm: Component<{ state: WizardState; setState: SetStoreFunction<WizardState> }> = (props) => {
  const [uploading, setUploading] = createSignal(false);
  const [dragActive, setDragActive] = createSignal(false);
  const [progress, setProgress] = createSignal('');
  const [pendingFolder, setPendingFolder] = createSignal<{
    name: string;
    allFiles: { data: File; path: string }[];
    matcher: GitignoreMatcher;
    gitignoreFound: boolean;
  } | null>(null);
  const [includeIgnored, setIncludeIgnored] = createSignal(false);

  const activeTab = () => props.state.localMode || 'path';

  const setTab = (tab: LocalMode) => {
    props.setState({
      localMode: tab,
      // Clear previous selections when switching tabs
      localPath: '',
      uploadId: '',
      uploadFilename: '',
      error: '',
    });
  };

  // Upload a File object (archive or zipped folder) via the existing upload endpoint
  const uploadFile = async (file: File) => {
    setUploading(true);
    setProgress('Uploading...');
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
      setProgress('');
    }
  };

  // --- Tab 2: Browse File handlers ---
  const handleFileDrop = (e: DragEvent) => {
    e.preventDefault();
    setDragActive(false);
    const file = e.dataTransfer?.files[0];
    if (file) uploadFile(file);
  };

  const handleFileInput = (e: Event) => {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (file) uploadFile(file);
  };

  // --- Tab 3: Browse Folder ---

  // Recursively collect all files from a FileSystemDirectoryHandle (deeply nested)
  async function collectFilesFromHandle(
    dirHandle: FileSystemDirectoryHandle,
    basePath: string,
  ): Promise<{ data: File; path: string }[]> {
    const files: { data: File; path: string }[] = [];
    for await (const entry of (dirHandle as any).values()) {
      const entryPath = basePath ? `${basePath}/${entry.name}` : entry.name;
      if (entry.kind === 'file') {
        const file = await (entry as FileSystemFileHandle).getFile();
        files.push({ data: file, path: entryPath });
      } else if (entry.kind === 'directory') {
        const nested = await collectFilesFromHandle(entry as FileSystemDirectoryHandle, entryPath);
        files.push(...nested);
      }
    }
    return files;
  }

  // Try to find and read .gitignore from collected files
  async function buildMatcher(files: { data: File; path: string }[], folderName: string): Promise<{ matcher: GitignoreMatcher; found: boolean }> {
    // Look for .gitignore at root level (folderName/.gitignore or just .gitignore)
    const gitignoreFile = files.find(f => {
      const rel = f.path.startsWith(folderName + '/') ? f.path.slice(folderName.length + 1) : f.path;
      return rel === '.gitignore';
    });
    if (gitignoreFile) {
      const content = await gitignoreFile.data.text();
      return { matcher: createGitignoreMatcher(content, true), found: true };
    }
    return { matcher: createDefaultMatcher(), found: false };
  }

  // Use File System Access API when available (no native confirm dialog).
  // Fall back to webkitdirectory for Firefox/Safari (browser-level confirm is unavoidable there).
  const supportsDirectoryPicker = typeof window.showDirectoryPicker === 'function';

  const handleSelectFolder = async () => {
    if (!supportsDirectoryPicker) {
      // Trigger the hidden webkitdirectory input as fallback
      const input = document.getElementById('folder-fallback-input') as HTMLInputElement | null;
      if (input) input.click();
      return;
    }

    try {
      const dirHandle = await window.showDirectoryPicker();
      setProgress('Scanning folder...');
      const allFiles = await collectFilesFromHandle(dirHandle, dirHandle.name);
      if (allFiles.length === 0) {
        props.setState('error', 'Selected folder is empty');
        return;
      }
      const { matcher, found } = await buildMatcher(allFiles, dirHandle.name);
      setIncludeIgnored(false);
      setPendingFolder({ name: dirHandle.name, allFiles, matcher, gitignoreFound: found });
    } catch (err: any) {
      // User cancelled the picker
      if (err.name === 'AbortError') return;
      props.setState('error', err.message || 'Failed to read folder');
    }
  };

  // Fallback handler for browsers without showDirectoryPicker
  const handleFolderFallbackInput = async (e: Event) => {
    const input = e.target as HTMLInputElement;
    const fileList = input.files;
    if (!fileList || fileList.length === 0) return;

    const files: { data: File; path: string }[] = [];
    for (let i = 0; i < fileList.length; i++) {
      const file = fileList[i];
      files.push({
        data: file,
        path: file.webkitRelativePath || file.name,
      });
    }

    const folderName = files[0].path.split('/')[0] || 'project';
    const { matcher, found } = await buildMatcher(files, folderName);
    setIncludeIgnored(false);
    setPendingFolder({ name: folderName, allFiles: files, matcher, gitignoreFound: found });
    // Reset input so re-selecting the same folder works
    input.value = '';
  };

  // Compute filtered file list based on includeIgnored toggle
  const filteredFiles = () => {
    const folder = pendingFolder();
    if (!folder) return [];
    if (includeIgnored()) return folder.allFiles;
    return folder.allFiles.filter(f => !folder.matcher.isIgnored(f.path));
  };

  const ignoredCount = () => {
    const folder = pendingFolder();
    if (!folder) return 0;
    return folder.allFiles.length - folder.allFiles.filter(f => !folder.matcher.isIgnored(f.path)).length;
  };

  // Zip and upload after user confirms
  const confirmFolderUpload = async () => {
    const folder = pendingFolder();
    if (!folder) return;
    const files = filteredFiles();
    setPendingFolder(null);

    if (files.length === 0) {
      props.setState('error', 'No files to upload after filtering');
      return;
    }

    setUploading(true);
    setProgress(`Zipping ${files.length} files...`);
    try {
      const zip = new JSZip();

      for (const file of files) {
        zip.file(file.path, file.data);
      }

      setProgress('Compressing...');
      const blob = await zip.generateAsync({ type: 'blob', compression: 'DEFLATE' });
      const zipFile = new File([blob], `${folder.name}.zip`, { type: 'application/zip' });

      setProgress('Uploading...');
      await uploadFile(zipFile);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to process folder';
      props.setState('error', msg);
      setUploading(false);
      setProgress('');
    }
  };

  const tabs: { key: LocalMode; label: string; icon: string }[] = [
    {
      key: 'path',
      label: 'Server Path',
      icon: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z',
    },
    {
      key: 'file',
      label: 'Browse File',
      icon: 'M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12',
    },
    {
      key: 'folder',
      label: 'Browse Folder',
      icon: 'M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z',
    },
  ];

  return (
    <div class="space-y-4">
      {/* Tab bar */}
      <div class="flex gap-1 p-1 rounded-lg bg-[var(--color-bg-tertiary)]">
        {tabs.map((tab) => (
          <button
            type="button"
            onClick={() => setTab(tab.key)}
            class={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors flex-1 justify-center ${
              activeTab() === tab.key
                ? 'bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] shadow-sm'
                : 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'
            }`}
          >
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d={tab.icon} />
            </svg>
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab 1: Server Path */}
      <Show when={activeTab() === 'path'}>
        <div class="space-y-3">
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
      </Show>

      {/* Tab 2: Browse File */}
      <Show when={activeTab() === 'file'}>
        <Show when={!props.state.uploadId} fallback={
          <div class="p-4 rounded-xl border border-green-500/30 bg-green-500/10">
            <div class="flex items-center gap-2">
              <svg class="w-5 h-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
              </svg>
              <span class="text-sm text-green-400 font-medium">Uploaded: {props.state.uploadFilename}</span>
            </div>
            <button
              type="button"
              onClick={() => props.setState({ uploadId: '', uploadFilename: '' })}
              class="mt-2 text-xs text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] underline"
            >
              Choose a different file
            </button>
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
            onDrop={handleFileDrop}
          >
            <Show when={!uploading()} fallback={
              <div class="flex flex-col items-center gap-2">
                <svg class="animate-spin h-8 w-8 text-indigo-400" viewBox="0 0 24 24" fill="none">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                <span class="text-sm text-[var(--color-text-tertiary)]">{progress()}</span>
              </div>
            }>
              <svg class="w-10 h-10 mx-auto text-[var(--color-text-tertiary)] mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
              </svg>
              <p class="text-sm text-[var(--color-text-secondary)] mb-2">
                Drag and drop your archive here, or{' '}
                <label class="text-indigo-400 hover:text-indigo-300 cursor-pointer">
                  browse
                  <input type="file" class="hidden" accept=".zip,.tar.gz,.tgz" onChange={handleFileInput} />
                </label>
              </p>
              <p class="text-xs text-[var(--color-text-tertiary)]">
                Supports .zip, .tar.gz, .tgz (max 500MB)
              </p>
            </Show>
          </div>
        </Show>
      </Show>

      {/* Tab 3: Browse Folder */}
      <Show when={activeTab() === 'folder'}>
        <Show when={!props.state.uploadId} fallback={
          <div class="p-4 rounded-xl border border-green-500/30 bg-green-500/10">
            <div class="flex items-center gap-2">
              <svg class="w-5 h-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
              </svg>
              <span class="text-sm text-green-400 font-medium">Folder uploaded: {props.state.uploadFilename}</span>
            </div>
            <button
              type="button"
              onClick={() => props.setState({ uploadId: '', uploadFilename: '' })}
              class="mt-2 text-xs text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] underline"
            >
              Choose a different folder
            </button>
          </div>
        }>
          <div class="space-y-3">
            <Show when={!uploading()} fallback={
              <div class="flex flex-col items-center gap-3 p-8">
                <svg class="animate-spin h-8 w-8 text-indigo-400" viewBox="0 0 24 24" fill="none">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                <span class="text-sm text-[var(--color-text-tertiary)]">{progress()}</span>
              </div>
            }>
              <div class="border-2 border-dashed rounded-xl p-8 text-center border-[var(--color-border-primary)] hover:border-[var(--color-border-secondary)] transition-colors">
                <svg class="w-10 h-10 mx-auto text-[var(--color-text-tertiary)] mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                </svg>
                <button
                  type="button"
                  onClick={handleSelectFolder}
                  class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium cursor-pointer transition-colors"
                >
                  <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V8a2 2 0 00-2-2h-5L9 4H4zm7 5a1 1 0 10-2 0v1H8a1 1 0 100 2h1v1a1 1 0 102 0v-1h1a1 1 0 100-2h-1V9z" clip-rule="evenodd" />
                  </svg>
                  Select Folder
                </button>
                {/* Hidden fallback input for browsers without File System Access API */}
                {/* @ts-expect-error - webkitdirectory is a non-standard but widely supported attribute */}
                <input id="folder-fallback-input" type="file" class="hidden" webkitdirectory="" directory="" onChange={handleFolderFallbackInput} />
                <p class="text-xs text-[var(--color-text-tertiary)] mt-3">
                  The folder will be zipped and uploaded automatically.
                </p>
              </div>
            </Show>
          </div>
        </Show>
      </Show>

      {/* Folder upload confirmation dialog */}
      <ConfirmDialog
        open={!!pendingFolder()}
        title="Upload Folder"
        onConfirm={confirmFolderUpload}
        onCancel={() => setPendingFolder(null)}
        confirmLabel={`Upload ${filteredFiles().length} files`}
        variant="primary"
      >
        <div class="space-y-3">
          <p class="text-sm text-[var(--color-text-secondary)]">
            Upload <span class="font-semibold text-[var(--color-text-primary)]">{filteredFiles().length}</span> files from
            folder <span class="font-mono text-[var(--color-text-primary)]">{pendingFolder()?.name}</span> to the server?
          </p>

          <Show when={ignoredCount() > 0}>
            <div class="p-3 rounded-lg bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">
              <p class="text-xs text-[var(--color-text-tertiary)] mb-2">
                <Show when={pendingFolder()?.gitignoreFound} fallback={
                  <>{ignoredCount()} files matched default ignore patterns (.git, node_modules, etc.)</>
                }>
                  {ignoredCount()} files matched .gitignore and default ignore patterns
                </Show>
              </p>
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={includeIgnored()}
                  onChange={(e) => setIncludeIgnored(e.currentTarget.checked)}
                  class="rounded border-[var(--color-border-primary)] text-indigo-600 focus:ring-indigo-500"
                />
                <span class="text-xs text-[var(--color-text-secondary)]">Include ignored files</span>
              </label>
            </div>
          </Show>

          <p class="text-xs text-[var(--color-text-tertiary)]">
            All files including nested subdirectories will be zipped and uploaded.
          </p>
        </div>
      </ConfirmDialog>
    </div>
  );
};

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
      case 'local':
        if (props.state.localMode === 'path') return !!props.state.localPath.trim();
        return !!props.state.uploadId; // file or folder mode
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
