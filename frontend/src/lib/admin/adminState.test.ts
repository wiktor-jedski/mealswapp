import { describe, expect, it } from 'bun:test';
import { candidateToAdminFood, createAdminController } from './adminState';
import type { AdminApi } from './adminState';
import type { AdminFoodItem, NormalizedExternalCandidate } from '../api/types';

describe('Admin controller', () => {
  it('searches external providers and imports a selected candidate', async () => {
    const api = fakeAdminApi();
    const controller = createAdminController(api);
    controller.setExternalQuery('tofu');
    controller.setExternalProvider('openfoodfacts');

    await controller.searchExternal();
    const candidate = controller.getState().external?.candidates[0] as NormalizedExternalCandidate;
    controller.selectCandidate(candidate);
    await controller.importSelectedCandidate();

    expect(api.calls[0]).toEqual(['external', 'tofu', 'openfoodfacts', 1, 10]);
    expect(api.created?.source?.provider).toBe('openfoodfacts');
    expect(api.created?.source?.curationState).toBe('approved');
    expect(controller.getState().lastAction).toBe('import_item');
  });

  it('loads items and transitions curation state', async () => {
    const api = fakeAdminApi();
    const controller = createAdminController(api);

    await controller.loadItems('tof', 2);
    await controller.transitionItem('food-1', 'approve');

    expect(api.calls[0]).toEqual(['items', 'tof', 2, 10]);
    expect(api.calls[1]).toEqual(['transition', 'food-1', 'approve']);
    expect(controller.getState().items?.items[0].source?.curationState).toBe('approved');
  });

  it('manages tags and assignments', async () => {
    const api = fakeAdminApi();
    const controller = createAdminController(api);

    await controller.loadTags('functionality');
    await controller.upsertTag({ name: 'High protein', kind: 'functionality' });
    await controller.assignTag('food-1', 'tag-1');
    await controller.mergeTags('tag-old', 'tag-1');

    expect(api.calls).toContainEqual(['tags', 'functionality']);
    expect(api.calls).toContainEqual(['assign-tag', 'food-1', 'tag-1']);
    expect(api.calls).toContainEqual(['merge-tags', 'tag-old', 'tag-1']);
    expect(controller.getState().tags[0].name ?? controller.getState().tags[0].Name).toBe('High protein');
  });

  it('loads users, disables accounts, resets lockout, and loads audit history', async () => {
    const api = fakeAdminApi();
    const controller = createAdminController(api);

    await controller.loadUsers('user', 1);
    await controller.selectUser('user-1');
    await controller.disableUser('user-1');
    await controller.resetUserLockout('user-1');
    await controller.loadUserAudit('user-1');

    expect(api.calls).toContainEqual(['users', 'user', 1, 10]);
    expect(api.calls).toContainEqual(['disable-user', 'user-1']);
    expect(api.calls).toContainEqual(['reset-lockout', 'user-1']);
    expect(controller.getState().selectedUser?.user.disabled).toBe(true);
    expect(controller.getState().audit?.entries[0].action).toBe('admin.disable_user');
  });

  it('maps normalized candidates into admin food drafts', () => {
    const draft = candidateToAdminFood(externalCandidate());

    expect(draft.name).toBe('Organic Tofu');
    expect(draft.macrosPer100?.protein).toBe(12);
    expect(draft.source?.externalId).toBe('off-1');
  });
});

function fakeAdminApi(): AdminApi & { calls: unknown[][]; created?: AdminFoodItem } {
  const calls: unknown[][] = [];
  const api: AdminApi & { calls: unknown[][]; created?: AdminFoodItem } = {
    calls,
    async adminExternalSearch(query, provider, page = 1, pageSize = 10) {
      calls.push(['external', query, provider, page, pageSize]);
      return { candidates: [externalCandidate()], page, pageSize, warnings: [] };
    },
    async adminCreateItem(item) {
      api.created = item;
      calls.push(['create-item', item.name]);
      return { ...item, id: 'food-1' };
    },
    async adminListItems(query = '', page = 1, pageSize = 10) {
      calls.push(['items', query, page, pageSize]);
      return { items: [{ id: 'food-1', name: 'Tofu', source: { curationState: 'draft' } }], total: 1, page, limit: pageSize };
    },
    async adminUpdateItem(id, item) {
      calls.push(['update-item', id]);
      return { ...item, id };
    },
    async adminTransitionItem(id, transition) {
      calls.push(['transition', id, transition]);
      return { id, name: 'Tofu', source: { curationState: transition === 'approve' ? 'approved' : transition === 'reject' ? 'rejected' : 'inactive' } };
    },
    async adminListTags(kind) {
      calls.push(['tags', kind]);
      return [{ id: 'tag-1', name: 'Vegan', kind: 'diet', active: true }];
    },
    async adminUpsertTag(tag) {
      calls.push(['upsert-tag', tag.name ?? tag.Name]);
      return { ...tag, id: 'tag-2', active: true };
    },
    async adminAssignTag(foodItemId, tagId) {
      calls.push(['assign-tag', foodItemId, tagId]);
    },
    async adminMergeTags(sourceId, targetId) {
      calls.push(['merge-tags', sourceId, targetId]);
    },
    async adminListUsers(query = '', page = 1, pageSize = 10) {
      calls.push(['users', query, page, pageSize]);
      return { users: [{ id: 'user-1', email: 'user@example.com', displayName: 'User', role: 'user', disabled: false }], total: 1, page, limit: pageSize };
    },
    async adminGetUser(id) {
      calls.push(['get-user', id]);
      return { user: { id, email: 'user@example.com', disabled: false }, entitlement: { plan: 'paid', status: 'active' } };
    },
    async adminDisableUser(id) {
      calls.push(['disable-user', id]);
      return { id, email: 'user@example.com', disabled: true };
    },
    async adminResetUserLockout(id) {
      calls.push(['reset-lockout', id]);
    },
    async adminUserAudit(id, page = 1, pageSize = 10) {
      calls.push(['audit', id, page, pageSize]);
      return { entries: [{ id: 'audit-1', action: 'admin.disable_user', target: `user:${id}` }], total: 1, page, limit: pageSize };
    }
  };
  return api;
}

function externalCandidate(): NormalizedExternalCandidate {
  return {
    provider: 'openfoodfacts',
    externalId: 'off-1',
    name: 'Organic Tofu',
    physicalState: 'solid',
    macrosPer100: { protein: 12, carbs: 2, fat: 6 },
    caloriesPer100: 110,
    micros: { Iron: 2.4 },
    servingSize: 100,
    servingUnit: 'gram',
    imageUrl: 'https://example.test/tofu.jpg'
  };
}
