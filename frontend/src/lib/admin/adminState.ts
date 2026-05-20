import type { ApiClient } from '../api/client';
import type {
  AdminAuditHistory,
  AdminFoodItem,
  AdminItemList,
  AdminTag,
  AdminUserDetail,
  AdminUserList,
  AppError,
  ExternalProvider,
  ExternalSearchResult,
  NormalizedExternalCandidate,
  TagFilterKind
} from '../api/types';

export type AdminTab = 'external' | 'items' | 'tags' | 'users' | 'audit';
export type AdminStatus = 'idle' | 'loading' | 'success' | 'error';

export interface AdminState {
  activeTab: AdminTab;
  status: AdminStatus;
  error?: AppError;
  externalQuery: string;
  externalProvider: ExternalProvider;
  external?: ExternalSearchResult;
  selectedCandidate?: NormalizedExternalCandidate;
  items?: AdminItemList;
  tags: AdminTag[];
  tagKind: TagFilterKind;
  users?: AdminUserList;
  selectedUser?: AdminUserDetail;
  audit?: AdminAuditHistory;
  lastAction?: string;
}

export interface AdminApi {
  adminExternalSearch: ApiClient['adminExternalSearch'];
  adminCreateItem: ApiClient['adminCreateItem'];
  adminListItems: ApiClient['adminListItems'];
  adminUpdateItem: ApiClient['adminUpdateItem'];
  adminTransitionItem: ApiClient['adminTransitionItem'];
  adminListTags: ApiClient['adminListTags'];
  adminUpsertTag: ApiClient['adminUpsertTag'];
  adminAssignTag: ApiClient['adminAssignTag'];
  adminMergeTags: ApiClient['adminMergeTags'];
  adminListUsers: ApiClient['adminListUsers'];
  adminGetUser: ApiClient['adminGetUser'];
  adminDisableUser: ApiClient['adminDisableUser'];
  adminResetUserLockout: ApiClient['adminResetUserLockout'];
  adminUserAudit: ApiClient['adminUserAudit'];
}

export function createDefaultAdminState(): AdminState {
  return {
    activeTab: 'external',
    status: 'idle',
    externalQuery: '',
    externalProvider: 'all',
    tags: [],
    tagKind: 'diet'
  };
}

