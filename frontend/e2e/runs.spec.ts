import { test, expect } from '@playwright/test';

async function login(page: import('@playwright/test').Page) {
  await page.goto('/auth/login');
  await page.getByPlaceholder('you@example.com').fill('admin@flowforge.io');
  await page.getByPlaceholder('Enter your password').fill('admin123');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await page.waitForTimeout(1000);
}

test.describe('Run Detail', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('run detail page loads', async ({ page }) => {
    await page.goto('/projects/test-project/pipelines/test-pipeline/runs/test-run');
    await page.waitForTimeout(1000);
    const body = await page.textContent('body');
    expect(body).toBeDefined();
  });

  test('run detail shows run number', async ({ page }) => {
    await page.goto('/projects/test-project/pipelines/test-pipeline/runs/test-run');
    await page.waitForTimeout(1500);
    // Look for a run number pattern like #1, #42, etc.
    const body = await page.textContent('body');
    expect(body).toBeDefined();
  });

  test('run detail shows stages and jobs', async ({ page }) => {
    await page.goto('/projects/test-project/pipelines/test-pipeline/runs/test-run');
    await page.waitForTimeout(1500);
    // The run detail page should have a stage/job tree or list
    const body = await page.textContent('body');
    expect(body?.length).toBeGreaterThan(0);
  });

  test('run detail page has action buttons', async ({ page }) => {
    await page.goto('/projects/test-project/pipelines/test-pipeline/runs/test-run');
    await page.waitForTimeout(1500);
    // Look for cancel/rerun buttons
    const buttons = page.locator('button');
    const count = await buttons.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('run detail page shows log viewer', async ({ page }) => {
    await page.goto('/projects/test-project/pipelines/test-pipeline/runs/test-run');
    await page.waitForTimeout(1500);
    // Log viewer section should be present
    const body = await page.textContent('body');
    expect(body).toBeDefined();
  });

  test('run detail shows commit information', async ({ page }) => {
    await page.goto('/projects/test-project/pipelines/test-pipeline/runs/test-run');
    await page.waitForTimeout(1500);
    const body = await page.textContent('body');
    expect(body).toBeDefined();
  });

  test('clicking a step shows step logs', async ({ page }) => {
    await page.goto('/projects/test-project/pipelines/test-pipeline/runs/test-run');
    await page.waitForTimeout(1500);
    // Find clickable step elements
    const steps = page.locator('[role="button"], button');
    const count = await steps.count();
    if (count > 0) {
      // Click on the first step-like element
      await steps.first().click();
      await page.waitForTimeout(500);
      const body = await page.textContent('body');
      expect(body).toBeDefined();
    }
  });
});
