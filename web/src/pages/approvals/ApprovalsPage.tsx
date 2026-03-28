import { createSignal, createResource, For, Show, type Component, createMemo } from 'solid-js';
import { api } from '../../api/client';
import type { Approval, ApprovalResponse, ApprovalStatus } from '../../types';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type FilterTab = 'pending' | 'approved' | 'rejected' | 'expired' | 'all';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function statusBadge(status: ApprovalStatus) {
	const map: Record<ApprovalStatus, { label: string; classes: string }> = {
		pending: { label: 'Pending', classes: 'bg-yellow-500/10 text-yellow-400 ring-yellow-400/30' },
		approved: { label: 'Approved', classes: 'bg-emerald-500/10 text-emerald-400 ring-emerald-400/30' },
		rejected: { label: 'Rejected', classes: 'bg-red-500/10 text-red-400 ring-red-400/30' },
		expired: { label: 'Expired', classes: 'bg-gray-500/10 text-gray-400 ring-gray-400/30' },
		cancelled: { label: 'Cancelled', classes: 'bg-gray-500/10 text-gray-400 ring-gray-400/30' },
	};
	const s = map[status] || map.pending;
	return (
		<span class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ring-1 ring-inset ${s.classes}`}>
			{s.label}
		</span>
	);
}

function typeIcon(type: string) {
	if (type === 'deployment') {
		return (
			<svg class="w-5 h-5 text-blue-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
				<path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
			</svg>
		);
	}
	return (
		<svg class="w-5 h-5 text-purple-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
			<path d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" />
		</svg>
	);
}

function timeAgo(dateStr: string): string {
	const now = Date.now();
	const then = new Date(dateStr).getTime();
	const diff = now - then;
	const mins = Math.floor(diff / 60000);
	if (mins < 1) return 'just now';
	if (mins < 60) return `${mins}m ago`;
	const hours = Math.floor(mins / 60);
	if (hours < 24) return `${hours}h ago`;
	const days = Math.floor(hours / 24);
	return `${days}d ago`;
}

function expiryCountdown(expiresAt: string | null): string {
	if (!expiresAt) return 'No expiry';
	const now = Date.now();
	const exp = new Date(expiresAt).getTime();
	const diff = exp - now;
	if (diff <= 0) return 'Expired';
	const hours = Math.floor(diff / 3600000);
	const mins = Math.floor((diff % 3600000) / 60000);
	if (hours > 24) {
		const days = Math.floor(hours / 24);
		return `${days}d ${hours % 24}h left`;
	}
	return `${hours}h ${mins}m left`;
}

function parseApprovers(json: string): string[] {
	try {
		return JSON.parse(json);
	} catch {
		return [];
	}
}

// ---------------------------------------------------------------------------
// ApprovalsPage
// ---------------------------------------------------------------------------
const ApprovalsPage: Component = () => {
	const [activeTab, setActiveTab] = createSignal<FilterTab>('pending');
	const [expandedId, setExpandedId] = createSignal<string | null>(null);
	const [commentText, setCommentText] = createSignal('');
	const [actionLoading, setActionLoading] = createSignal<string | null>(null);
	const [actionError, setActionError] = createSignal('');

	// Fetch pending approvals
	const [pendingApprovals, { refetch: refetchPending }] = createResource(
		() => api.approvals.listPending().catch(() => [] as Approval[])
	);

	// Fetch all approvals (we don't have a global list endpoint, so we'll combine pending with a placeholder)
	// Since we don't have a "list all" endpoint, we'll show pending from the dedicated endpoint
	// For history, users can view per-project approvals from the project detail page

	// Expanded approval detail
	const [expandedDetail, { refetch: refetchDetail }] = createResource(
		() => expandedId(),
		async (id) => {
			if (!id) return null;
			try {
				return await api.approvals.get(id);
			} catch {
				return null;
			}
		}
	);

	const filteredApprovals = createMemo(() => {
		const list = pendingApprovals() || [];
		const tab = activeTab();
		if (tab === 'all') return list;
		return list.filter((a) => a.status === tab);
	});

	const tabs: { key: FilterTab; label: string }[] = [
		{ key: 'pending', label: 'Pending' },
		{ key: 'approved', label: 'Approved' },
		{ key: 'rejected', label: 'Rejected' },
		{ key: 'expired', label: 'Expired' },
		{ key: 'all', label: 'All' },
	];

	const pendingCount = createMemo(() => {
		const list = pendingApprovals() || [];
		return list.filter((a) => a.status === 'pending').length;
	});

	// Actions
	async function handleApprove(approvalId: string) {
		setActionLoading(approvalId);
		setActionError('');
		try {
			await api.approvals.approve(approvalId, commentText());
			setCommentText('');
			setExpandedId(null);
			refetchPending();
		} catch (e: any) {
			setActionError(e.message || 'Failed to approve');
		} finally {
			setActionLoading(null);
		}
	}

	async function handleReject(approvalId: string) {
		if (!commentText()) {
			setActionError('A comment is required when rejecting');
			return;
		}
		setActionLoading(approvalId);
		setActionError('');
		try {
			await api.approvals.reject(approvalId, commentText());
			setCommentText('');
			setExpandedId(null);
			refetchPending();
		} catch (e: any) {
			setActionError(e.message || 'Failed to reject');
		} finally {
			setActionLoading(null);
		}
	}

	async function handleCancel(approvalId: string) {
		setActionLoading(approvalId);
		setActionError('');
		try {
			await api.approvals.cancel(approvalId);
			setExpandedId(null);
			refetchPending();
		} catch (e: any) {
			setActionError(e.message || 'Failed to cancel');
		} finally {
			setActionLoading(null);
		}
	}

	function toggleExpand(id: string) {
		if (expandedId() === id) {
			setExpandedId(null);
		} else {
			setExpandedId(id);
			setCommentText('');
			setActionError('');
		}
	}

	return (
		<div class="min-h-screen">
			{/* Header */}
			<div class="mb-6">
				<h1 class="text-2xl font-bold text-[var(--color-text-primary)]">Approvals</h1>
				<p class="mt-1 text-sm text-[var(--color-text-tertiary)]">
					Review and manage deployment and pipeline approval requests.
				</p>
			</div>

			{/* Summary Cards */}
			<div class="grid grid-cols-1 sm:grid-cols-4 gap-4 mb-6">
				<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4">
					<div class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Pending</div>
					<div class="mt-1 text-2xl font-bold text-yellow-400">{pendingCount()}</div>
				</div>
				<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4">
					<div class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Approved</div>
					<div class="mt-1 text-2xl font-bold text-emerald-400">
						{(pendingApprovals() || []).filter((a) => a.status === 'approved').length}
					</div>
				</div>
				<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4">
					<div class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Rejected</div>
					<div class="mt-1 text-2xl font-bold text-red-400">
						{(pendingApprovals() || []).filter((a) => a.status === 'rejected').length}
					</div>
				</div>
				<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl p-4">
					<div class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider">Expired</div>
					<div class="mt-1 text-2xl font-bold text-gray-400">
						{(pendingApprovals() || []).filter((a) => a.status === 'expired').length}
					</div>
				</div>
			</div>

			{/* Filter Tabs */}
			<div class="flex gap-1 mb-4 bg-[var(--color-bg-secondary)] rounded-lg p-1 border border-[var(--color-border-primary)] w-fit">
				<For each={tabs}>
					{(tab) => (
						<button
							class={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${activeTab() === tab.key
									? 'bg-[var(--color-accent-bg)] text-indigo-400'
									: 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)]'
								}`}
							onClick={() => setActiveTab(tab.key)}
						>
							{tab.label}
							<Show when={tab.key === 'pending' && pendingCount() > 0}>
								<span class="ml-1.5 inline-flex items-center justify-center w-5 h-5 rounded-full bg-yellow-500/20 text-yellow-400 text-[10px] font-bold">
									{pendingCount()}
								</span>
							</Show>
						</button>
					)}
				</For>
			</div>

			{/* Approval List */}
			<Show
				when={!pendingApprovals.loading}
				fallback={
					<div class="flex items-center justify-center py-12">
						<svg class="animate-spin h-6 w-6 text-[var(--color-text-tertiary)]" viewBox="0 0 24 24" fill="none">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
						</svg>
					</div>
				}
			>
				<Show
					when={filteredApprovals().length > 0}
					fallback={
						<div class="text-center py-12 bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl">
							<svg class="mx-auto h-12 w-12 text-[var(--color-text-tertiary)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
								<path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
							</svg>
							<h3 class="mt-3 text-sm font-medium text-[var(--color-text-primary)]">No approvals found</h3>
							<p class="mt-1 text-sm text-[var(--color-text-tertiary)]">
								{activeTab() === 'pending'
									? 'No pending approvals requiring your attention.'
									: `No ${activeTab()} approvals to show.`}
							</p>
						</div>
					}
				>
					<div class="space-y-3">
						<For each={filteredApprovals()}>
							{(approval) => (
								<div class="bg-[var(--color-bg-secondary)] border border-[var(--color-border-primary)] rounded-xl overflow-hidden">
									{/* Card Header */}
									<div
										class="p-4 cursor-pointer hover:bg-[var(--color-bg-hover)] transition-colors"
										onClick={() => toggleExpand(approval.id)}
									>
										<div class="flex items-center justify-between">
											<div class="flex items-center gap-3">
												{typeIcon(approval.type)}
												<div>
													<div class="flex items-center gap-2">
														<span class="text-sm font-medium text-[var(--color-text-primary)]">
															{approval.type === 'deployment' ? 'Deployment' : 'Pipeline Run'} Approval
														</span>
														{statusBadge(approval.status)}
													</div>
													<div class="flex items-center gap-3 mt-1 text-xs text-[var(--color-text-tertiary)]">
														<span>Project: {approval.project_id}</span>
														<Show when={approval.environment_id}>
															<span>• Env: {approval.environment_id}</span>
														</Show>
														<span>• Requested {timeAgo(approval.created_at)}</span>
													</div>
												</div>
											</div>

											<div class="flex items-center gap-4">
												{/* Approval Progress */}
												<div class="text-right">
													<div class="text-xs text-[var(--color-text-tertiary)]">Progress</div>
													<div class="text-sm font-medium text-[var(--color-text-primary)]">
														{approval.current_approvals} / {approval.min_approvals}
													</div>
												</div>

												{/* Expiry */}
												<Show when={approval.status === 'pending'}>
													<div class="text-right">
														<div class="text-xs text-[var(--color-text-tertiary)]">Expires</div>
														<div class={`text-xs font-medium ${approval.expires_at && new Date(approval.expires_at).getTime() - Date.now() < 3600000
																? 'text-red-400'
																: 'text-[var(--color-text-secondary)]'
															}`}>
															{expiryCountdown(approval.expires_at)}
														</div>
													</div>
												</Show>

												{/* Expand icon */}
												<svg
													class={`w-5 h-5 text-[var(--color-text-tertiary)] transition-transform ${expandedId() === approval.id ? 'rotate-180' : ''
														}`}
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="1.5"
												>
													<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
												</svg>
											</div>
										</div>
									</div>

									{/* Expanded Detail */}
									<Show when={expandedId() === approval.id}>
										<div class="border-t border-[var(--color-border-primary)] p-4">
											{/* Required Approvers */}
											<div class="mb-4">
												<div class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider mb-2">
													Required Approvers
												</div>
												<div class="flex flex-wrap gap-2">
													<For each={parseApprovers(approval.required_approvers)}>
														{(approver) => (
															<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] ring-1 ring-inset ring-[var(--color-border-secondary)]">
																{approver}
															</span>
														)}
													</For>
												</div>
											</div>

											{/* Responses */}
											<Show when={expandedDetail() && expandedDetail()!.responses?.length}>
												<div class="mb-4">
													<div class="text-xs font-medium text-[var(--color-text-tertiary)] uppercase tracking-wider mb-2">
														Responses
													</div>
													<div class="space-y-2">
														<For each={expandedDetail()!.responses}>
															{(response: ApprovalResponse) => (
																<div class="flex items-start gap-3 p-2 rounded-lg bg-[var(--color-bg-tertiary)]">
																	<div class={`mt-0.5 w-2 h-2 rounded-full shrink-0 ${response.decision === 'approve' ? 'bg-emerald-400' : 'bg-red-400'
																		}`} />
																	<div class="flex-1 min-w-0">
																		<div class="flex items-center gap-2">
																			<span class="text-sm font-medium text-[var(--color-text-primary)]">
																				{response.approver_name || response.approver_id}
																			</span>
																			<span class={`text-xs font-medium ${response.decision === 'approve' ? 'text-emerald-400' : 'text-red-400'
																				}`}>
																				{response.decision === 'approve' ? 'Approved' : 'Rejected'}
																			</span>
																			<span class="text-xs text-[var(--color-text-tertiary)]">
																				{timeAgo(response.created_at)}
																			</span>
																		</div>
																		<Show when={response.comment}>
																			<p class="mt-1 text-xs text-[var(--color-text-secondary)]">{response.comment}</p>
																		</Show>
																	</div>
																</div>
															)}
														</For>
													</div>
												</div>
											</Show>

											{/* Action Section */}
											<Show when={approval.status === 'pending'}>
												<div class="border-t border-[var(--color-border-primary)] pt-4 mt-4">
													{/* Error */}
													<Show when={actionError()}>
														<div class="mb-3 p-2 rounded-lg bg-red-500/10 border border-red-500/20 text-xs text-red-400">
															{actionError()}
														</div>
													</Show>

													{/* Comment Input */}
													<div class="mb-3">
														<textarea
															class="w-full px-3 py-2 text-sm bg-[var(--color-bg-tertiary)] border border-[var(--color-border-primary)] rounded-lg text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)] focus:outline-none focus:ring-1 focus:ring-indigo-500 resize-none"
															rows={2}
															placeholder="Add a comment (required for rejection)..."
															value={commentText()}
															onInput={(e) => setCommentText(e.currentTarget.value)}
														/>
													</div>

													{/* Action Buttons */}
													<div class="flex items-center gap-2">
														<button
															class="px-4 py-2 rounded-lg text-sm font-medium bg-emerald-600 hover:bg-emerald-700 text-white transition-colors disabled:opacity-50"
															disabled={actionLoading() === approval.id}
															onClick={() => handleApprove(approval.id)}
														>
															{actionLoading() === approval.id ? 'Processing...' : '✓ Approve'}
														</button>
														<button
															class="px-4 py-2 rounded-lg text-sm font-medium bg-red-600 hover:bg-red-700 text-white transition-colors disabled:opacity-50"
															disabled={actionLoading() === approval.id}
															onClick={() => handleReject(approval.id)}
														>
															{actionLoading() === approval.id ? 'Processing...' : '✕ Reject'}
														</button>
														<button
															class="px-4 py-2 rounded-lg text-sm font-medium bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors disabled:opacity-50"
															disabled={actionLoading() === approval.id}
															onClick={() => handleCancel(approval.id)}
														>
															Cancel Request
														</button>
													</div>
												</div>
											</Show>

											{/* Info for resolved approvals */}
											<Show when={approval.status !== 'pending' && approval.resolved_at}>
												<div class="text-xs text-[var(--color-text-tertiary)] mt-2">
													Resolved {timeAgo(approval.resolved_at!)}
												</div>
											</Show>
										</div>
									</Show>
								</div>
							)}
						</For>
					</div>
				</Show>
			</Show>
		</div>
	);
};

export default ApprovalsPage;
