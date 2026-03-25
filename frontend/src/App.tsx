import { Router, Route, Navigate } from '@solidjs/router';
import { createSignal, onMount, Show, type Component } from 'solid-js';
import { authStore } from './stores/auth';
import ToastContainer from './components/ui/Toast';

// Layouts
import AppLayout from './components/layout/AppLayout';
import AuthLayout from './components/layout/AuthLayout';

// Auth pages
import LoginPage from './pages/auth/LoginPage';
import RegisterPage from './pages/auth/RegisterPage';
import OAuthCallback from './pages/auth/OAuthCallback';

// App pages
import DashboardPage from './pages/dashboard/DashboardPage';
import ProjectsPage from './pages/projects/ProjectsPage';
import ProjectDetailPage from './pages/projects/ProjectDetailPage';
import ImportProjectPage from './pages/projects/ImportProjectPage';
import PipelinesPage from './pages/pipelines/PipelinesPage';
import PipelineDetailPage from './pages/pipelines/PipelineDetailPage';
import RunDetailPage from './pages/runs/RunDetailPage';
import RunsPage from './pages/runs/RunsPage';
import AgentsPage from './pages/agents/AgentsPage';
import SettingsPage from './pages/settings/SettingsPage';
import AdminPage from './pages/admin/AdminPage';

// ---------------------------------------------------------------------------
// Auth guard — redirects to /auth/login if not authenticated
// ---------------------------------------------------------------------------
const RequireAuth: Component<{ children?: any }> = (props) => {
  return (
    <Show
      when={authStore.isAuthenticated()}
      fallback={<Navigate href="/auth/login" />}
    >
      <AppLayout>{props.children}</AppLayout>
    </Show>
  );
};

// ---------------------------------------------------------------------------
// Guest guard — redirects to / if already authenticated
// ---------------------------------------------------------------------------
const GuestOnly: Component<{ children?: any }> = (props) => {
  return (
    <Show
      when={!authStore.isAuthenticated()}
      fallback={<Navigate href="/" />}
    >
      <AuthLayout>{props.children}</AuthLayout>
    </Show>
  );
};

// ---------------------------------------------------------------------------
// App
// ---------------------------------------------------------------------------
const App: Component = () => {
  const [ready, setReady] = createSignal(false);

  onMount(async () => {
    await authStore.initialize();
    setReady(true);
  });

  return (
    <>
      <Show
        when={ready()}
        fallback={
          <div class="min-h-screen bg-[var(--color-bg-primary)] flex items-center justify-center">
            <div class="flex flex-col items-center gap-4">
              <div class="w-10 h-10 rounded-lg bg-indigo-600 flex items-center justify-center">
                <svg class="w-6 h-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <div class="flex items-center gap-2 text-sm text-[var(--color-text-tertiary)]">
                <svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                Loading...
              </div>
            </div>
          </div>
        }
      >
        <Router>
          {/* Auth routes — guest only */}
          <Route path="/auth" component={GuestOnly}>
            <Route path="/login" component={LoginPage} />
            <Route path="/register" component={RegisterPage} />
            <Route path="/oauth/callback" component={OAuthCallback} />
            <Route path="/*" component={() => <Navigate href="/auth/login" />} />
          </Route>

          {/* App routes — authenticated */}
          <Route path="/" component={RequireAuth}>
            <Route path="/" component={DashboardPage} />
            <Route path="/projects" component={ProjectsPage} />
            <Route path="/projects/import" component={ImportProjectPage} />
            <Route path="/projects/:id" component={ProjectDetailPage} />
            <Route path="/projects/:id/pipelines" component={PipelinesPage} />
            <Route path="/projects/:id/pipelines/:pid" component={PipelineDetailPage} />
            <Route path="/projects/:id/pipelines/:pid/runs/:rid" component={RunDetailPage} />
            <Route path="/runs" component={RunsPage} />
            <Route path="/agents" component={AgentsPage} />
            <Route path="/settings/*rest" component={SettingsPage} />
            <Route path="/admin/*rest" component={AdminPage} />
          </Route>

          {/* Catch-all */}
          <Route path="/*" component={() => <Navigate href="/" />} />
        </Router>
      </Show>

      <ToastContainer />
    </>
  );
};

export default App;
