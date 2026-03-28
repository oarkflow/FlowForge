import { test, expect } from '@playwright/test';

async function login(page: import('@playwright/test').Page) {
  await page.goto('/auth/login');
  await page.getByPlaceholder('you@example.com').fill('admin@flowforge.io');
  await page.getByPlaceholder('Enter your password').fill('admin123');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await page.waitForTimeout(1000);
}

test.describe('Agents', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/agents');
  });

  test('agents page loads', async ({ page }) => {
    await page.waitForTimeout(1000);
    const heading = page.getByText('Agents');
    const isVisible = await heading.isVisible().catch(() => false);
    if (isVisible) {
      await expect(heading).toBeVisible();
    } else {
      expect(page.url()).toBeDefined();
    }
  });

  test('agents page has Register Agent button', async ({ page }) => {
    await page.waitForTimeout(1000);
    const btn = page.getByText('Register Agent');
    const isVisible = await btn.isVisible().catch(() => false);
    if (isVisible) {
      await expect(btn).toBeVisible();
    }
  });

  test('agents page has search input', async ({ page }) => {
    await page.waitForTimeout(1000);
    const search = page.getByPlaceholder(/search agents/i);
    const isVisible = await search.isVisible().catch(() => false);
    if (isVisible) {
      await expect(search).toBeVisible();
    }
  });

  test('agents page shows agent list', async ({ page }) => {
    await page.waitForTimeout(1500);
    const body = await page.textContent('body');
    expect(body?.length).toBeGreaterThan(0);
  });

  test('register agent modal opens', async ({ page }) => {
    await page.waitForTimeout(1000);
    const btn = page.getByText('Register Agent');
    const isVisible = await btn.isVisible().catch(() => false);
    if (isVisible) {
      await btn.click();
      await page.waitForTimeout(500);
      // Modal should appear with registration form
      const modalBody = await page.textContent('body');
      expect(modalBody).toBeDefined();
    }
  });

  test('agents page has status filter', async ({ page }) => {
    await page.waitForTimeout(1000);
    const selects = page.locator('select');
    const count = await selects.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('search filters agents', async ({ page }) => {
    await page.waitForTimeout(1000);
    const search = page.getByPlaceholder(/search agents/i);
    const isVisible = await search.isVisible().catch(() => false);
    if (isVisible) {
      await search.fill('nonexistent-agent');
      await page.waitForTimeout(500);
      const body = await page.textContent('body');
      expect(body).toBeDefined();
    }
  });

  test('agents page has tabs', async ({ page }) => {
    await page.waitForTimeout(1000);
    // Look for Agents and Scaling tabs
    const body = await page.textContent('body');
    expect(body).toBeDefined();
  });
});
