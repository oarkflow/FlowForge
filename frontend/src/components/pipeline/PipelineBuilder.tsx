import type { Component } from 'solid-js';
import { createSignal, createResource, For, Show, onMount } from 'solid-js';
import * as yaml from 'js-yaml';
import Button from '../ui/Button';
import KeyValueEditor, { type KeyValuePair } from '../ui/KeyValueEditor';
import { api } from '../../api/client';
import type { EnvVar, Secret } from '../../types';

// ---------------------------------------------------------------------------
// Builder data model
// ---------------------------------------------------------------------------
interface BuilderStep {
	id: string;
	name: string;
	run: string;
	uses: string;
	env: KeyValuePair[];
}

interface BuilderJob {
	id: string;
	name: string;
	image: string;
	env: KeyValuePair[];
	steps: BuilderStep[];
	privileged: boolean;
}

interface BuilderStage {
	id: string;
	name: string;
	jobs: BuilderJob[];
}

// Trigger builder data model
interface BuilderTriggers {
	pushEnabled: boolean;
	pushBranches: string[];
	pushTags: string[];
	pushPaths: string[];
	pushIgnorePaths: string[];
	prEnabled: boolean;
	prTypes: string[];
	prBranches: string[];
	scheduleEnabled: boolean;
	schedules: { cron: string; timezone: string; branch: string }[];
	manualEnabled: boolean;
}

const defaultTriggers = (): BuilderTriggers => ({
	pushEnabled: false,
	pushBranches: [],
	pushTags: [],
	pushPaths: [],
	pushIgnorePaths: [],
	prEnabled: false,
	prTypes: [],
	prBranches: [],
	scheduleEnabled: false,
	schedules: [],
	manualEnabled: false,
});

