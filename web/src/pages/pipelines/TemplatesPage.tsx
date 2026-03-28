import type { Component } from 'solid-js';
import { createSignal, createMemo, For, Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import PageContainer from '../../components/layout/PageContainer';
import Card from '../../components/ui/Card';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';
import Modal from '../../components/ui/Modal';
import TemplateCard from '../../components/pipeline/TemplateCard';
import type { PipelineTemplate } from '../../components/pipeline/TemplateCard';
import { toast } from '../../components/ui/Toast';

// ---------------------------------------------------------------------------
// Built-in templates
// ---------------------------------------------------------------------------
const BUILTIN_TEMPLATES: PipelineTemplate[] = [
	{
		id: 'go-ci',
		name: 'Go CI Pipeline',
		description: 'Build, test, and lint a Go project with race detection and coverage.',
		category: 'ci',
		icon: '🐹',
		tags: ['go', 'golang', 'testing'],
		author: 'FlowForge',
		downloads: 1240,
		isOfficial: true,
		yaml: `version: "1"
name: "Go CI"

on:
  push:
    branches: ["main", "develop"]
  pull_request:
    types: [opened, synchronize]

stages:
  - test
  - build

jobs:
  test:
    stage: test
    executor: docker
    image: golang:1.22-alpine
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Run tests
        run: go test ./... -race -coverprofile=coverage.out
      - name: Upload coverage
        uses: flowforge/upload-artifact@v1
        with:
          name: coverage
          path: coverage.out

  lint:
    stage: test
    executor: docker
    image: golangci/golangci-lint:latest
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Lint
        run: golangci-lint run ./...

  build:
    stage: build
    needs: [test, lint]
    executor: docker
    image: golang:1.22-alpine
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Build
        run: CGO_ENABLED=0 go build -o dist/app ./cmd/server
`,
	},
	{
		id: 'node-ci',
		name: 'Node.js CI Pipeline',
		description: 'Install dependencies, lint, test, and build a Node.js/TypeScript project.',
		category: 'ci',
		icon: '🟢',
		tags: ['node', 'npm', 'typescript'],
		author: 'FlowForge',
		downloads: 2100,
		isOfficial: true,
		yaml: `version: "1"
name: "Node.js CI"

on:
  push:
    branches: ["main"]
  pull_request:
    types: [opened, synchronize]

stages:
  - setup
  - test
  - build

jobs:
  install:
    stage: setup
    executor: docker
    image: node:20-alpine
    cache:
      - key: node-modules-\${{ hash("package-lock.json") }}
        paths: [node_modules]
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Install
        run: npm ci

  lint:
    stage: test
    needs: [install]
    executor: docker
    image: node:20-alpine
    steps:
      - name: Lint
        run: npm run lint

  test:
    stage: test
    needs: [install]
    executor: docker
    image: node:20-alpine
    steps:
      - name: Test
        run: npm test -- --coverage

  build:
    stage: build
    needs: [lint, test]
    executor: docker
    image: node:20-alpine
    steps:
      - name: Build
        run: npm run build
`,
	},
	{
		id: 'python-ci',
		name: 'Python CI Pipeline',
		description: 'Test and lint a Python project with pytest, mypy, and ruff.',
		category: 'ci',
		icon: '🐍',
		tags: ['python', 'pytest', 'django'],
		author: 'FlowForge',
		downloads: 980,
		isOfficial: true,
		yaml: `version: "1"
name: "Python CI"

on:
  push:
    branches: ["main"]
  pull_request:
    types: [opened, synchronize]

stages:
  - test
  - quality

jobs:
  test:
    stage: test
    executor: docker
    image: python:3.12-slim
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Install deps
        run: pip install -r requirements.txt -r requirements-dev.txt
      - name: Run tests
        run: pytest --junitxml=results.xml --cov=src

  lint:
    stage: quality
    executor: docker
    image: python:3.12-slim
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Install
        run: pip install ruff mypy
      - name: Ruff
        run: ruff check .
      - name: Mypy
        run: mypy src/
`,
	},
	{
		id: 'docker-build',
		name: 'Docker Build & Push',
		description: 'Build a Docker image and push it to a container registry.',
		category: 'docker',
		icon: '🐳',
		tags: ['docker', 'registry', 'container'],
		author: 'FlowForge',
		downloads: 1560,
		isOfficial: true,
		yaml: `version: "1"
name: "Docker Build & Push"

on:
  push:
    branches: ["main"]
    tags: ["v*"]

stages:
  - build
  - push

jobs:
  build:
    stage: build
    executor: docker
    image: docker:26-dind
    privileged: true
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Build image
        run: docker build -t \${{ env.IMAGE_NAME }}:\${{ git.sha }} .

  push:
    stage: push
    needs: [build]
    executor: docker
    image: docker:26-dind
    privileged: true
    when: \${{ git.branch == 'main' || git.tag }}
    steps:
      - name: Push to registry
        uses: flowforge/docker-build-push@v1
        with:
          registry: ghcr.io
          image: \${{ secrets.IMAGE_NAME }}
          tags: |
            latest
            \${{ git.sha }}
          push: true
`,
	},
	{
		id: 'k8s-deploy',
		name: 'Kubernetes Deployment',
		description: 'Deploy to Kubernetes using Helm with staging and production environments.',
		category: 'kubernetes',
		icon: '☸️',
		tags: ['kubernetes', 'helm', 'deploy'],
		author: 'FlowForge',
		downloads: 720,
		isOfficial: true,
		yaml: `version: "1"
name: "K8s Deploy"

on:
  push:
    branches: ["main"]
    tags: ["v*"]

stages:
  - deploy-staging
  - deploy-production

jobs:
  staging:
    stage: deploy-staging
    environment: staging
    executor: docker
    image: alpine/helm:latest
    steps:
      - name: Deploy to staging
        uses: flowforge/helm-deploy@v1
        with:
          chart: ./deploy/helm/app
          release: app-staging
          namespace: staging
          values: |
            image.tag: \${{ git.sha }}

  production:
    stage: deploy-production
    needs: [staging]
    environment: production
    approval_required: true
    when: \${{ git.tag =~ /^v\\d+/ }}
    executor: docker
    image: alpine/helm:latest
    steps:
      - name: Deploy to production
        uses: flowforge/helm-deploy@v1
        with:
          chart: ./deploy/helm/app
          release: app-production
          namespace: production
`,
	},
	{
		id: 'security-scan',
		name: 'Security Scanning',
		description: 'Run container vulnerability scanning and SAST with Trivy and Semgrep.',
		category: 'security',
		icon: '🔒',
		tags: ['security', 'trivy', 'sast'],
		author: 'FlowForge',
		downloads: 540,
		isOfficial: true,
		yaml: `version: "1"
name: "Security Scan"

on:
  push:
    branches: ["main"]
  schedule:
    - cron: "0 6 * * 1"
      timezone: "UTC"

stages:
  - scan

jobs:
  trivy:
    stage: scan
    executor: docker
    image: aquasec/trivy:latest
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Scan filesystem
        run: trivy fs --severity HIGH,CRITICAL --exit-code 1 .

  semgrep:
    stage: scan
    executor: docker
    image: returntocorp/semgrep:latest
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: SAST scan
        run: semgrep --config auto .
`,
	},
	{
		id: 'e2e-testing',
		name: 'E2E Testing with Playwright',
		description: 'Run end-to-end tests with Playwright across multiple browsers.',
		category: 'testing',
		icon: '🎭',
		tags: ['playwright', 'e2e', 'browser'],
		author: 'FlowForge',
		downloads: 430,
		isOfficial: true,
		yaml: `version: "1"
name: "E2E Tests"

on:
  pull_request:
    types: [opened, synchronize]

stages:
  - test

jobs:
  e2e:
    stage: test
    executor: docker
    image: mcr.microsoft.com/playwright:latest
    matrix:
      browser: ["chromium", "firefox", "webkit"]
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Install deps
        run: npm ci
      - name: Run Playwright tests
        run: npx playwright test --project=\${{ matrix.browser }}
      - name: Upload report
        uses: flowforge/upload-artifact@v1
        with:
          name: playwright-report-\${{ matrix.browser }}
          path: playwright-report/
`,
	},
	{
		id: 'cd-multi-env',
		name: 'Multi-Environment CD',
		description: 'Promote deployments through dev, staging, and production with approval gates.',
		category: 'cd',
		icon: '🚀',
		tags: ['deploy', 'promotion', 'environments'],
		author: 'FlowForge',
		downloads: 890,
		isOfficial: true,
		yaml: `version: "1"
name: "Multi-Env CD"

on:
  push:
    branches: ["main"]

stages:
  - build
  - deploy-dev
  - deploy-staging
  - deploy-production

jobs:
  build:
    stage: build
    executor: docker
    image: node:20-alpine
    steps:
      - name: Build
        run: npm ci && npm run build

  deploy-dev:
    stage: deploy-dev
    needs: [build]
    environment: development
    steps:
      - name: Deploy to dev
        run: echo "Deploying to dev..."

  deploy-staging:
    stage: deploy-staging
    needs: [deploy-dev]
    environment: staging
    steps:
      - name: Deploy to staging
        run: echo "Deploying to staging..."

  deploy-production:
    stage: deploy-production
    needs: [deploy-staging]
    environment: production
    approval_required: true
    steps:
      - name: Deploy to production
        run: echo "Deploying to production..."
`,
	},
];

// ---------------------------------------------------------------------------
// Templates Page
// ---------------------------------------------------------------------------
const TemplatesPage: Component = () => {
	const navigate = useNavigate();
	const [search, setSearch] = createSignal('');
	const [category, setCategory] = createSignal<string>('all');
	const [previewTemplate, setPreviewTemplate] = createSignal<PipelineTemplate | null>(null);

	const categories = ['all', 'ci', 'cd', 'security', 'testing', 'docker', 'kubernetes'];

	const filtered = createMemo(() => {
		let templates = BUILTIN_TEMPLATES;
		const cat = category();
		if (cat !== 'all') {
			templates = templates.filter(t => t.category === cat);
		}
		const q = search().toLowerCase().trim();
		if (q) {
			templates = templates.filter(t =>
				t.name.toLowerCase().includes(q) ||
				t.description.toLowerCase().includes(q) ||
				t.tags.some(tag => tag.includes(q))
			);
		}
		return templates;
	});

	const handleUse = (template: PipelineTemplate) => {
		// Store template in sessionStorage and redirect to project creation
		sessionStorage.setItem('flowforge_template_yaml', template.yaml);
		sessionStorage.setItem('flowforge_template_name', template.name);
		toast.success(`Template "${template.name}" selected. Create a project to use it.`);
		navigate('/projects/import');
	};

	return (
		<PageContainer
			title="Pipeline Templates"
			description="Start with a pre-built pipeline configuration"
			actions={
				<Button size="sm" variant="outline" onClick={() => navigate(-1)}>Back</Button>
			}
		>
			{/* Search and filters */}
			<div class="flex items-center gap-4 mb-6">
				<div class="flex-1 max-w-md">
					<Input
						placeholder="Search templates..."
						value={search()}
						onInput={(e) => setSearch(e.currentTarget.value)}
						icon={<svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clip-rule="evenodd" /></svg>}
					/>
				</div>
				<div class="flex gap-1 bg-[var(--color-bg-tertiary)] p-1 rounded-lg">
					<For each={categories}>
						{(cat) => (
							<button
								class={`px-3 py-1.5 text-xs font-medium rounded-md capitalize transition-colors cursor-pointer ${
									category() === cat
										? 'bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] shadow-sm'
										: 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]'
								}`}
								onClick={() => setCategory(cat)}
							>
								{cat}
							</button>
						)}
					</For>
				</div>
			</div>

			{/* Template grid */}
			<Show when={filtered().length > 0} fallback={
				<div class="flex flex-col items-center justify-center py-16 text-[var(--color-text-tertiary)]">
					<svg class="w-12 h-12 mb-3 opacity-50" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
						<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
					</svg>
					<p class="text-sm">No templates match your search.</p>
				</div>
			}>
				<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
					<For each={filtered()}>
						{(template) => (
							<TemplateCard
								template={template}
								onPreview={(t) => setPreviewTemplate(t)}
								onUse={handleUse}
							/>
						)}
					</For>
				</div>
			</Show>

			{/* Preview Modal */}
			<Show when={previewTemplate()}>
				<Modal
					open={!!previewTemplate()}
					onClose={() => setPreviewTemplate(null)}
					title={previewTemplate()!.name}
					description={previewTemplate()!.description}
					footer={
						<>
							<Button variant="ghost" onClick={() => setPreviewTemplate(null)}>Close</Button>
							<Button onClick={() => { handleUse(previewTemplate()!); setPreviewTemplate(null); }}>Use Template</Button>
						</>
					}
				>
					<div class="rounded-lg overflow-hidden border border-[var(--color-border-primary)]">
						<div class="px-3 py-2 bg-[#161b22] border-b border-[var(--color-border-primary)] flex items-center gap-2">
							<span class="text-xs text-[var(--color-text-tertiary)]">flowforge.yml</span>
						</div>
						<pre class="text-xs font-mono text-[#c9d1d9] bg-[#0d1117] p-4 overflow-auto max-h-[50vh] leading-relaxed">
							{previewTemplate()!.yaml}
						</pre>
					</div>
				</Modal>
			</Show>
		</PageContainer>
	);
};

export default TemplatesPage;
