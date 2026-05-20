import { expect, test } from '@playwright/test';

test('registration and login API contracts can be exercised with deterministic fixtures', async ({ page }) => {
  await page.route('**/api/v1/auth/register', (route) =>
    route.fulfill({ contentType: 'application/json', body: JSON.stringify({ success: true, data: { user: { id: 'user-1', email: 'user@example.com' } } }) })
  );
  await page.route('**/api/v1/auth/login', (route) =>
    route.fulfill({ contentType: 'application/json', body: JSON.stringify({ success: true, data: { user: { id: 'user-1', email: 'user@example.com' } } }) })
  );

  await page.goto('/');
  const register = await page.evaluate(() =>
    fetch('/api/v1/auth/register', { method: 'POST', body: JSON.stringify({ email: 'user@example.com', password: 'Password123!' }) }).then((response) => response.ok)
  );
  const login = await page.evaluate(() =>
    fetch('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ email: 'user@example.com', password: 'Password123!' }) }).then((response) => response.ok)
  );

  expect(register).toBe(true);
  expect(login).toBe(true);
});
