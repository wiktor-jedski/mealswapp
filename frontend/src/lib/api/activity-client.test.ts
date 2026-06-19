import { describe, expect, test } from "bun:test";
import type { SavedItemsEnvelope, SearchHistoryEnvelope } from "./generated";
import { ActivityClient } from "./activity-client";

const historyEnvelope = {
  status: "ok",
  requestId: "history",
  data: { history: [{ id: "history-1", query: "banana", mode: "catalog", filtersHash: "none" }] }
} satisfies SearchHistoryEnvelope;

const savedItemsEnvelope = {
  status: "ok",
  requestId: "favorites",
  data: { items: [{ id: "saved-1", itemId: "food-1", kind: "favorite" }] }
} satisfies SavedItemsEnvelope;

// Implements DESIGN-001 SidebarComponent generated authenticated activity contract verification.
describe("ActivityClient", () => {
  test("loads history and favorites from generated envelopes", async () => {
    const fetcher = async (input: RequestInfo | URL) => Response.json(
      String(input).includes("search-history") ? historyEnvelope : savedItemsEnvelope
    );

    const activity = await new ActivityClient(fetcher as typeof fetch).load();

    expect(activity).toEqual({
      authenticated: true,
      history: historyEnvelope.data.history,
      favorites: savedItemsEnvelope.data.items
    });
  });
});
