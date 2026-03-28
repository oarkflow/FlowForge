import type { Component, JSX } from 'solid-js';
import { Show } from 'solid-js';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
export interface PipelineTemplate {
	id: string;
	name: string;
	description: string;
	category: string;
	icon: string;
	tags: string[];
	yaml: string;
	author: string;
	downloads: number;
	isOfficial: boolean;
}

interface TemplateCardProps {
	template: PipelineTemplate;
	onPreview: (template: PipelineTemplate) => void;
	onUse: (template: PipelineTemplate) => void;
}

// ---------------------------------------------------------------------------
// Category colors
// ---------------------------------------------------------------------------
const categoryColors: Record<string, { bg: string; text: string; border: string }> = {
	ci: { bg: 'bg-blue-500/10', text: 'text-blue-400', border: 'border-blue-500/20' },
	cd: { bg: 'bg-emerald-500/10', text: 'text-emerald-400', border: 'border-emerald-500/20' },
	security: { bg: 'bg-red-500/10', text: 'text-red-400', border: 'border-red-500/20' },
	testing: { bg: 'bg-amber-500/10', text: 'text-amber-400', border: 'border-amber-500/20' },
	docker: { bg: 'bg-cyan-500/10', text: 'text-cyan-400', border: 'border-cyan-500/20' },
	kubernetes: { bg: 'bg-violet-500/10', text: 'text-violet-400', border: 'border-violet-500/20' },
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
const TemplateCard: Component<TemplateCardProps> = (props) => {
	const t = () => props.template;
	const colors = () => categoryColors[t().category] || categoryColors.ci;

	return (
		<div class="group flex flex-col rounded-xl border border-[var(--color-border-primary)] bg-[var(--color-bg-secondary)] hover:border-[var(--color-border-secondary)] transition-all hover:shadow-lg hover:shadow-black/20">
			{/* Header */}
			<div class="flex items-start gap-3 p-4 pb-3">
				{/* Icon */}
				<div class={`w-10 h-10 rounded-lg ${colors().bg} ${colors().border} border flex items-center justify-center flex-shrink-0`}>
					<span class="text-lg">{t().icon}</span>
				</div>

				<div class="flex-1 min-w-0">
					<div class="flex items-center gap-2">
						<h3 class="text-sm font-semibold text-[var(--color-text-primary)] truncate">{t().name}</h3>
						<Show when={t().isOfficial}>
							<span class="px-1.5 py-0.5 text-[9px] font-medium rounded bg-indigo-500/20 text-indigo-400 border border-indigo-500/30 flex-shrink-0">
								Official
							</span>
						</Show>
					</div>
					<p class="text-xs text-[var(--color-text-tertiary)] mt-0.5 line-clamp-2">{t().description}</p>
				</div>
			</div>

			{/* Tags */}
			<div class="px-4 pb-3 flex flex-wrap gap-1">
				<span class={`px-1.5 py-0.5 text-[10px] font-medium rounded ${colors().bg} ${colors().text} ${colors().border} border`}>
					{t().category}
				</span>
				{t().tags.slice(0, 3).map(tag => (
					<span class="px-1.5 py-0.5 text-[10px] rounded bg-[var(--color-bg-tertiary)] text-[var(--color-text-tertiary)] border border-[var(--color-border-primary)]">
						{tag}
					</span>
				))}
			</div>

			{/* Footer */}
			<div class="mt-auto flex items-center justify-between px-4 py-3 border-t border-[var(--color-border-primary)]">
				<div class="flex items-center gap-3 text-xs text-[var(--color-text-tertiary)]">
					<span>{t().author}</span>
					<Show when={t().downloads > 0}>
						<span class="flex items-center gap-1">
							<svg class="w-3 h-3" viewBox="0 0 20 20" fill="currentColor">
								<path d="M10.75 2.75a.75.75 0 00-1.5 0v8.614L6.295 8.235a.75.75 0 10-1.09 1.03l4.25 4.5a.75.75 0 001.09 0l4.25-4.5a.75.75 0 00-1.09-1.03l-2.955 3.129V2.75z" />
								<path d="M3.5 12.75a.75.75 0 00-1.5 0v2.5A2.75 2.75 0 004.75 18h10.5A2.75 2.75 0 0018 15.25v-2.5a.75.75 0 00-1.5 0v2.5c0 .69-.56 1.25-1.25 1.25H4.75c-.69 0-1.25-.56-1.25-1.25v-2.5z" />
							</svg>
							{t().downloads}
						</span>
					</Show>
				</div>
				<div class="flex items-center gap-2">
					<button
						onClick={(e) => { e.stopPropagation(); props.onPreview(t()); }}
						class="px-2.5 py-1 text-xs rounded-md bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] border border-[var(--color-border-primary)] transition-colors cursor-pointer"
					>
						Preview
					</button>
					<button
						onClick={(e) => { e.stopPropagation(); props.onUse(t()); }}
						class="px-2.5 py-1 text-xs rounded-md bg-indigo-600 text-white hover:bg-indigo-500 transition-colors cursor-pointer"
					>
						Use Template
					</button>
				</div>
			</div>
		</div>
	);
};

export default TemplateCard;
