import { expect, test } from '@playwright/test';
import { installAdminFixtures } from './fixtures';

test('admin import flow searches provider candidates and imports a selected item', async ({ page }) => {
  await installAdminFixtures(page);
  await page.goto('/admin');

  await page.getByLabel('External food search').fill('tofu');
  await page.getByRole('button', { name: 'Search' }).click();
  await page.getByRole('button', { name: /Select Provider tofu/ }).click();
  await expect(page.getByText('123')).toBeVisible();
  await page.getByRole('button', { name: 'Import', exact: true }).click();
});
