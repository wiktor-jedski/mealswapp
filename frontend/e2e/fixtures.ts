import type { Page, Route } from '@playwright/test';

export async function installApiFixtures(page: Page) {
  await page.route('**/api/v1/subscription/entitlement', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          userId: 'user-1',
          tier: 'free',
          status: 'active',
          searchLimitPer24h: 3,
          allowedModes: ['single'],
          allowedFeatures: ['single']
        }
      })
    })
  );
  await page.route('**/api/v1/autocomplete**', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: [{ itemId: 'food-1', label: 'Tofu', exactMatch: true, levenshteinDistance: 0, length: 4, rank: 1 }]
      })
    })
  );
  await page.route('**/api/v1/search**', async (route) => fulfillSearch(route));
  await page.route('**/api/v1/profile/export**', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ success: true, data: { ok: true } }) }));
  await page.route('**/api/v1/profile', (route) => {
    if (route.request().method() === 'DELETE') {
      return route.fulfill({ contentType: 'application/json', body: JSON.stringify({ success: true, data: { status: 'deleted' } }) });
    }
    return route.continue();
  });
}

export async function installAdminFixtures(page: Page) {
  await page.route('**/api/v1/admin/external-search**', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          candidates: [{ provider: 'usda', externalId: '123', name: 'Provider tofu', macrosPer100: { protein: 12, carbs: 2, fat: 5 }, raw: {} }],
          warnings: []
        }
      })
    })
  );
  await page.route('**/api/v1/admin/items', (route) => {
    if (route.request().method() === 'POST') {
      return route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { id: 'food-1', name: 'Provider tofu', tags: [], macros: { protein: 12, carbs: 2, fat: 5, unitBasis: '100g' } } })
      });
    }
    return route.fulfill({ contentType: 'application/json', body: JSON.stringify({ success: true, data: { items: [], totalCount: 0, page: 1, pageSize: 10 } }) });
  });
}

async function fulfillSearch(route: Route) {
  const payload = route.request().postDataJSON() as { mode?: string; query?: string } | undefined;
  const name = payload?.mode === 'replacement' ? 'Olive oil' : 'Tofu';
  await route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      success: true,
      data: {
        items: [{ id: 'food-1', name, tags: ['vegan'], macros: { protein: 10, carbs: 2, fat: 4, unitBasis: '100g' }, calories: 120 }],
        totalCount: 1,
        page: 1,
        pageSize: 10,
        similarityScores: [],
        warnings: []
      }
    })
  });
}
