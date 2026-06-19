import type { SavedItem, SavedItemsEnvelope, SearchHistoryEntry, SearchHistoryEnvelope } from "./generated";

export interface ActivityData {
  authenticated: boolean;
  history: SearchHistoryEntry[];
  favorites: SavedItem[];
}

// Implements DESIGN-001 SidebarComponent generated authenticated activity loading.
export class ActivityClient {
  constructor(private readonly fetcher: typeof fetch = globalThis.fetch.bind(globalThis), private readonly baseURL = "") {}

  async load(): Promise<ActivityData> {
    const init: RequestInit = { credentials: "include" };
    const [historyResponse, favoritesResponse] = await Promise.all([
      this.fetcher(`${this.baseURL}/api/v1/search-history`, init),
      this.fetcher(`${this.baseURL}/api/v1/saved-items?kind=favorite`, init)
    ]);
    if (historyResponse.status === 401 || favoritesResponse.status === 401) return { authenticated: false, history: [], favorites: [] };
    if (!historyResponse.ok || !favoritesResponse.ok) throw new Error("activity_unavailable");
    const historyEnvelope = await historyResponse.json() as SearchHistoryEnvelope;
    const favoriteEnvelope = await favoritesResponse.json() as SavedItemsEnvelope;
    return { authenticated: true, history: historyEnvelope.data?.history ?? [], favorites: favoriteEnvelope.data?.items ?? [] };
  }
}
