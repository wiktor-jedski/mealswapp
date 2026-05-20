import { expect, test } from '@playwright/test';
import { installApiFixtures } from './fixtures';

test.beforeEach(async ({ page }) => {
  await installApiFixtures(page);
});

test('basic search renders deterministic results', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Search food').fill('tofu');
  await expect(page.getByRole('heading', { name: 'Tofu' })).toBeVisible();
  await expect(page.getByText('10 g')).toBeVisible();
});

test('paid replacement mode is gated for free users', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('button', { name: /Replacement/ })).toBeDisabled();
  await expect(page.getByText('Upgrade to unlock replacement and diet modes.')).toBeVisible();
});

test('saved, history, export, and account deletion controls are reachable', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('button', { name: 'Saved searches' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'History' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Favorites' })).toBeVisible();

  await page.getByRole('button', { name: 'Settings' }).first().click();
  await expect(page.getByRole('button', { name: 'Export JSON' })).toBeVisible();
  await page.getByLabel('Confirm deletion').fill('DELETE');
  await expect(page.getByRole('button', { name: 'Delete account' })).toBeEnabled();
});