export function createAdminController(api: AdminApi) {
  let state = createDefaultAdminState();
  const listeners = new Set<(state: AdminState) => void>();

  function subscribe(listener: (state: AdminState) => void): () => void {
    listeners.add(listener);
    listener(state);
    return () => listeners.delete(listener);
  }

  function emit(next: Partial<AdminState>): AdminState {
    state = { ...state, ...next };
    for (const listener of listeners) {
      listener(state);
    }
    return state;
  }

  async function run<T>(action: () => Promise<T>, apply: (result: T) => Partial<AdminState>): Promise<T | undefined> {
    emit({ status: 'loading', error: undefined });
    try {
      const result = await action();
      emit({ status: 'success', ...apply(result) });
      return result;
    } catch (error) {
      emit({ status: 'error', error: normalizeAdminError(error) });
      return undefined;
    }
  }

  function setTab(activeTab: AdminTab): AdminState {
    return emit({ activeTab, error: undefined });
  }

  function setExternalQuery(externalQuery: string): AdminState {
    return emit({ externalQuery });
  }

  function setExternalProvider(externalProvider: ExternalProvider): AdminState {
    return emit({ externalProvider });
  }

  async function searchExternal(page = 1): Promise<ExternalSearchResult | undefined> {
    return run(
      () => api.adminExternalSearch(state.externalQuery.trim(), state.externalProvider, page, 10),
      (external) => ({ external, activeTab: 'external', lastAction: 'external_search' })
    );
  }

  function selectCandidate(candidate: NormalizedExternalCandidate): AdminState {
    return emit({ selectedCandidate: candidate });
  }

  async function importSelectedCandidate(): Promise<AdminFoodItem | undefined> {
    if (!state.selectedCandidate) {
      return undefined;
    }
    return run(
      () => api.adminCreateItem(candidateToAdminFood(state.selectedCandidate as NormalizedExternalCandidate)),
      (created) => ({ selectedCandidate: undefined, lastAction: 'import_item', items: appendItem(state.items, created) })
    );
  }

  async function loadItems(query = '', page = 1): Promise<AdminItemList | undefined> {
    return run(() => api.adminListItems(query, page, 10), (items) => ({ items, activeTab: 'items', lastAction: 'load_items' }));
  }

  async function transitionItem(id: string, transition: 'approve' | 'reject' | 'deactivate'): Promise<AdminFoodItem | undefined> {
    return run(() => api.adminTransitionItem(id, transition), (item) => ({ items: replaceItem(state.items, item), lastAction: `item_${transition}` }));
  }

  async function loadTags(kind: TagFilterKind = state.tagKind): Promise<AdminTag[] | undefined> {
    return run(() => api.adminListTags(kind), (tags) => ({ tags, tagKind: kind, activeTab: 'tags', lastAction: 'load_tags' }));
  }

  async function upsertTag(tag: AdminTag): Promise<AdminTag | undefined> {
    return run(() => api.adminUpsertTag(tag), (saved) => ({ tags: upsertTagInList(state.tags, saved), lastAction: 'upsert_tag' }));
  }

  async function assignTag(foodItemId: string, tagId: string): Promise<void | undefined> {
    return run(() => api.adminAssignTag(foodItemId, tagId), () => ({ lastAction: 'assign_tag' }));
  }

  async function mergeTags(sourceId: string, targetId: string): Promise<void | undefined> {
    return run(() => api.adminMergeTags(sourceId, targetId), () => ({ lastAction: 'merge_tags' }));
  }

  async function loadUsers(query = '', page = 1): Promise<AdminUserList | undefined> {
    return run(() => api.adminListUsers(query, page, 10), (users) => ({ users, activeTab: 'users', lastAction: 'load_users' }));
  }

  async function selectUser(id: string): Promise<AdminUserDetail | undefined> {
    return run(() => api.adminGetUser(id), (selectedUser) => ({ selectedUser, activeTab: 'users', lastAction: 'select_user' }));
  }

  async function disableUser(id: string): Promise<AdminUserDetail['user'] | undefined> {
    return run(() => api.adminDisableUser(id), (user) => ({ selectedUser: state.selectedUser ? { ...state.selectedUser, user } : undefined, lastAction: 'disable_user' }));
  }

  async function resetUserLockout(id: string): Promise<void | undefined> {
    return run(() => api.adminResetUserLockout(id), () => ({ lastAction: 'reset_lockout' }));
  }

  async function loadUserAudit(id: string, page = 1): Promise<AdminAuditHistory | undefined> {
    return run(() => api.adminUserAudit(id, page, 10), (audit) => ({ audit, activeTab: 'audit', lastAction: 'load_audit' }));
  }

  function getState(): AdminState {
    return state;
  }

  return {
    subscribe,
    getState,
    setTab,
    setExternalQuery,
    setExternalProvider,
    searchExternal,
    selectCandidate,
    importSelectedCandidate,
    loadItems,
    transitionItem,
    loadTags,
    upsertTag,
    assignTag,
    mergeTags,
    loadUsers,
    selectUser,
    disableUser,
    resetUserLockout,
    loadUserAudit
  };
}

export function candidateToAdminFood(candidate: NormalizedExternalCandidate): AdminFoodItem {
  return {
    name: candidate.name,
    physicalState: candidate.physicalState ?? 'solid',
    servingUnit: candidate.servingUnit ?? 'gram',
    servingSize: candidate.servingSize ?? 100,
    caloriesPer100: candidate.caloriesPer100,
    macrosPer100: candidate.macrosPer100,
    micros: candidate.micros,
    imageUrl: candidate.imageUrl,
    source: {
      provider: candidate.provider,
      externalId: candidate.externalId,
      curationState: 'approved'
    }
  };
}

function appendItem(items: AdminItemList | undefined, item: AdminFoodItem): AdminItemList | undefined {
  if (!items) {
    return items;
  }
  return { ...items, items: [item, ...items.items], total: items.total + 1 };
}

function replaceItem(items: AdminItemList | undefined, item: AdminFoodItem): AdminItemList | undefined {
  if (!items) {
    return items;
  }
  const id = item.id ?? item.ID;
  return { ...items, items: items.items.map((existing) => ((existing.id ?? existing.ID) === id ? item : existing)) };
}

function upsertTagInList(tags: AdminTag[], tag: AdminTag): AdminTag[] {
  const id = tag.id ?? tag.ID;
  const existing = tags.findIndex((candidate) => (candidate.id ?? candidate.ID) === id);
  if (existing < 0) {
    return [tag, ...tags];
  }
  return tags.map((candidate, index) => (index === existing ? tag : candidate));
}

function normalizeAdminError(error: unknown): AppError {
  if (error && typeof error === 'object' && 'category' in error && 'code' in error) {
    return error as AppError;
  }
  return {
    category: 'unknown',
    code: 'admin_error',
    message: 'Admin request failed',
    retryable: false,
    cause: error
  };
}
