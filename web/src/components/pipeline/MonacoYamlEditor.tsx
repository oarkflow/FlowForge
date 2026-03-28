import type { Component } from 'solid-js';
import { createSignal, createEffect, onMount, onCleanup, Show } from 'solid-js';

// ---------------------------------------------------------------------------
// FlowForge pipeline YAML schema keywords for autocomplete
// ---------------------------------------------------------------------------
const PIPELINE_KEYWORDS = {
	topLevel: ['version', 'name', 'on', 'defaults', 'env', 'stages', 'jobs', 'notify'],
	triggers: ['push', 'pull_request', 'schedule', 'manual', 'api', 'pipeline'],
	triggerPush: ['branches', 'tags', 'paths', 'ignore_paths'],
	triggerPR: ['types', 'branches'],
	triggerSchedule: ['cron', 'timezone', 'branch'],
	triggerManual: ['inputs'],
	inputProps: ['description', 'required', 'default', 'type', 'options'],
	defaults: ['timeout', 'retry', 'executor', 'image'],
	retry: ['count', 'delay', 'on'],
	jobProps: ['stage', 'executor', 'image', 'env', 'cache', 'steps', 'needs', 'matrix', 'when', 'privileged', 'environment', 'approval_required'],
	stepProps: ['name', 'run', 'uses', 'with', 'env', 'outputs', 'if', 'continue_on_error', 'timeout', 'retry'],
	executors: ['local', 'docker', 'kubernetes'],
	cacheProps: ['key', 'paths'],
	notifyEvents: ['on_failure', 'on_success', 'on_deployment'],
	prTypes: ['opened', 'synchronize', 'reopened', 'closed', 'labeled'],
	retryOn: ['failure', 'timeout', 'error'],
};

// ---------------------------------------------------------------------------
// YAML validation helpers
// ---------------------------------------------------------------------------
interface YamlDiagnostic {
	line: number;
	column: number;
	message: string;
	severity: 'error' | 'warning' | 'info';
}

