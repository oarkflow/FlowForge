import type { Component } from 'solid-js';
import { createSignal, createMemo, For, Show } from 'solid-js';
import Badge from '../ui/Badge';

// ---------------------------------------------------------------------------
// JUnit XML Types
// ---------------------------------------------------------------------------
export interface TestCase {
	name: string;
	classname: string;
	time: number;
	status: 'passed' | 'failed' | 'skipped' | 'error';
	failureMessage?: string;
	failureType?: string;
	stackTrace?: string;
}

export interface TestSuite {
	name: string;
	tests: number;
	failures: number;
	errors: number;
	skipped: number;
	time: number;
	testCases: TestCase[];
}

export interface TestReport {
	suites: TestSuite[];
	totalTests: number;
	totalPassed: number;
	totalFailed: number;
	totalSkipped: number;
	totalErrors: number;
	totalTime: number;
}

// ---------------------------------------------------------------------------
// JUnit XML Parser
// ---------------------------------------------------------------------------
export function parseJUnitXml(xmlString: string): TestReport {
	const parser = new DOMParser();
	const doc = parser.parseFromString(xmlString, 'text/xml');
	const suites: TestSuite[] = [];

	const parseSuite = (suiteEl: Element): TestSuite => {
		const testCases: TestCase[] = [];

		const testCaseEls = suiteEl.querySelectorAll('testcase');
		testCaseEls.forEach(tc => {
			const failureEl = tc.querySelector('failure');
			const errorEl = tc.querySelector('error');
			const skippedEl = tc.querySelector('skipped');

			let status: TestCase['status'] = 'passed';
			let failureMessage = '';
			let failureType = '';
			let stackTrace = '';

			if (failureEl) {
				status = 'failed';
				failureMessage = failureEl.getAttribute('message') || '';
				failureType = failureEl.getAttribute('type') || '';
				stackTrace = failureEl.textContent || '';
			} else if (errorEl) {
				status = 'error';
				failureMessage = errorEl.getAttribute('message') || '';
				failureType = errorEl.getAttribute('type') || '';
				stackTrace = errorEl.textContent || '';
			} else if (skippedEl) {
				status = 'skipped';
				failureMessage = skippedEl.getAttribute('message') || '';
			}

			testCases.push({
				name: tc.getAttribute('name') || 'Unknown',
				classname: tc.getAttribute('classname') || '',
				time: parseFloat(tc.getAttribute('time') || '0'),
				status,
				failureMessage,
				failureType,
				stackTrace,
			});
		});

		return {
			name: suiteEl.getAttribute('name') || 'Test Suite',
			tests: parseInt(suiteEl.getAttribute('tests') || '0'),
			failures: parseInt(suiteEl.getAttribute('failures') || '0'),
			errors: parseInt(suiteEl.getAttribute('errors') || '0'),
			skipped: parseInt(suiteEl.getAttribute('skipped') || '0'),
			time: parseFloat(suiteEl.getAttribute('time') || '0'),
			testCases,
		};
	};

	// Handle both <testsuites> and single <testsuite> root
	const suitesRoot = doc.querySelector('testsuites');
	if (suitesRoot) {
		suitesRoot.querySelectorAll(':scope > testsuite').forEach(el => {
			suites.push(parseSuite(el));
		});
	} else {
		const singleSuite = doc.querySelector('testsuite');
		if (singleSuite) {
			suites.push(parseSuite(singleSuite));
		}
	}

	const totalTests = suites.reduce((s, t) => s + t.tests, 0);
	const totalFailed = suites.reduce((s, t) => s + t.failures, 0);
	const totalErrors = suites.reduce((s, t) => s + t.errors, 0);
	const totalSkipped = suites.reduce((s, t) => s + t.skipped, 0);
	const totalPassed = totalTests - totalFailed - totalErrors - totalSkipped;
	const totalTime = suites.reduce((s, t) => s + t.time, 0);

	return { suites, totalTests, totalPassed, totalFailed, totalSkipped, totalErrors, totalTime };
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------
interface TestResultsProps {
	xmlContent?: string;
	report?: TestReport;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const TestResults: Component<TestResultsProps> = (props) => {
	const [filter, setFilter] = createSignal<'all' | 'passed' | 'failed' | 'skipped'>('all');
	const [expandedSuites, setExpandedSuites] = createSignal<Set<string>>(new Set());
	const [expandedTests, setExpandedTests] = createSignal<Set<string>>(new Set());

	const report = createMemo((): TestReport | null => {
		if (props.report) return props.report;
		if (props.xmlContent) {
			try {
				return parseJUnitXml(props.xmlContent);
			} catch {
				return null;
			}
		}
		return null;
	});

	const filteredSuites = createMemo(() => {
		const r = report();
		if (!r) return [];
		const f = filter();
		if (f === 'all') return r.suites;
		return r.suites.map(suite => ({
			...suite,
			testCases: suite.testCases.filter(tc =>
				f === 'failed' ? (tc.status === 'failed' || tc.status === 'error') :
				f === 'skipped' ? tc.status === 'skipped' :
				tc.status === 'passed'
			),
		})).filter(suite => suite.testCases.length > 0);
	});

	const toggleSuite = (name: string) => {
		setExpandedSuites(prev => {
			const next = new Set(prev);
			if (next.has(name)) next.delete(name); else next.add(name);
			return next;
		});
	};

	const toggleTest = (key: string) => {
		setExpandedTests(prev => {
			const next = new Set(prev);
			if (next.has(key)) next.delete(key); else next.add(key);
			return next;
		});
	};

	const statusIcon = (status: TestCase['status']) => {
		switch (status) {
			case 'passed': return <svg class="w-4 h-4 text-emerald-400 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" /></svg>;
			case 'failed': return <svg class="w-4 h-4 text-red-400 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd" /></svg>;
			case 'error': return <svg class="w-4 h-4 text-red-400 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-5a.75.75 0 01.75.75v4.5a.75.75 0 01-1.5 0v-4.5A.75.75 0 0110 5zm0 10a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" /></svg>;
			case 'skipped': return <svg class="w-4 h-4 text-gray-500 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM6.75 9.25a.75.75 0 000 1.5h6.5a.75.75 0 000-1.5h-6.5z" clip-rule="evenodd" /></svg>;
		}
	};

	const formatTime = (seconds: number): string => {
		if (seconds < 0.001) return '<1ms';
		if (seconds < 1) return `${Math.round(seconds * 1000)}ms`;
		if (seconds < 60) return `${seconds.toFixed(2)}s`;
		return `${Math.floor(seconds / 60)}m ${Math.round(seconds % 60)}s`;
	};

	// Duration bar width (relative to max in suite)
	const maxDuration = createMemo(() => {
		const r = report();
		if (!r) return 1;
		const allTimes = r.suites.flatMap(s => s.testCases.map(tc => tc.time));
		return Math.max(...allTimes, 0.001);
	});

	return (
		<Show when={report()} fallback={
			<div class="flex items-center justify-center py-8 text-sm text-[var(--color-text-tertiary)]">
				No test results available.
			</div>
		}>
			{(r) => (
				<div class="space-y-4">
					{/* Summary bar */}
					<div class="flex items-center gap-4 p-4 rounded-xl bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)]">
						<div class="flex-1 grid grid-cols-5 gap-4">
							<div>
								<div class="text-xs text-[var(--color-text-tertiary)]">Total</div>
								<div class="text-xl font-bold text-[var(--color-text-primary)] tabular-nums">{r().totalTests}</div>
							</div>
							<div>
								<div class="text-xs text-emerald-400/80">Passed</div>
								<div class="text-xl font-bold text-emerald-400 tabular-nums">{r().totalPassed}</div>
							</div>
							<div>
								<div class="text-xs text-red-400/80">Failed</div>
								<div class="text-xl font-bold text-red-400 tabular-nums">{r().totalFailed + r().totalErrors}</div>
							</div>
							<div>
								<div class="text-xs text-gray-400/80">Skipped</div>
								<div class="text-xl font-bold text-gray-400 tabular-nums">{r().totalSkipped}</div>
							</div>
							<div>
								<div class="text-xs text-[var(--color-text-tertiary)]">Duration</div>
								<div class="text-xl font-bold text-[var(--color-text-primary)] tabular-nums">{formatTime(r().totalTime)}</div>
							</div>
						</div>
					</div>

					{/* Pass rate bar */}
					<div class="h-2 rounded-full overflow-hidden flex bg-[var(--color-bg-tertiary)]">
						<Show when={r().totalTests > 0}>
							<div class="bg-emerald-500 transition-all" style={{ width: `${(r().totalPassed / r().totalTests) * 100}%` }} />
							<div class="bg-red-500 transition-all" style={{ width: `${((r().totalFailed + r().totalErrors) / r().totalTests) * 100}%` }} />
							<div class="bg-gray-500 transition-all" style={{ width: `${(r().totalSkipped / r().totalTests) * 100}%` }} />
						</Show>
					</div>

					{/* Filters */}
					<div class="flex gap-1 bg-[var(--color-bg-tertiary)] p-1 rounded-lg w-fit">
						{(['all', 'passed', 'failed', 'skipped'] as const).map(f => (
							<button
								class={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors cursor-pointer ${
									filter() === f
										? 'bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] shadow-sm'
										: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'
								}`}
								onClick={() => setFilter(f)}
							>
								{f === 'all' ? `All (${r().totalTests})` :
								 f === 'passed' ? `Passed (${r().totalPassed})` :
								 f === 'failed' ? `Failed (${r().totalFailed + r().totalErrors})` :
								 `Skipped (${r().totalSkipped})`}
							</button>
						))}
					</div>

					{/* Test suites tree */}
					<div class="space-y-2">
						<For each={filteredSuites()}>
							{(suite) => {
								const isExpanded = () => expandedSuites().has(suite.name);
								const passRate = suite.tests > 0 ? Math.round(((suite.tests - suite.failures - suite.errors - suite.skipped) / suite.tests) * 100) : 0;
								return (
									<div class="rounded-xl border border-[var(--color-border-primary)] overflow-hidden">
										{/* Suite header */}
										<button
											class="w-full flex items-center gap-3 px-4 py-3 bg-[var(--color-bg-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors cursor-pointer text-left"
											onClick={() => toggleSuite(suite.name)}
										>
											<svg class={`w-4 h-4 text-[var(--color-text-tertiary)] transition-transform ${isExpanded() ? 'rotate-90' : ''}`} viewBox="0 0 20 20" fill="currentColor">
												<path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
											</svg>
											<div class="flex-1 min-w-0">
												<div class="text-sm font-medium text-[var(--color-text-primary)] truncate">{suite.name}</div>
												<div class="text-xs text-[var(--color-text-tertiary)] mt-0.5">
													{suite.testCases.length} tests | {formatTime(suite.time)}
												</div>
											</div>
											<div class="flex items-center gap-2 flex-shrink-0">
												<div class="w-20 h-1.5 rounded-full bg-[var(--color-bg-tertiary)] overflow-hidden">
													<div class="h-full bg-emerald-500" style={{ width: `${passRate}%` }} />
												</div>
												<span class="text-xs tabular-nums text-[var(--color-text-tertiary)]">{passRate}%</span>
												<Show when={suite.failures > 0 || suite.errors > 0}>
													<Badge variant="error" size="sm">{suite.failures + suite.errors} failed</Badge>
												</Show>
											</div>
										</button>

										{/* Test cases */}
										<Show when={isExpanded()}>
											<div class="divide-y divide-[var(--color-border-primary)]">
												<For each={suite.testCases}>
													{(tc) => {
														const testKey = `${suite.name}::${tc.name}`;
														const isTestExpanded = () => expandedTests().has(testKey);
														const hasDetails = () => tc.failureMessage || tc.stackTrace;
														return (
															<div>
																<button
																	class={`w-full flex items-center gap-3 px-4 py-2.5 hover:bg-[var(--color-bg-hover)] transition-colors text-left ${hasDetails() ? 'cursor-pointer' : 'cursor-default'}`}
																	onClick={() => hasDetails() && toggleTest(testKey)}
																>
																	{statusIcon(tc.status)}
																	<div class="flex-1 min-w-0">
																		<span class="text-sm text-[var(--color-text-primary)]">{tc.name}</span>
																		<Show when={tc.classname}>
																			<span class="text-xs text-[var(--color-text-tertiary)] ml-2">{tc.classname}</span>
																		</Show>
																	</div>
																	{/* Duration bar */}
																	<div class="flex items-center gap-2 flex-shrink-0">
																		<div class="w-16 h-1 rounded-full bg-[var(--color-bg-tertiary)] overflow-hidden">
																			<div
																				class={`h-full rounded-full ${tc.status === 'passed' ? 'bg-emerald-500/50' : tc.status === 'failed' || tc.status === 'error' ? 'bg-red-500/50' : 'bg-gray-500/50'}`}
																				style={{ width: `${Math.min((tc.time / maxDuration()) * 100, 100)}%` }}
																			/>
																		</div>
																		<span class="text-xs tabular-nums text-[var(--color-text-tertiary)] w-16 text-right">{formatTime(tc.time)}</span>
																	</div>
																</button>

																{/* Failure details */}
																<Show when={isTestExpanded() && hasDetails()}>
																	<div class="px-4 pb-3 pt-1 ml-7">
																		<Show when={tc.failureMessage}>
																			<div class="text-xs text-red-400 mb-2">
																				<span class="font-medium">
																					{tc.failureType ? `${tc.failureType}: ` : ''}
																				</span>
																				{tc.failureMessage}
																			</div>
																		</Show>
																		<Show when={tc.stackTrace}>
																			<pre class="text-xs text-[var(--color-text-tertiary)] bg-[#0d1117] rounded-lg p-3 overflow-x-auto font-mono whitespace-pre-wrap max-h-48 overflow-y-auto border border-[var(--color-border-primary)]">
																				{tc.stackTrace}
																			</pre>
																		</Show>
																	</div>
																</Show>
															</div>
														);
													}}
												</For>
											</div>
										</Show>
									</div>
								);
							}}
						</For>
					</div>
				</div>
			)}
		</Show>
	);
};

export default TestResults;
