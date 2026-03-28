import { Component, JSX, ErrorBoundary as SolidErrorBoundary, createSignal } from 'solid-js';

interface ErrorBoundaryProps {
  children: JSX.Element;
  fallback?: (error: Error, reset: () => void) => JSX.Element;
}

/**
 * ErrorBoundary - Catches rendering errors and shows a friendly fallback UI.
 */
export const ErrorBoundary: Component<ErrorBoundaryProps> = (props) => {
  return (
    <SolidErrorBoundary
      fallback={(err, reset) => {
        if (props.fallback) {
          return props.fallback(err instanceof Error ? err : new Error(String(err)), reset);
        }
        return <DefaultErrorFallback error={err} reset={reset} />;
      }}
    >
      {props.children}
    </SolidErrorBoundary>
  );
};

const DefaultErrorFallback: Component<{ error: any; reset: () => void }> = (props) => {
  const [showDetails, setShowDetails] = createSignal(false);

  const errorMessage = () => {
    if (props.error instanceof Error) return props.error.message;
    return String(props.error);
  };

  const errorStack = () => {
    if (props.error instanceof Error) return props.error.stack || '';
    return '';
  };

  return (
    <div class="flex items-center justify-center min-h-[200px] p-6">
      <div class="max-w-md w-full rounded-lg p-6 text-center" style="background: var(--bg-secondary); border: 1px solid var(--border-primary);">
        <div class="text-3xl mb-3">⚠️</div>
        <h3 class="text-lg font-semibold mb-2" style="color: var(--text-primary);">
          Something went wrong
        </h3>
        <p class="text-sm mb-4" style="color: var(--text-secondary);">
          {errorMessage()}
        </p>

        <div class="flex items-center justify-center gap-3 mb-4">
          <button
            onClick={() => props.reset()}
            class="px-4 py-2 rounded-md text-sm font-medium"
            style="background: var(--accent-primary); color: white;"
          >
            Try Again
          </button>
          <button
            onClick={() => window.location.reload()}
            class="px-4 py-2 rounded-md text-sm"
            style="background: var(--bg-tertiary); color: var(--text-primary); border: 1px solid var(--border-primary);"
          >
            Reload Page
          </button>
        </div>

        <button
          onClick={() => setShowDetails(!showDetails())}
          class="text-xs underline"
          style="color: var(--text-tertiary);"
        >
          {showDetails() ? 'Hide' : 'Show'} details
        </button>

        {showDetails() && (
          <pre
            class="mt-3 p-3 rounded text-left text-xs overflow-x-auto max-h-48"
            style="background: var(--bg-tertiary); color: var(--text-secondary);"
          >
            {errorStack()}
          </pre>
        )}
      </div>
    </div>
  );
};