function validatePipelineYaml(content: string): YamlDiagnostic[] {
	const diagnostics: YamlDiagnostic[] = [];

	try {
		// Dynamic import js-yaml for validation
		const lines = content.split('\n');

		// Check for tabs (YAML doesn't allow tabs for indentation)
		lines.forEach((line, idx) => {
			if (line.includes('\t')) {
				diagnostics.push({
					line: idx + 1,
					column: line.indexOf('\t') + 1,
					message: 'YAML does not allow tab characters for indentation. Use spaces instead.',
					severity: 'error',
				});
			}
		});

		// Check for common YAML mistakes
		lines.forEach((line, idx) => {
			// Trailing whitespace after colon in mapping
			const match = line.match(/^(\s*)(\w+):\s*$/);
			if (match && idx + 1 < lines.length) {
				const nextLine = lines[idx + 1];
				const currentIndent = match[1].length;
				const nextIndent = nextLine.match(/^(\s*)/)?.[1].length ?? 0;
				if (nextLine.trim() && nextIndent <= currentIndent && !nextLine.trim().startsWith('#') && !nextLine.trim().startsWith('-')) {
					diagnostics.push({
						line: idx + 1,
						column: 1,
						message: `Expected indented block under "${match[2]}:"`,
						severity: 'warning',
					});
				}
			}
		});

		// Try to parse as YAML to detect syntax errors
		// We use a simple check for now since js-yaml is imported async
		let braceCount = 0;
		let bracketCount = 0;
		lines.forEach((line, idx) => {
			const stripped = line.replace(/#.*$/, '').replace(/'[^']*'/g, '').replace(/"[^"]*"/g, '');
			for (const ch of stripped) {
				if (ch === '{') braceCount++;
				if (ch === '}') braceCount--;
				if (ch === '[') bracketCount++;
				if (ch === ']') bracketCount--;
			}
			if (braceCount < 0) {
				diagnostics.push({ line: idx + 1, column: 1, message: 'Unmatched closing brace "}"', severity: 'error' });
				braceCount = 0;
			}
			if (bracketCount < 0) {
				diagnostics.push({ line: idx + 1, column: 1, message: 'Unmatched closing bracket "]"', severity: 'error' });
				bracketCount = 0;
			}
		});

	} catch {
		// Validation failed, which is ok
	}

	return diagnostics;
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------
interface MonacoYamlEditorProps {
	value: string;
	onChange?: (value: string) => void;
	readOnly?: boolean;
	height?: string;
	onValidation?: (errors: YamlDiagnostic[]) => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const MonacoYamlEditor: Component<MonacoYamlEditorProps> = (props) => {
	let containerRef: HTMLDivElement | undefined;
	let editor: any;
	let monaco: any;
	const [loaded, setLoaded] = createSignal(false);
	const [error, setError] = createSignal<string | null>(null);
	const [diagnostics, setDiagnostics] = createSignal<YamlDiagnostic[]>([]);

	onMount(async () => {
		try {
			const monacoModule = await import('monaco-editor');
			monaco = monacoModule;

			// Define FlowForge dark theme
			monaco.editor.defineTheme('flowforge-dark', {
				base: 'vs-dark',
				inherit: true,
				rules: [
					{ token: 'comment', foreground: '6a737d', fontStyle: 'italic' },
					{ token: 'string', foreground: '9ecbff' },
					{ token: 'number', foreground: '79c0ff' },
					{ token: 'keyword', foreground: 'ff7b72' },
					{ token: 'type', foreground: 'ff7b72' },
					{ token: 'tag', foreground: '7ee787' },
				],
				colors: {
					'editor.background': '#0d1117',
					'editor.foreground': '#c9d1d9',
					'editor.lineHighlightBackground': '#161b22',
					'editor.selectionBackground': '#264f78',
					'editorCursor.foreground': '#58a6ff',
					'editor.inactiveSelectionBackground': '#264f7833',
					'editorLineNumber.foreground': '#484f58',
					'editorLineNumber.activeForeground': '#c9d1d9',
					'editorIndentGuide.background': '#21262d',
					'editorIndentGuide.activeBackground': '#30363d',
					'editorWidget.background': '#161b22',
					'editorWidget.border': '#30363d',
					'editorSuggestWidget.background': '#161b22',
					'editorSuggestWidget.border': '#30363d',
					'editorSuggestWidget.selectedBackground': '#264f78',
					'minimap.background': '#0d1117',
					'scrollbarSlider.background': '#484f5833',
					'scrollbarSlider.hoverBackground': '#484f5855',
					'scrollbarSlider.activeBackground': '#484f5877',
				},
			});

			// Register YAML completions for flowforge pipeline
			monaco.languages.registerCompletionItemProvider('yaml', {
				provideCompletionItems: (model: any, position: any) => {
					const textUntilPosition = model.getValueInRange({
						startLineNumber: 1,
						startColumn: 1,
						endLineNumber: position.lineNumber,
						endColumn: position.column,
					});

					const word = model.getWordUntilPosition(position);
					const range = {
						startLineNumber: position.lineNumber,
						endLineNumber: position.lineNumber,
						startColumn: word.startColumn,
						endColumn: word.endColumn,
					};

					const suggestions: any[] = [];
					const currentLine = model.getLineContent(position.lineNumber);
					const indent = currentLine.match(/^(\s*)/)?.[1].length ?? 0;

					// Determine context based on indentation and preceding lines
					const linesAbove = textUntilPosition.split('\n');
					let context = 'top';

					for (let i = linesAbove.length - 2; i >= 0; i--) {
						const line = linesAbove[i].trim();
						if (!line || line.startsWith('#')) continue;
						if (line === 'on:' || line.startsWith('on:')) { context = 'triggers'; break; }
						if (line === 'push:') { context = 'triggerPush'; break; }
						if (line === 'pull_request:') { context = 'triggerPR'; break; }
						if (line === 'schedule:') { context = 'triggerSchedule'; break; }
						if (line === 'manual:') { context = 'triggerManual'; break; }
						if (line === 'defaults:') { context = 'defaults'; break; }
						if (line === 'retry:') { context = 'retry'; break; }
						if (line === 'jobs:') { context = 'jobs'; break; }
						if (line === 'steps:') { context = 'steps'; break; }
						if (line.match(/^\w+:$/) && context === 'jobs') { context = 'jobProps'; break; }
						if (line === 'notify:') { context = 'notify'; break; }
						if (line === 'cache:') { context = 'cacheProps'; break; }
						if (indent === 0) { context = 'top'; break; }
						break;
					}

					const addSuggestions = (keywords: string[], kind: number = 14) => {
						for (const kw of keywords) {
							suggestions.push({
								label: kw,
								kind,
								insertText: kw + ': ',
								range,
								detail: 'FlowForge pipeline',
							});
						}
					};

					switch (context) {
						case 'top': addSuggestions(PIPELINE_KEYWORDS.topLevel); break;
						case 'triggers': addSuggestions(PIPELINE_KEYWORDS.triggers); break;
						case 'triggerPush': addSuggestions(PIPELINE_KEYWORDS.triggerPush); break;
						case 'triggerPR': addSuggestions(PIPELINE_KEYWORDS.triggerPR); break;
						case 'triggerSchedule': addSuggestions(PIPELINE_KEYWORDS.triggerSchedule); break;
						case 'triggerManual': addSuggestions(PIPELINE_KEYWORDS.triggerManual); break;
						case 'defaults': addSuggestions(PIPELINE_KEYWORDS.defaults); break;
						case 'retry': addSuggestions(PIPELINE_KEYWORDS.retry); break;
						case 'jobProps': addSuggestions(PIPELINE_KEYWORDS.jobProps); break;
						case 'steps': addSuggestions(PIPELINE_KEYWORDS.stepProps); break;
						case 'notify': addSuggestions(PIPELINE_KEYWORDS.notifyEvents); break;
						case 'cacheProps': addSuggestions(PIPELINE_KEYWORDS.cacheProps); break;
						default: addSuggestions(PIPELINE_KEYWORDS.topLevel); break;
					}

					// Add snippet completions
					suggestions.push({
						label: 'job (snippet)',
						kind: 15,
						insertText: [
							'${1:job_name}:',
							'  stage: ${2:build}',
							'  executor: docker',
							'  image: ${3:ubuntu:22.04}',
							'  steps:',
							'    - name: ${4:Step name}',
							'      run: ${5:echo "Hello"}',
						].join('\n'),
						insertTextRules: 4, // InsertAsSnippet
						range,
						detail: 'New job definition',
						documentation: 'Creates a new job with stage, executor, image, and steps',
					});

					suggestions.push({
						label: 'step (snippet)',
						kind: 15,
						insertText: [
							'- name: ${1:Step name}',
							'  run: |',
							'    ${2:echo "Hello World"}',
						].join('\n'),
						insertTextRules: 4,
						range,
						detail: 'New step with run command',
					});

					suggestions.push({
						label: 'trigger push (snippet)',
						kind: 15,
						insertText: [
							'push:',
							'  branches: [${1:main}, ${2:develop}]',
							'  paths: [${3:src/**}]',
						].join('\n'),
						insertTextRules: 4,
						range,
						detail: 'Push trigger configuration',
					});

					return { suggestions };
				},
			});

			if (!containerRef) return;

			editor = monaco.editor.create(containerRef, {
				value: props.value,
				language: 'yaml',
				theme: 'flowforge-dark',
				readOnly: props.readOnly ?? false,
				minimap: { enabled: true, scale: 2, showSlider: 'mouseover' },
				scrollBeyondLastLine: false,
				fontSize: 13,
				fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, Monaco, Consolas, monospace",
				lineNumbers: 'on',
				renderLineHighlight: 'gutter',
				matchBrackets: 'always',
				automaticLayout: true,
				tabSize: 2,
				insertSpaces: true,
				wordWrap: 'off',
				folding: true,
				foldingStrategy: 'indentation',
				bracketPairColorization: { enabled: true },
				guides: { indentation: true, bracketPairs: true },
				suggest: {
					showKeywords: true,
					showSnippets: true,
				},
				quickSuggestions: { other: true, strings: true, comments: false },
				padding: { top: 8, bottom: 8 },
				overviewRulerLanes: 2,
				scrollbar: {
					verticalScrollbarSize: 10,
					horizontalScrollbarSize: 10,
				},
			});

			// Listen for content changes
			editor.onDidChangeModelContent(() => {
				const value = editor.getModel()?.getValue() ?? '';
				props.onChange?.(value);

				// Run validation
				const errors = validatePipelineYaml(value);
				setDiagnostics(errors);
				props.onValidation?.(errors);

				// Set markers on the model
				const model = editor.getModel();
				if (model && monaco) {
					const markers = errors.map((e: YamlDiagnostic) => ({
						severity: e.severity === 'error'
							? monaco.MarkerSeverity.Error
							: e.severity === 'warning'
								? monaco.MarkerSeverity.Warning
								: monaco.MarkerSeverity.Info,
						startLineNumber: e.line,
						startColumn: e.column,
						endLineNumber: e.line,
						endColumn: model.getLineMaxColumn(e.line),
						message: e.message,
					}));
					monaco.editor.setModelMarkers(model, 'flowforge', markers);
				}
			});

			setLoaded(true);
		} catch (err) {
			console.error('Failed to load Monaco editor:', err);
			setError('Failed to load code editor. Falling back to basic editor.');
		}
	});

	// Sync external value changes
	createEffect(() => {
		const val = props.value;
		if (editor && editor.getModel()?.getValue() !== val) {
			editor.getModel()?.setValue(val);
		}
	});

	// Sync readOnly
	createEffect(() => {
		if (editor) {
			editor.updateOptions({ readOnly: props.readOnly ?? false });
		}
	});

	onCleanup(() => {
		editor?.dispose();
	});

	return (
		<div class="flex flex-col rounded-xl overflow-hidden border border-[var(--color-border-primary)]">
			{/* Toolbar */}
			<div class="flex items-center justify-between px-3 py-2 bg-[#161b22] border-b border-[var(--color-border-primary)]">
				<div class="flex items-center gap-2">
					<span class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">YAML Editor</span>
					<Show when={props.readOnly}>
						<span class="px-1.5 py-0.5 text-[10px] rounded bg-amber-500/20 text-amber-400 border border-amber-500/30">Read Only</span>
					</Show>
				</div>
				<div class="flex items-center gap-2">
					<Show when={diagnostics().filter(d => d.severity === 'error').length > 0}>
						<span class="flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded bg-red-500/20 text-red-400 border border-red-500/30">
							<svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" />
							</svg>
							{diagnostics().filter(d => d.severity === 'error').length} error{diagnostics().filter(d => d.severity === 'error').length !== 1 ? 's' : ''}
						</span>
					</Show>
					<Show when={diagnostics().filter(d => d.severity === 'warning').length > 0}>
						<span class="flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded bg-amber-500/20 text-amber-400 border border-amber-500/30">
							<svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
							</svg>
							{diagnostics().filter(d => d.severity === 'warning').length}
						</span>
					</Show>
					<Show when={loaded() && diagnostics().length === 0}>
						<span class="flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
							<svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" />
							</svg>
							Valid
						</span>
					</Show>
				</div>
			</div>

			{/* Error fallback */}
			<Show when={error()}>
				<div class="p-3 bg-amber-500/10 border-b border-amber-500/30 text-xs text-amber-400">
					{error()}
				</div>
			</Show>

			{/* Loading state */}
			<Show when={!loaded() && !error()}>
				<div class="flex items-center justify-center bg-[#0d1117]" style={{ height: props.height || '400px' }}>
					<div class="flex items-center gap-2 text-sm text-[var(--color-text-tertiary)]">
						<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
						</svg>
						Loading editor...
					</div>
				</div>
			</Show>

			{/* Monaco container */}
			<div
				ref={containerRef}
				class={loaded() ? '' : 'hidden'}
				style={{ height: props.height || '400px', background: '#0d1117' }}
			/>
		</div>
	);
};

export default MonacoYamlEditor;
