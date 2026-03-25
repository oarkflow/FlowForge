import type { Component, JSX } from 'solid-js';
import Sidebar from './Sidebar';
import TopBar from './TopBar';

interface AppLayoutProps {
  children?: JSX.Element;
}

const AppLayout: Component<AppLayoutProps> = (props) => {
  return (
    <div class="min-h-screen bg-[var(--color-bg-primary)]">
      <Sidebar />
      <TopBar />

      {/* Main content area */}
      <main
        class="pt-[var(--topbar-height)] min-h-screen"
        style={{ "margin-left": "var(--sidebar-width)" }}
      >
        <div class="p-6 max-w-[1400px] mx-auto">
          {props.children}
        </div>
      </main>
    </div>
  );
};

export default AppLayout;
