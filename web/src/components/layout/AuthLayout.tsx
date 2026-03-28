import type { Component, JSX } from 'solid-js';

interface AuthLayoutProps {
  children?: JSX.Element;
}

const AuthLayout: Component<AuthLayoutProps> = (props) => {
  return (
    <div class="min-h-screen bg-[var(--color-bg-primary)] flex">
      {/* Left side — branding */}
      <div class="hidden lg:flex lg:w-1/2 flex-col justify-between p-12 bg-[var(--color-bg-secondary)] border-r border-[var(--color-border-primary)]">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-indigo-600 flex items-center justify-center">
            <svg class="w-6 h-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="text-xl font-bold text-[var(--color-text-primary)] tracking-tight">
            FlowForge
          </span>
        </div>

        <div class="max-w-md">
          <h1 class="text-4xl font-bold text-[var(--color-text-primary)] leading-tight tracking-tight">
            Build, test, and deploy with confidence.
          </h1>
          <p class="mt-4 text-lg text-[var(--color-text-secondary)] leading-relaxed">
            A self-hosted CI/CD platform that connects your repositories, automates your pipelines, and scales with your team.
          </p>
        </div>

        <p class="text-sm text-[var(--color-text-tertiary)]">
          FlowForge v1.0
        </p>
      </div>

      {/* Right side — auth form */}
      <div class="flex-1 flex items-center justify-center p-8">
        <div class="w-full max-w-md">
          {/* Mobile logo */}
          <div class="lg:hidden flex items-center gap-3 mb-8 justify-center">
            <div class="w-10 h-10 rounded-xl bg-indigo-600 flex items-center justify-center">
              <svg class="w-6 h-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <span class="text-xl font-bold text-[var(--color-text-primary)] tracking-tight">
              FlowForge
            </span>
          </div>

          {props.children}
        </div>
      </div>
    </div>
  );
};

export default AuthLayout;