interface PipelineBuilderProps {
	initialYaml: string;
	projectId: string;
	pipelineId: string;
	pipelineName: string;
	onSave: (yaml: string) => void;
	saving: boolean;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const uid = () => crypto.randomUUID();

function kvToMap(pairs: KeyValuePair[]): Record<string, string> | undefined {
	const m: Record<string, string> = {};
	let count = 0;
	for (const p of pairs) {
		if (p.key.trim()) { m[p.key] = p.value; count++; }
	}
	return count > 0 ? m : undefined;
}

function mapToKv(m?: Record<string, string>): KeyValuePair[] {
	if (!m) return [];
	return Object.entries(m).map(([key, value]) => ({ key, value }));
}

// ---------------------------------------------------------------------------
// YAML ↔ Builder conversion
// ---------------------------------------------------------------------------
interface YamlSpec {
	version?: string;
	name?: string;
	on?: {
		push?: { branches?: string[]; tags?: string[]; paths?: string[]; ignore_paths?: string[] };
		pull_request?: { types?: string[]; branches?: string[] };
		schedule?: { cron: string; timezone?: string; branch?: string }[];
		manual?: unknown;
	};
	defaults?: unknown;
	env?: Record<string, string>;
	stages?: string[];
	jobs?: Record<string, YamlJob>;
	notify?: unknown;
}

interface YamlJob {
	stage?: string;
	executor?: string;
	image?: string;
	env?: Record<string, string>;
	steps?: YamlStep[];
	needs?: string[];
	privileged?: boolean;
	when?: string;
	timeout?: string;
	continue_on_error?: boolean;
	[key: string]: unknown;
}

interface YamlStep {
	name?: string;
	run?: string;
	uses?: string;
	with?: Record<string, string>;
	env?: Record<string, string>;
	if?: string;
	outputs?: string[];
}

function yamlToBuilder(yamlStr: string): { stages: BuilderStage[]; pipelineEnv: KeyValuePair[]; pipelineName: string; triggers: BuilderTriggers; raw: YamlSpec } {
	let spec: YamlSpec;
	try {
		spec = (yaml.load(yamlStr) as YamlSpec) || {};
	} catch {
		spec = {};
	}

	// Extract trigger config
	const triggers = defaultTriggers();
	if (spec.on) {
		if (spec.on.push) {
			triggers.pushEnabled = true;
			triggers.pushBranches = spec.on.push.branches || [];
			triggers.pushTags = spec.on.push.tags || [];
			triggers.pushPaths = spec.on.push.paths || [];
			triggers.pushIgnorePaths = spec.on.push.ignore_paths || [];
		}
		if (spec.on.pull_request) {
			triggers.prEnabled = true;
			triggers.prTypes = spec.on.pull_request.types || [];
			triggers.prBranches = spec.on.pull_request.branches || [];
		}
		if (spec.on.schedule && spec.on.schedule.length > 0) {
			triggers.scheduleEnabled = true;
			triggers.schedules = spec.on.schedule.map((s) => ({
				cron: s.cron || '',
				timezone: s.timezone || '',
				branch: s.branch || '',
			}));
		}
		if (spec.on.manual !== undefined && spec.on.manual !== null) {
			triggers.manualEnabled = true;
		}
	}

	const jobs = spec.jobs || {};
	const stageNames: string[] = spec.stages || [];

	// Derive stage order from jobs if not explicit
	if (stageNames.length === 0) {
		const seen = new Set<string>();
		for (const job of Object.values(jobs)) {
			const s = job.stage || 'default';
			if (!seen.has(s)) { seen.add(s); stageNames.push(s); }
		}
	}

	// Group jobs by stage
	const stageMap = new Map<string, BuilderJob[]>();
	for (const sn of stageNames) stageMap.set(sn, []);

	for (const [jobKey, job] of Object.entries(jobs)) {
		const stageName = job.stage || 'default';
		if (!stageMap.has(stageName)) {
			stageNames.push(stageName);
			stageMap.set(stageName, []);
		}
		const steps: BuilderStep[] = (job.steps || []).map((s) => ({
			id: uid(),
			name: s.name || '',
			run: s.run || '',
			uses: s.uses || '',
			env: mapToKv(s.env),
		}));
		stageMap.get(stageName)!.push({
			id: uid(),
			name: jobKey,
			image: job.image || '',
			env: mapToKv(job.env),
			steps,
			privileged: job.privileged || false,
		});
	}

	const stages: BuilderStage[] = stageNames.map((name) => ({
		id: uid(),
		name,
		jobs: stageMap.get(name) || [],
	}));

	return {
		stages,
		pipelineEnv: mapToKv(spec.env),
		pipelineName: spec.name || '',
		triggers,
		raw: spec,
	};
}

function builderToYaml(stages: BuilderStage[], pipelineName: string, pipelineEnv: KeyValuePair[], triggers: BuilderTriggers, rawSpec?: YamlSpec): string {
	const spec: Record<string, unknown> = {};
	spec.version = rawSpec?.version || '1';
	spec.name = pipelineName || rawSpec?.name || 'pipeline';

	// Build triggers from builder state
	const onConfig: Record<string, unknown> = {};
	if (triggers.pushEnabled) {
		const push: Record<string, unknown> = {};
		if (triggers.pushBranches.length > 0) push.branches = triggers.pushBranches;
		if (triggers.pushTags.length > 0) push.tags = triggers.pushTags;
		if (triggers.pushPaths.length > 0) push.paths = triggers.pushPaths;
		if (triggers.pushIgnorePaths.length > 0) push.ignore_paths = triggers.pushIgnorePaths;
		onConfig.push = push;
	}
	if (triggers.prEnabled) {
		const pr: Record<string, unknown> = {};
		if (triggers.prTypes.length > 0) pr.types = triggers.prTypes;
		if (triggers.prBranches.length > 0) pr.branches = triggers.prBranches;
		onConfig.pull_request = pr;
	}
	if (triggers.scheduleEnabled && triggers.schedules.length > 0) {
		onConfig.schedule = triggers.schedules.map((s) => {
			const entry: Record<string, string> = { cron: s.cron };
			if (s.timezone) entry.timezone = s.timezone;
			if (s.branch) entry.branch = s.branch;
			return entry;
		});
	}
	if (triggers.manualEnabled) {
		onConfig.manual = {};
	}
	if (Object.keys(onConfig).length > 0) spec.on = onConfig;

	if (rawSpec?.defaults) spec.defaults = rawSpec.defaults;

	const envMap = kvToMap(pipelineEnv);
	if (envMap) spec.env = envMap;

	// Stages list
	const stageNames = stages.map((s) => s.name);
	if (stageNames.length > 0) spec.stages = stageNames;

	// Jobs
	const jobs: Record<string, unknown> = {};
	for (const stage of stages) {
		for (const job of stage.jobs) {
			const jobKey = job.name || `job-${job.id.slice(0, 8)}`;
			const jobSpec: Record<string, unknown> = { stage: stage.name };
			if (job.image) jobSpec.image = job.image;
			if (job.privileged) jobSpec.privileged = true;
			const jobEnv = kvToMap(job.env);
			if (jobEnv) jobSpec.env = jobEnv;

			const steps: Record<string, unknown>[] = [];
			for (const step of job.steps) {
				const s: Record<string, unknown> = {};
				if (step.name) s.name = step.name;
				if (step.uses) s.uses = step.uses;
				if (step.run) s.run = step.run;
				const stepEnv = kvToMap(step.env);
				if (stepEnv) s.env = stepEnv;
				steps.push(s);
			}
			if (steps.length > 0) jobSpec.steps = steps;
			jobs[jobKey] = jobSpec;
		}
	}
	spec.jobs = jobs;

	if (rawSpec?.notify) spec.notify = rawSpec.notify;

	return yaml.dump(spec, { lineWidth: -1, noRefs: true, quotingType: '"', forceQuotes: false });
}

// ---------------------------------------------------------------------------
// Common images for suggestions
// ---------------------------------------------------------------------------
const COMMON_IMAGES = [
	'alpine:latest', 'ubuntu:22.04', 'node:20-alpine', 'node:18-alpine',
	'golang:1.22-alpine', 'python:3.12-slim', 'ruby:3.3-slim',
	'rust:1.77-slim', 'openjdk:21-slim', 'docker:26-cli',
	'amazon/aws-cli:latest', 'hashicorp/terraform:latest',
];

// ---------------------------------------------------------------------------
// TagInput — inline chip/tag input for pattern lists
// ---------------------------------------------------------------------------
const TagInput: Component<{ values: string[]; onChange: (v: string[]) => void; placeholder?: string }> = (props) => {
	const [input, setInput] = createSignal('');

	const addTag = () => {
		const val = input().trim();
		if (val && !props.values.includes(val)) {
			props.onChange([...props.values, val]);
			setInput('');
		}
	};

	const removeTag = (tag: string) => {
		props.onChange(props.values.filter((v) => v !== tag));
	};

	return (
		<div class="flex flex-wrap items-center gap-1.5 p-1.5 rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] min-h-[34px]">
			<For each={props.values}>
				{(tag) => (
					<span class="inline-flex items-center gap-1 text-xs font-mono px-2 py-0.5 rounded bg-indigo-500/15 text-indigo-300 border border-indigo-500/25">
						{tag}
						<button
							type="button"
							onClick={() => removeTag(tag)}
							class="hover:text-red-400 transition-colors ml-0.5"
						>
							<svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor"><path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" /></svg>
						</button>
					</span>
				)}
			</For>
			<input
				type="text"
				value={input()}
				onInput={(e) => setInput(e.currentTarget.value)}
				onKeyDown={(e) => {
					if (e.key === 'Enter') { e.preventDefault(); addTag(); }
					if (e.key === ',' && input().trim()) { e.preventDefault(); addTag(); }
					if (e.key === 'Backspace' && !input() && props.values.length > 0) {
						removeTag(props.values[props.values.length - 1]);
					}
				}}
				onBlur={() => { if (input().trim()) addTag(); }}
				class="flex-1 min-w-[120px] px-1 py-0.5 text-xs bg-transparent text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] border-none focus:outline-none"
				placeholder={props.values.length === 0 ? props.placeholder : 'Add...'}
			/>
		</div>
	);
};

// ---------------------------------------------------------------------------
// PipelineBuilder component
// ---------------------------------------------------------------------------
const PipelineBuilder: Component<PipelineBuilderProps> = (props) => {
	const [viewMode, setViewMode] = createSignal<'builder' | 'yaml'>('builder');
	const [stages, setStages] = createSignal<BuilderStage[]>([]);
	const [pipelineEnv, setPipelineEnv] = createSignal<KeyValuePair[]>([]);
	const [pipelineName, setPipelineName] = createSignal('');
	const [rawSpec, setRawSpec] = createSignal<YamlSpec | undefined>();
	const [yamlEditContent, setYamlEditContent] = createSignal('');
	const [yamlError, setYamlError] = createSignal('');
	const [collapsedJobs, setCollapsedJobs] = createSignal<Set<string>>(new Set());
	const [collapsedSteps, setCollapsedSteps] = createSignal<Set<string>>(new Set());
	const [collapsedStages, setCollapsedStages] = createSignal<Set<string>>(new Set());
	const [showPipelineEnv, setShowPipelineEnv] = createSignal(false);
	const [showRefPanel, setShowRefPanel] = createSignal(false);
	const [showImageSuggestions, setShowImageSuggestions] = createSignal<string | null>(null);
	const [showTriggers, setShowTriggers] = createSignal(false);
	const [triggers, setTriggers] = createSignal<BuilderTriggers>(defaultTriggers());
	const [dirty, setDirty] = createSignal(false);

	// Fetch project env vars and secrets for reference
	const [projectEnvVars] = createResource(() => props.projectId, async (pid) => {
		try { return await api.envVars.list(pid); } catch { return [] as EnvVar[]; }
	});
	const [projectSecrets] = createResource(() => props.projectId, async (pid) => {
		try { return await api.secrets.list(pid); } catch { return [] as Secret[]; }
	});

	// Initialize from YAML
	onMount(() => {
		if (props.initialYaml) {
			const parsed = yamlToBuilder(props.initialYaml);
			setStages(parsed.stages);
			setPipelineEnv(parsed.pipelineEnv);
			setPipelineName(parsed.pipelineName || props.pipelineName);
			setTriggers(parsed.triggers);
			setRawSpec(parsed.raw);

			// Collapse all stages, jobs, and steps by default
			const stageIds = new Set<string>();
			const jobIds = new Set<string>();
			const stepIds = new Set<string>();
			for (const stage of parsed.stages) {
				stageIds.add(stage.id);
				for (const job of stage.jobs) {
					jobIds.add(job.id);
					for (const step of job.steps) {
						stepIds.add(step.id);
					}
				}
			}
			setCollapsedStages(stageIds);
			setCollapsedJobs(jobIds);
			setCollapsedSteps(stepIds);
		} else {
			setPipelineName(props.pipelineName);
		}
	});

	// Generate current YAML
	const generateYaml = () => builderToYaml(stages(), pipelineName(), pipelineEnv(), triggers(), rawSpec());

	// Mark dirty on any state change
	const markDirty = () => setDirty(true);

	// Stage operations
	const addStage = () => {
		const name = `stage-${stages().length + 1}`;
		setStages([...stages(), { id: uid(), name, jobs: [] }]);
		markDirty();
	};

	const removeStage = (stageId: string) => {
		setStages(stages().filter((s) => s.id !== stageId));
		markDirty();
	};

	const renameStageName = (stageId: string, name: string) => {
		setStages(stages().map((s) => s.id === stageId ? { ...s, name: name.toLowerCase().replace(/[^a-z0-9_-]/g, '') } : s));
		markDirty();
	};

	const moveStage = (stageId: string, dir: -1 | 1) => {
		const arr = [...stages()];
		const idx = arr.findIndex((s) => s.id === stageId);
		if (idx < 0) return;
		const newIdx = idx + dir;
		if (newIdx < 0 || newIdx >= arr.length) return;
		[arr[idx], arr[newIdx]] = [arr[newIdx], arr[idx]];
		setStages(arr);
		markDirty();
	};

	// Job operations
	const addJob = (stageId: string) => {
		setStages(stages().map((s) => {
			if (s.id !== stageId) return s;
			const name = `${s.name}-job-${s.jobs.length + 1}`;
			return { ...s, jobs: [...s.jobs, { id: uid(), name, image: '', env: [], steps: [{ id: uid(), name: 'Run command', run: '', uses: '', env: [] }], privileged: false }] };
		}));
		markDirty();
	};

	const removeJob = (stageId: string, jobId: string) => {
		setStages(stages().map((s) => s.id !== stageId ? s : { ...s, jobs: s.jobs.filter((j) => j.id !== jobId) }));
		markDirty();
	};

	const updateJob = (stageId: string, jobId: string, patch: Partial<BuilderJob>) => {
		setStages(stages().map((s) => {
			if (s.id !== stageId) return s;
			return { ...s, jobs: s.jobs.map((j) => j.id !== jobId ? j : { ...j, ...patch }) };
		}));
		markDirty();
	};

	// Step operations
	const addStep = (stageId: string, jobId: string) => {
		setStages(stages().map((s) => {
			if (s.id !== stageId) return s;
			return {
				...s,
				jobs: s.jobs.map((j) => {
					if (j.id !== jobId) return j;
					return { ...j, steps: [...j.steps, { id: uid(), name: '', run: '', uses: '', env: [] }] };
				}),
			};
		}));
		markDirty();
	};

	const removeStep = (stageId: string, jobId: string, stepId: string) => {
		setStages(stages().map((s) => {
			if (s.id !== stageId) return s;
			return {
				...s,
				jobs: s.jobs.map((j) => {
					if (j.id !== jobId) return j;
					return { ...j, steps: j.steps.filter((st) => st.id !== stepId) };
				}),
			};
		}));
		markDirty();
	};

	const updateStep = (stageId: string, jobId: string, stepId: string, patch: Partial<BuilderStep>) => {
		setStages(stages().map((s) => {
			if (s.id !== stageId) return s;
			return {
				...s,
				jobs: s.jobs.map((j) => {
					if (j.id !== jobId) return j;
					return { ...j, steps: j.steps.map((st) => st.id !== stepId ? st : { ...st, ...patch }) };
				}),
			};
		}));
		markDirty();
	};

	const moveStep = (stageId: string, jobId: string, stepId: string, dir: -1 | 1) => {
		setStages(stages().map((s) => {
			if (s.id !== stageId) return s;
			return {
				...s,
				jobs: s.jobs.map((j) => {
					if (j.id !== jobId) return j;
					const arr = [...j.steps];
					const idx = arr.findIndex((st) => st.id === stepId);
					if (idx < 0) return j;
					const newIdx = idx + dir;
					if (newIdx < 0 || newIdx >= arr.length) return j;
					[arr[idx], arr[newIdx]] = [arr[newIdx], arr[idx]];
					return { ...j, steps: arr };
				}),
			};
		}));
		markDirty();
	};

	// Toggle collapse
	const toggleJob = (jobId: string) => {
		const s = new Set(collapsedJobs());
		if (s.has(jobId)) s.delete(jobId); else s.add(jobId);
		setCollapsedJobs(s);
	};

	const toggleStep = (stepId: string) => {
		const s = new Set(collapsedSteps());
		if (s.has(stepId)) s.delete(stepId); else s.add(stepId);
		setCollapsedSteps(s);
	};

	const toggleStage = (stageId: string) => {
		const s = new Set(collapsedStages());
		if (s.has(stageId)) s.delete(stageId); else s.add(stageId);
		setCollapsedStages(s);
	};

	// Switch to YAML view
	const switchToYaml = () => {
		setYamlEditContent(generateYaml());
		setYamlError('');
		setViewMode('yaml');
	};

	// Switch back to builder from YAML
	const switchToBuilder = () => {
		if (yamlEditContent().trim()) {
			try {
				const parsed = yamlToBuilder(yamlEditContent());
				setStages(parsed.stages);
				setPipelineEnv(parsed.pipelineEnv);
				if (parsed.pipelineName) setPipelineName(parsed.pipelineName);
				setTriggers(parsed.triggers);
				setRawSpec(parsed.raw);
				setYamlError('');
			} catch (e) {
				setYamlError(`Invalid YAML: ${(e as Error).message}`);
				return;
			}
		}
		setViewMode('builder');
	};

	// Save
	const handleSave = () => {
		const yamlContent = viewMode() === 'yaml' ? yamlEditContent() : generateYaml();
		props.onSave(yamlContent);
		setDirty(false);
	};

	// Handle tab key in textareas
	const handleTabKey = (e: KeyboardEvent) => {
		if (e.key === 'Tab') {
			e.preventDefault();
			const target = e.currentTarget as HTMLTextAreaElement;
			const start = target.selectionStart;
			const end = target.selectionEnd;
			const val = target.value;
			target.value = val.substring(0, start) + '  ' + val.substring(end);
			target.selectionStart = target.selectionEnd = start + 2;
			target.dispatchEvent(new Event('input', { bubbles: true }));
		}
	};

	// Auto-resize textarea
	const autoResize = (el: HTMLTextAreaElement) => {
		el.style.height = 'auto';
		el.style.height = Math.max(80, el.scrollHeight) + 'px';
	};

	return (
		<div class="space-y-4">
			{/* Toolbar */}
			<div class="flex items-center justify-between gap-3 p-3 rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
				<div class="flex items-center gap-1 rounded-lg bg-[var(--color-bg-tertiary)] p-1">
					<button
						type="button"
						class={`px-3 py-1.5 text-xs font-medium rounded-md transition-all ${viewMode() === 'builder' ? 'bg-indigo-500 text-white shadow-sm' : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'}`}
						onClick={() => viewMode() === 'yaml' ? switchToBuilder() : undefined}
					>
						Visual Builder
					</button>
					<button
						type="button"
						class={`px-3 py-1.5 text-xs font-medium rounded-md transition-all ${viewMode() === 'yaml' ? 'bg-indigo-500 text-white shadow-sm' : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'}`}
						onClick={() => viewMode() === 'builder' ? switchToYaml() : undefined}
					>
						View YAML
					</button>
				</div>
				<div class="flex items-center gap-2">
					<Show when={dirty()}>
						<span class="text-xs text-amber-400">Unsaved changes</span>
					</Show>
					<Button size="sm" onClick={handleSave} loading={props.saving}>Save Configuration</Button>
				</div>
			</div>

			{/* YAML Error */}
			<Show when={yamlError()}>
				<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-400">{yamlError()}</div>
			</Show>

			{/* YAML View */}
			<Show when={viewMode() === 'yaml'}>
				<div class="rounded-xl border border-[var(--color-border-primary)] overflow-hidden">
					<textarea
						value={yamlEditContent()}
						onInput={(e) => { setYamlEditContent(e.currentTarget.value); markDirty(); }}
						onKeyDown={handleTabKey}
						class="w-full h-[600px] px-4 py-3 text-sm font-mono bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] focus:outline-none resize-y border-none"
						placeholder="# FlowForge Pipeline Configuration (YAML)"
						spellcheck={false}
					/>
				</div>
			</Show>

			{/* Builder View */}
			<Show when={viewMode() === 'builder'}>
				{/* Triggers section */}
				<div class="rounded-xl border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)]">
					<button
						type="button"
						class="w-full flex items-center justify-between px-4 py-3 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
						onClick={() => setShowTriggers(!showTriggers())}
					>
						<span class="flex items-center gap-2">
							<svg class={`w-4 h-4 transition-transform ${showTriggers() ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" /></svg>
							Triggers
							<Show when={triggers().pushEnabled || triggers().prEnabled || triggers().scheduleEnabled || triggers().manualEnabled}>
								<span class="text-xs px-1.5 py-0.5 rounded-full bg-indigo-500/20 text-indigo-400">
									{[triggers().pushEnabled, triggers().prEnabled, triggers().scheduleEnabled, triggers().manualEnabled].filter(Boolean).length} active
								</span>
							</Show>
						</span>
						<span class="text-xs text-[var(--color-text-tertiary)]">Configure when this pipeline runs</span>
					</button>
					<Show when={showTriggers()}>
						<div class="px-4 pb-4 border-t border-[var(--color-border-primary)] pt-3 space-y-4">
							{/* Push Trigger */}
							<div class="rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-primary)] overflow-hidden">
								<label class="flex items-center justify-between px-3 py-2.5 cursor-pointer hover:bg-[var(--color-bg-hover)] transition-colors">
									<span class="flex items-center gap-2 text-sm font-medium text-[var(--color-text-primary)]">
										<svg class="w-4 h-4 text-green-400" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm.75-11.25a.75.75 0 00-1.5 0v2.5h-2.5a.75.75 0 000 1.5h2.5v2.5a.75.75 0 001.5 0v-2.5h2.5a.75.75 0 000-1.5h-2.5v-2.5z" clip-rule="evenodd" /></svg>
										Push
									</span>
									<input
										type="checkbox"
										checked={triggers().pushEnabled}
										onChange={(e) => { setTriggers({ ...triggers(), pushEnabled: e.currentTarget.checked }); markDirty(); }}
										class="w-4 h-4 rounded border-gray-600 text-indigo-500 focus:ring-indigo-500 bg-[var(--color-bg-tertiary)]"
									/>
								</label>
								<Show when={triggers().pushEnabled}>
									<div class="px-3 pb-3 space-y-3 border-t border-[var(--color-border-primary)] pt-3">
										{/* Branches */}
										<div>
											<label class="text-xs font-medium text-[var(--color-text-secondary)] mb-1 block">Branch Patterns</label>
											<TagInput
												values={triggers().pushBranches}
												onChange={(v) => { setTriggers({ ...triggers(), pushBranches: v }); markDirty(); }}
												placeholder="e.g. main, develop, feature/**"
											/>
										</div>
										{/* Tags */}
										<div>
											<label class="text-xs font-medium text-[var(--color-text-secondary)] mb-1 block">Tag Patterns</label>
											<TagInput
												values={triggers().pushTags}
												onChange={(v) => { setTriggers({ ...triggers(), pushTags: v }); markDirty(); }}
												placeholder="e.g. v*, release-*"
											/>
										</div>
										{/* Paths */}
										<div>
											<label class="text-xs font-medium text-[var(--color-text-secondary)] mb-1 block">Path Filters <span class="text-[var(--color-text-tertiary)] font-normal">(only trigger for changes in these paths)</span></label>
											<TagInput
												values={triggers().pushPaths}
												onChange={(v) => { setTriggers({ ...triggers(), pushPaths: v }); markDirty(); }}
												placeholder="e.g. src/**, backend/**"
											/>
										</div>
										{/* Ignore Paths */}
										<div>
											<label class="text-xs font-medium text-[var(--color-text-secondary)] mb-1 block">Ignore Paths <span class="text-[var(--color-text-tertiary)] font-normal">(skip trigger if only these paths change)</span></label>
											<TagInput
												values={triggers().pushIgnorePaths}
												onChange={(v) => { setTriggers({ ...triggers(), pushIgnorePaths: v }); markDirty(); }}
												placeholder="e.g. docs/**, *.md"
											/>
										</div>
									</div>
								</Show>
							</div>

							{/* Pull Request Trigger */}
							<div class="rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-primary)] overflow-hidden">
								<label class="flex items-center justify-between px-3 py-2.5 cursor-pointer hover:bg-[var(--color-bg-hover)] transition-colors">
									<span class="flex items-center gap-2 text-sm font-medium text-[var(--color-text-primary)]">
										<svg class="w-4 h-4 text-blue-400" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M15 10a.75.75 0 01-.75.75H7.612l2.158 1.96a.75.75 0 11-1.04 1.08l-3.5-3.25a.75.75 0 010-1.08l3.5-3.25a.75.75 0 111.04 1.08L7.612 9.25h6.638A.75.75 0 0115 10z" clip-rule="evenodd" /></svg>
										Pull Request
									</span>
									<input
										type="checkbox"
										checked={triggers().prEnabled}
										onChange={(e) => { setTriggers({ ...triggers(), prEnabled: e.currentTarget.checked }); markDirty(); }}
										class="w-4 h-4 rounded border-gray-600 text-indigo-500 focus:ring-indigo-500 bg-[var(--color-bg-tertiary)]"
									/>
								</label>
								<Show when={triggers().prEnabled}>
									<div class="px-3 pb-3 space-y-3 border-t border-[var(--color-border-primary)] pt-3">
										{/* PR Event Types */}
										<div>
											<label class="text-xs font-medium text-[var(--color-text-secondary)] mb-2 block">Event Types</label>
											<div class="flex flex-wrap gap-2">
												<For each={['opened', 'synchronize', 'reopened', 'closed']}>
													{(type) => (
														<label class="inline-flex items-center gap-1.5 cursor-pointer">
															<input
																type="checkbox"
																checked={triggers().prTypes.includes(type)}
																onChange={(e) => {
																	const types = e.currentTarget.checked
																		? [...triggers().prTypes, type]
																		: triggers().prTypes.filter((t) => t !== type);
																	setTriggers({ ...triggers(), prTypes: types });
																	markDirty();
																}}
																class="w-3.5 h-3.5 rounded border-gray-600 text-indigo-500 focus:ring-indigo-500 bg-[var(--color-bg-tertiary)]"
															/>
															<span class="text-xs text-[var(--color-text-secondary)]">{type}</span>
														</label>
													)}
												</For>
											</div>
										</div>
										{/* Target Branches */}
										<div>
											<label class="text-xs font-medium text-[var(--color-text-secondary)] mb-1 block">Target Branch Patterns</label>
											<TagInput
												values={triggers().prBranches}
												onChange={(v) => { setTriggers({ ...triggers(), prBranches: v }); markDirty(); }}
												placeholder="e.g. main, develop"
											/>
										</div>
									</div>
								</Show>
							</div>

							{/* Schedule Trigger */}
							<div class="rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-primary)] overflow-hidden">
								<label class="flex items-center justify-between px-3 py-2.5 cursor-pointer hover:bg-[var(--color-bg-hover)] transition-colors">
									<span class="flex items-center gap-2 text-sm font-medium text-[var(--color-text-primary)]">
										<svg class="w-4 h-4 text-yellow-400" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm.75-13a.75.75 0 00-1.5 0v5c0 .414.336.75.75.75h4a.75.75 0 000-1.5h-3.25V5z" clip-rule="evenodd" /></svg>
										Schedule
									</span>
									<input
										type="checkbox"
										checked={triggers().scheduleEnabled}
										onChange={(e) => {
											const enabled = e.currentTarget.checked;
											const t = { ...triggers(), scheduleEnabled: enabled };
											if (enabled && t.schedules.length === 0) {
												t.schedules = [{ cron: '0 0 * * *', timezone: '', branch: 'main' }];
											}
											setTriggers(t);
											markDirty();
										}}
										class="w-4 h-4 rounded border-gray-600 text-indigo-500 focus:ring-indigo-500 bg-[var(--color-bg-tertiary)]"
									/>
								</label>
								<Show when={triggers().scheduleEnabled}>
									<div class="px-3 pb-3 space-y-2 border-t border-[var(--color-border-primary)] pt-3">
										<For each={triggers().schedules}>
											{(sched, idx) => (
												<div class="flex items-start gap-2 p-2 rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)]">
													<div class="flex-1 space-y-2">
														<div class="flex gap-2">
															<div class="flex-1">
																<label class="text-xs text-[var(--color-text-tertiary)] block mb-0.5">Cron Expression</label>
																<input
																	type="text"
																	value={sched.cron}
																	onInput={(e) => {
																		const scheds = [...triggers().schedules];
																		scheds[idx()] = { ...scheds[idx()], cron: e.currentTarget.value };
																		setTriggers({ ...triggers(), schedules: scheds });
																		markDirty();
																	}}
																	class="w-full px-2 py-1 text-xs font-mono rounded bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:border-indigo-500/50 focus:outline-none focus:ring-1 focus:ring-indigo-500/30"
																	placeholder="0 0 * * *"
																/>
															</div>
															<div class="w-32">
																<label class="text-xs text-[var(--color-text-tertiary)] block mb-0.5">Branch</label>
																<input
																	type="text"
																	value={sched.branch}
																	onInput={(e) => {
																		const scheds = [...triggers().schedules];
																		scheds[idx()] = { ...scheds[idx()], branch: e.currentTarget.value };
																		setTriggers({ ...triggers(), schedules: scheds });
																		markDirty();
																	}}
																	class="w-full px-2 py-1 text-xs rounded bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] text-[var(--color-text-primary)] focus:border-indigo-500/50 focus:outline-none focus:ring-1 focus:ring-indigo-500/30"
																	placeholder="main"
																/>
															</div>
														</div>
														<div class="flex flex-wrap gap-1.5">
															<For each={[
																{ label: 'Hourly', cron: '0 * * * *' },
																{ label: 'Daily midnight', cron: '0 0 * * *' },
																{ label: 'Daily 9am', cron: '0 9 * * *' },
																{ label: 'Weekly', cron: '0 0 * * 0' },
															]}>
																{(preset) => (
																	<button
																		type="button"
																		onClick={() => {
																			const scheds = [...triggers().schedules];
																			scheds[idx()] = { ...scheds[idx()], cron: preset.cron };
																			setTriggers({ ...triggers(), schedules: scheds });
																			markDirty();
																		}}
																		class={`text-[10px] px-1.5 py-0.5 rounded border transition-colors ${sched.cron === preset.cron
																				? 'bg-indigo-500/20 border-indigo-500/30 text-indigo-400'
																				: 'border-[var(--color-border-primary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] hover:border-[var(--color-border-secondary)]'
																			}`}
																	>{preset.label}</button>
																)}
															</For>
														</div>
													</div>
													<button
														type="button"
														onClick={() => {
															const scheds = triggers().schedules.filter((_, i) => i !== idx());
															setTriggers({ ...triggers(), schedules: scheds });
															markDirty();
														}}
														class="mt-4 p-1 rounded text-[var(--color-text-tertiary)] hover:text-red-400 hover:bg-red-500/10 transition-colors"
														title="Remove schedule"
													>
														<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.519.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5z" clip-rule="evenodd" /></svg>
													</button>
												</div>
											)}
										</For>
										<button
											type="button"
											onClick={() => {
												setTriggers({ ...triggers(), schedules: [...triggers().schedules, { cron: '0 0 * * *', timezone: '', branch: 'main' }] });
												markDirty();
											}}
											class="text-xs text-indigo-400 hover:text-indigo-300 transition-colors flex items-center gap-1"
										>
											<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>
											Add Schedule
										</button>
									</div>
								</Show>
							</div>

							{/* Manual Trigger */}
							<div class="rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-primary)] overflow-hidden">
								<label class="flex items-center justify-between px-3 py-2.5 cursor-pointer hover:bg-[var(--color-bg-hover)] transition-colors">
									<span class="flex items-center gap-2 text-sm font-medium text-[var(--color-text-primary)]">
										<svg class="w-4 h-4 text-purple-400" viewBox="0 0 20 20" fill="currentColor"><path d="M6.3 2.841A1.5 1.5 0 004 4.11V15.89a1.5 1.5 0 002.3 1.269l9.344-5.89a1.5 1.5 0 000-2.538L6.3 2.84z" /></svg>
										Manual
										<span class="text-xs text-[var(--color-text-tertiary)] font-normal">— always available from UI</span>
									</span>
									<input
										type="checkbox"
										checked={triggers().manualEnabled}
										onChange={(e) => { setTriggers({ ...triggers(), manualEnabled: e.currentTarget.checked }); markDirty(); }}
										class="w-4 h-4 rounded border-gray-600 text-indigo-500 focus:ring-indigo-500 bg-[var(--color-bg-tertiary)]"
									/>
								</label>
							</div>

							<Show when={!triggers().pushEnabled && !triggers().prEnabled && !triggers().scheduleEnabled && !triggers().manualEnabled}>
								<p class="text-xs text-[var(--color-text-tertiary)] text-center py-1">
									No triggers configured — pipeline will trigger on all events.
								</p>
							</Show>
						</div>
					</Show>
				</div>

				{/* Pipeline-level env vars toggle */}
				<div class="rounded-xl border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)]">
					<button
						type="button"
						class="w-full flex items-center justify-between px-4 py-3 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
						onClick={() => setShowPipelineEnv(!showPipelineEnv())}
					>
						<span class="flex items-center gap-2">
							<svg class={`w-4 h-4 transition-transform ${showPipelineEnv() ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" /></svg>
							Pipeline Environment Variables
							<Show when={pipelineEnv().length > 0}>
								<span class="text-xs px-1.5 py-0.5 rounded-full bg-indigo-500/20 text-indigo-400">{pipelineEnv().length}</span>
							</Show>
						</span>
					</button>
					<Show when={showPipelineEnv()}>
						<div class="px-4 pb-4 border-t border-[var(--color-border-primary)] pt-3">
							<KeyValueEditor
								items={pipelineEnv()}
								onChange={(items) => { setPipelineEnv(items); markDirty(); }}
								keyPlaceholder="VARIABLE_NAME"
								valuePlaceholder="value"
							/>
						</div>
					</Show>
				</div>

				{/* Stages block */}
				<div class="rounded-xl border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] overflow-hidden">
					<div class="flex items-center justify-between px-4 py-3 bg-[var(--color-bg-tertiary)] border-b border-[var(--color-border-primary)]">
						<span class="flex items-center gap-2 text-sm font-medium text-[var(--color-text-primary)]">
							<svg class="w-4 h-4 text-indigo-400" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M2 4.75A.75.75 0 012.75 4h14.5a.75.75 0 010 1.5H2.75A.75.75 0 012 4.75zm0 10.5a.75.75 0 01.75-.75h7.5a.75.75 0 010 1.5h-7.5a.75.75 0 01-.75-.75zM2 10a.75.75 0 01.75-.75h14.5a.75.75 0 010 1.5H2.75A.75.75 0 012 10z" clip-rule="evenodd" /></svg>
							Stages
							<Show when={stages().length > 0}>
								<span class="text-xs px-1.5 py-0.5 rounded-full bg-indigo-500/20 text-indigo-400">{stages().length}</span>
							</Show>
						</span>
						<button
							type="button"
							onClick={addStage}
							class="flex items-center gap-1 px-2.5 py-1 text-xs font-medium text-indigo-400 hover:text-indigo-300 hover:bg-indigo-500/10 rounded-lg transition-colors"
						>
							<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>
							Add Stage
						</button>
					</div>

					<div class="p-4 space-y-3">
						<Show when={stages().length === 0}>
							<div class="text-center py-6">
								<p class="text-sm text-[var(--color-text-tertiary)] mb-2">No stages defined yet.</p>
								<button
									type="button"
									onClick={addStage}
									class="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-indigo-400 hover:text-indigo-300 border border-dashed border-indigo-500/30 hover:border-indigo-500/50 rounded-lg hover:bg-indigo-500/5 transition-colors"
								>
									<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>
									Add your first stage
								</button>
							</div>
						</Show>

						<For each={stages()}>
							{(stage, stageIdx) => {
								const isStageCollapsed = () => collapsedStages().has(stage.id);

								return (
									<div class="rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-primary)] overflow-hidden">
										{/* Stage header — clickable to collapse */}
										<div class="flex items-center gap-2 px-3 py-2.5 bg-[var(--color-bg-secondary)] border-b border-[var(--color-border-primary)]">
											<button
												type="button"
												onClick={() => toggleStage(stage.id)}
												class="p-0.5 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors"
											>
												<svg class={`w-4 h-4 transition-transform ${isStageCollapsed() ? '' : 'rotate-90'}`} viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" /></svg>
											</button>
											<span class="text-xs px-1.5 py-0.5 rounded bg-indigo-500/20 text-indigo-400 font-mono">{stageIdx() + 1}</span>
											<input
												type="text"
												value={stage.name}
												onInput={(e) => renameStageName(stage.id, e.currentTarget.value)}
												class="flex-1 px-2 py-0.5 text-sm font-medium font-mono bg-transparent border border-transparent hover:border-[var(--color-border-primary)] focus:border-indigo-500/50 rounded text-[var(--color-text-primary)] focus:outline-none focus:ring-1 focus:ring-indigo-500/30 transition-colors"
											/>
											<Show when={isStageCollapsed()}>
												<span class="text-xs text-[var(--color-text-tertiary)]">
													{stage.jobs.length} job{stage.jobs.length !== 1 ? 's' : ''}
													{stage.jobs.reduce((sum, j) => sum + j.steps.length, 0) > 0 && `, ${stage.jobs.reduce((sum, j) => sum + j.steps.length, 0)} steps`}
												</span>
											</Show>
											<div class="flex items-center gap-1">
												<button
													type="button"
													disabled={stageIdx() === 0}
													onClick={() => moveStage(stage.id, -1)}
													class="p-1 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
													title="Move up"
												>
													<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M14.77 12.79a.75.75 0 01-1.06-.02L10 9.168l-3.71 3.602a.75.75 0 01-1.08-1.04l4.25-4.5a.75.75 0 011.08 0l4.25 4.5a.75.75 0 01-.02 1.06z" clip-rule="evenodd" /></svg>
												</button>
												<button
													type="button"
													disabled={stageIdx() === stages().length - 1}
													onClick={() => moveStage(stage.id, 1)}
													class="p-1 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
													title="Move down"
												>
													<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 10.832l3.71-3.602a.75.75 0 011.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" /></svg>
												</button>
												<button
													type="button"
													onClick={() => removeStage(stage.id)}
													class="p-1 rounded text-[var(--color-text-tertiary)] hover:text-red-400 hover:bg-red-500/10 transition-colors"
													title="Remove stage"
												>
													<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
												</button>
											</div>
										</div>

										{/* Stage content — collapsible */}
										<Show when={!isStageCollapsed()}>
											{/* Jobs */}
											<div class="p-4 space-y-4">
												<For each={stage.jobs}>
													{(job) => {
														const isCollapsed = () => collapsedJobs().has(job.id);
														const jobEnvCount = () => job.env.filter((e) => e.key.trim()).length;

														return (
															<div class="rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-primary)] overflow-hidden">
																{/* Job header */}
																<div class="flex items-center gap-2 px-3 py-2.5 bg-[var(--color-bg-secondary)] border-b border-[var(--color-border-primary)]">
																	<button
																		type="button"
																		onClick={() => toggleJob(job.id)}
																		class="p-0.5 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] transition-colors"
																	>
																		<svg class={`w-4 h-4 transition-transform ${isCollapsed() ? '' : 'rotate-90'}`} viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" /></svg>
																	</button>
																	<span class="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)]">Job</span>
																	<input
																		type="text"
																		value={job.name}
																		onInput={(e) => updateJob(stage.id, job.id, { name: e.currentTarget.value.toLowerCase().replace(/[^a-z0-9_-]/g, '') })}
																		class="flex-1 px-2 py-0.5 text-sm font-mono bg-transparent border border-transparent hover:border-[var(--color-border-primary)] focus:border-indigo-500/50 rounded text-[var(--color-text-primary)] focus:outline-none focus:ring-1 focus:ring-indigo-500/30 transition-colors"
																	/>
																	<Show when={job.steps.length > 0}>
																		<span class="text-xs text-[var(--color-text-tertiary)]">{job.steps.length} step{job.steps.length !== 1 ? 's' : ''}</span>
																	</Show>
																	<button
																		type="button"
																		onClick={() => removeJob(stage.id, job.id)}
																		class="p-1 rounded text-[var(--color-text-tertiary)] hover:text-red-400 hover:bg-red-500/10 transition-colors"
																		title="Remove job"
																	>
																		<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
																	</button>
																</div>

																<Show when={!isCollapsed()}>
																	<div class="p-3 space-y-3">
																		{/* Image field */}
																		<div class="relative">
																			<label class="block text-xs font-medium text-[var(--color-text-tertiary)] mb-1">Docker Image</label>
																			<div class="flex items-center gap-2">
																				<input
																					type="text"
																					value={job.image}
																					onInput={(e) => updateJob(stage.id, job.id, { image: e.currentTarget.value })}
																					onFocus={() => setShowImageSuggestions(job.id)}
																					onBlur={() => setTimeout(() => setShowImageSuggestions(null), 200)}
																					placeholder="alpine:latest"
																					class="flex-1 px-3 py-1.5 text-sm font-mono bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-lg text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500/50"
																				/>
																				<Show when={job.privileged}>
																					<span class="text-xs px-2 py-1 rounded bg-amber-500/20 text-amber-400 font-medium">Privileged</span>
																				</Show>
																				<button
																					type="button"
																					onClick={() => updateJob(stage.id, job.id, { privileged: !job.privileged })}
																					class={`px-2 py-1.5 text-xs rounded-lg border transition-colors ${job.privileged ? 'border-amber-500/50 text-amber-400 bg-amber-500/10' : 'border-[var(--color-border-primary)] text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'}`}
																					title="Toggle privileged mode (Docker socket access)"
																				>
																					Privileged
																				</button>
																			</div>
																			{/* Image suggestions dropdown */}
																			<Show when={showImageSuggestions() === job.id && !job.image}>
																				<div class="absolute z-10 top-full left-0 right-0 mt-1 max-h-48 overflow-auto rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-primary)] shadow-lg">
																					<For each={COMMON_IMAGES}>
																						{(img) => (
																							<button
																								type="button"
																								onMouseDown={() => updateJob(stage.id, job.id, { image: img })}
																								class="w-full px-3 py-2 text-left text-sm font-mono text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-primary)] transition-colors"
																							>
																								{img}
																							</button>
																						)}
																					</For>
																				</div>
																			</Show>
																		</div>

																		{/* Job env vars (collapsible) */}
																		<div class="rounded-lg border border-[var(--color-border-primary)]">
																			<button
																				type="button"
																				class="w-full flex items-center gap-2 px-3 py-2 text-xs text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] transition-colors"
																				onClick={() => {
																					const key = `job-env-${job.id}`;
																					const s = new Set(collapsedSteps());
																					if (s.has(key)) s.delete(key); else s.add(key);
																					setCollapsedSteps(s);
																				}}
																			>
																				<svg class={`w-3 h-3 transition-transform ${collapsedSteps().has(`job-env-${job.id}`) ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" /></svg>
																				Environment Variables
																				<Show when={jobEnvCount() > 0}>
																					<span class="px-1.5 py-0.5 rounded-full bg-indigo-500/20 text-indigo-400">{jobEnvCount()}</span>
																				</Show>
																			</button>
																			<Show when={collapsedSteps().has(`job-env-${job.id}`)}>
																				<div class="px-3 pb-3 border-t border-[var(--color-border-primary)] pt-2">
																					<KeyValueEditor
																						items={job.env}
																						onChange={(items) => updateJob(stage.id, job.id, { env: items })}
																						keyPlaceholder="VAR_NAME"
																						valuePlaceholder="value"
																					/>
																				</div>
																			</Show>
																		</div>

																		{/* Steps */}
																		<div class="space-y-2">
																			<For each={job.steps}>
																				{(step, stepIdx) => {
																					const stepCollapsed = () => collapsedSteps().has(step.id);
																					const stepEnvCount = () => step.env.filter((e) => e.key.trim()).length;

																					return (
																						<div class="rounded-lg border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)]">
																							{/* Step header */}
																							<div class="flex items-center gap-2 px-3 py-2">
																								<button
																									type="button"
																									onClick={() => toggleStep(step.id)}
																									class="p-0.5 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]"
																								>
																									<svg class={`w-3.5 h-3.5 transition-transform ${stepCollapsed() ? '' : 'rotate-90'}`} viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" /></svg>
																								</button>
																								<span class="text-xs text-[var(--color-text-tertiary)] font-mono">{stepIdx() + 1}.</span>
																								<input
																									type="text"
																									value={step.name}
																									onInput={(e) => updateStep(stage.id, job.id, step.id, { name: e.currentTarget.value })}
																									placeholder="Step name"
																									class="flex-1 px-2 py-0.5 text-sm bg-transparent border border-transparent hover:border-[var(--color-border-primary)] focus:border-indigo-500/50 rounded text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] focus:outline-none focus:ring-1 focus:ring-indigo-500/30 transition-colors"
																								/>
																								<div class="flex items-center gap-0.5">
																									<button
																										type="button"
																										disabled={stepIdx() === 0}
																										onClick={() => moveStep(stage.id, job.id, step.id, -1)}
																										class="p-0.5 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
																										title="Move up"
																									>
																										<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M14.77 12.79a.75.75 0 01-1.06-.02L10 9.168l-3.71 3.602a.75.75 0 01-1.08-1.04l4.25-4.5a.75.75 0 011.08 0l4.25 4.5a.75.75 0 01-.02 1.06z" clip-rule="evenodd" /></svg>
																									</button>
																									<button
																										type="button"
																										disabled={stepIdx() === job.steps.length - 1}
																										onClick={() => moveStep(stage.id, job.id, step.id, 1)}
																										class="p-0.5 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
																										title="Move down"
																									>
																										<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 10.832l3.71-3.602a.75.75 0 011.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" /></svg>
																									</button>
																									<button
																										type="button"
																										onClick={() => removeStep(stage.id, job.id, step.id)}
																										class="p-1 rounded text-[var(--color-text-tertiary)] hover:text-red-400 hover:bg-red-500/10 transition-colors"
																										title="Remove step"
																									>
																										<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
																									</button>
																								</div>
																							</div>

																							<Show when={!stepCollapsed()}>
																								<div class="px-3 pb-3 space-y-2">
																									{/* Uses field (action reference) */}
																									<Show when={step.uses || !step.run}>
																										<div>
																											<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Action (uses)</label>
																											<input
																												type="text"
																												value={step.uses}
																												onInput={(e) => updateStep(stage.id, job.id, step.id, { uses: e.currentTarget.value })}
																												placeholder="e.g. flowforge/checkout@v1"
																												class="w-full px-3 py-1.5 text-sm font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] rounded-lg text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500/50"
																											/>
																										</div>
																									</Show>

																									{/* Run field (command) */}
																									<div>
																										<label class="block text-xs text-[var(--color-text-tertiary)] mb-1">Command</label>
																										<textarea
																											value={step.run}
																											onInput={(e) => {
																												updateStep(stage.id, job.id, step.id, { run: e.currentTarget.value });
																												autoResize(e.currentTarget);
																											}}
																											onKeyDown={handleTabKey}
																											ref={(el) => setTimeout(() => autoResize(el), 0)}
																											placeholder={'echo "Hello World"\n# Add your commands here...'}
																											spellcheck={false}
																											class="w-full px-3 py-2 text-sm font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border-primary)] rounded-lg text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500/50 resize-y leading-relaxed"
																											style={{ 'min-height': '80px' }}
																										/>
																									</div>

																									{/* Step env vars (collapsible) */}
																									<div class="rounded border border-[var(--color-border-primary)]">
																										<button
																											type="button"
																											class="w-full flex items-center gap-2 px-2.5 py-1.5 text-xs text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] transition-colors"
																											onClick={() => {
																												const key = `step-env-${step.id}`;
																												const s = new Set(collapsedSteps());
																												if (s.has(key)) s.delete(key); else s.add(key);
																												setCollapsedSteps(s);
																											}}
																										>
																											<svg class={`w-3 h-3 transition-transform ${collapsedSteps().has(`step-env-${step.id}`) ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" /></svg>
																											Step Environment
																											<Show when={stepEnvCount() > 0}>
																												<span class="px-1.5 py-0.5 rounded-full bg-indigo-500/20 text-indigo-400">{stepEnvCount()}</span>
																											</Show>
																										</button>
																										<Show when={collapsedSteps().has(`step-env-${step.id}`)}>
																											<div class="px-2.5 pb-2.5 border-t border-[var(--color-border-primary)] pt-2">
																												<KeyValueEditor
																													items={step.env}
																													onChange={(items) => updateStep(stage.id, job.id, step.id, { env: items })}
																													keyPlaceholder="VAR_NAME"
																													valuePlaceholder="value"
																												/>
																											</div>
																										</Show>
																									</div>
																								</div>
																							</Show>
																						</div>
																					);
																				}}
																			</For>
																		</div>

																		{/* Add step button */}
																		<button
																			type="button"
																			onClick={() => addStep(stage.id, job.id)}
																			class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-indigo-400 hover:text-indigo-300 hover:bg-indigo-500/10 rounded-lg transition-colors"
																		>
																			<svg class="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>
																			Add Step
																		</button>
																	</div>
																</Show>
															</div>
														);
													}}
												</For>

												{/* Add job button */}
												<button
													type="button"
													onClick={() => addJob(stage.id)}
													class="flex items-center gap-1.5 px-3 py-2 text-xs text-emerald-400 hover:text-emerald-300 hover:bg-emerald-500/10 rounded-lg transition-colors"
												>
													<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z" /></svg>
													Add Job
												</button>
											</div>
										</Show>
									</div>
								);
							}}
						</For>
					</div>
				</div>

				{/* Project variables & secrets reference panel */}
				<div class="rounded-xl border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)]">
					<button
						type="button"
						class="w-full flex items-center justify-between px-4 py-3 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
						onClick={() => setShowRefPanel(!showRefPanel())}
					>
						<span class="flex items-center gap-2">
							<svg class={`w-4 h-4 transition-transform ${showRefPanel() ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" /></svg>
							<svg class="w-4 h-4 text-[var(--color-text-tertiary)]" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM8.94 6.94a.75.75 0 11-1.061-1.061 3 3 0 112.871 5.026v.345a.75.75 0 01-1.5 0v-.916c0-.414.336-.75.75-.75a1.5 1.5 0 10-1.06-2.56zm-.25 6.81a1 1 0 112 0 1 1 0 01-2 0z" clip-rule="evenodd" /></svg>
							Available Project Variables &amp; Secrets
						</span>
					</button>
					<Show when={showRefPanel()}>
						<div class="px-4 pb-4 border-t border-[var(--color-border-primary)] pt-3 space-y-4">
							{/* Env vars */}
							<div>
								<h4 class="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] mb-2">Environment Variables</h4>
								<Show when={(projectEnvVars() ?? []).length > 0} fallback={
									<p class="text-xs text-[var(--color-text-tertiary)]">No project environment variables defined.</p>
								}>
									<div class="flex flex-wrap gap-1.5">
										<For each={projectEnvVars()}>
											{(v) => (
												<button
													type="button"
													onClick={() => navigator.clipboard.writeText(`\${{ env.${v.key} }}`)}
													class="inline-flex items-center gap-1 px-2 py-1 text-xs font-mono rounded bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] text-[var(--color-text-secondary)] hover:border-indigo-500/50 hover:text-indigo-400 transition-colors cursor-pointer"
													title={`Click to copy: \${{ env.${v.key} }}`}
												>
													{v.key}
													<svg class="w-3 h-3 opacity-50" viewBox="0 0 20 20" fill="currentColor"><path d="M7 3.5A1.5 1.5 0 018.5 2h3.879a1.5 1.5 0 011.06.44l3.122 3.12A1.5 1.5 0 0117 6.622V12.5a1.5 1.5 0 01-1.5 1.5h-1v-3.379a3 3 0 00-.879-2.121L10.5 5.379A3 3 0 008.379 4.5H7v-1z" /><path d="M4.5 6A1.5 1.5 0 003 7.5v9A1.5 1.5 0 004.5 18h7a1.5 1.5 0 001.5-1.5v-5.879a1.5 1.5 0 00-.44-1.06L9.44 6.439A1.5 1.5 0 008.378 6H4.5z" /></svg>
												</button>
											)}
										</For>
									</div>
								</Show>
							</div>

							{/* Secrets */}
							<div>
								<h4 class="text-xs font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] mb-2">Secrets</h4>
								<Show when={(projectSecrets() ?? []).length > 0} fallback={
									<p class="text-xs text-[var(--color-text-tertiary)]">No project secrets defined.</p>
								}>
									<div class="flex flex-wrap gap-1.5">
										<For each={projectSecrets()}>
											{(s) => (
												<button
													type="button"
													onClick={() => navigator.clipboard.writeText(`\${{ secrets.${s.key} }}`)}
													class="inline-flex items-center gap-1 px-2 py-1 text-xs font-mono rounded bg-amber-500/10 border border-amber-500/30 text-amber-400 hover:border-amber-400 transition-colors cursor-pointer"
													title={`Click to copy: \${{ secrets.${s.key} }}`}
												>
													<svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 1a4.5 4.5 0 00-4.5 4.5V9H5a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-.5V5.5A4.5 4.5 0 0010 1zm3 8V5.5a3 3 0 10-6 0V9h6z" clip-rule="evenodd" /></svg>
													{s.key}
													<svg class="w-3 h-3 opacity-50" viewBox="0 0 20 20" fill="currentColor"><path d="M7 3.5A1.5 1.5 0 018.5 2h3.879a1.5 1.5 0 011.06.44l3.122 3.12A1.5 1.5 0 0117 6.622V12.5a1.5 1.5 0 01-1.5 1.5h-1v-3.379a3 3 0 00-.879-2.121L10.5 5.379A3 3 0 008.379 4.5H7v-1z" /><path d="M4.5 6A1.5 1.5 0 003 7.5v9A1.5 1.5 0 004.5 18h7a1.5 1.5 0 001.5-1.5v-5.879a1.5 1.5 0 00-.44-1.06L9.44 6.439A1.5 1.5 0 008.378 6H4.5z" /></svg>
												</button>
											)}
										</For>
									</div>
								</Show>
							</div>

							<p class="text-xs text-[var(--color-text-tertiary)]">
								Click a variable or secret to copy its expression syntax. Use <code class="px-1 py-0.5 rounded bg-[var(--color-bg-tertiary)]">{'${{ env.NAME }}'}</code> for variables and <code class="px-1 py-0.5 rounded bg-[var(--color-bg-tertiary)]">{'${{ secrets.NAME }}'}</code> for secrets in your commands.
							</p>
						</div>
					</Show>
				</div>
			</Show>
		</div>
	);
};

export default PipelineBuilder;
