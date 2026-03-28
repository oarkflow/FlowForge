import type { Component, JSX } from 'solid-js';
import { createSignal, onMount, onCleanup } from 'solid-js';
import Sidebar from './Sidebar';
import TopBar from './TopBar';
import CommandPalette from '../search/CommandPalette';

interface AppLayoutProps {
	children?: JSX.Element;
}

const AppLayout: Component<AppLayoutProps> = (props) => {
	const [searchOpen, setSearchOpen] = createSignal(false);

	const handleKeyDown = (e: KeyboardEvent) => {
		// Cmd+K (Mac) or Ctrl+K (Windows/Linux) to open search
		if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
			e.preventDefault();
			setSearchOpen(true);
		}
	};

	onMount(() => {
		document.addEventListener('keydown', handleKeyDown);
	});

	onCleanup(() => {
		document.removeEventListener('keydown', handleKeyDown);
	});

	return (
		<div class="min-h-screen bg-[var(--color-bg-primary)]">
			<Sidebar />
			<TopBar onOpenSearch={() => setSearchOpen(true)} />
			<CommandPalette isOpen={searchOpen()} onClose={() => setSearchOpen(false)} />

			{/* Main content area */}
			<main
				class="pt-[var(--topbar-height)] min-h-screen"
				style={{ "margin-left": "var(--sidebar-width)" }}
			>
				<div class="p-6 mx-auto">
					{props.children}
				</div>
			</main>
		</div>
	);
};

export default AppLayout;
