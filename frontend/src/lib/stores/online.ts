import { readable, type Readable } from "svelte/store";

export interface OnlineEventTarget {
  onLine: boolean;
  addEventListener(type: "online" | "offline", listener: () => void): void;
  removeEventListener(type: "online" | "offline", listener: () => void): void;
}

// Implements DESIGN-001 OfflineBanner browser connectivity subscription and cleanup.
export function createOnlineStatus(target: OnlineEventTarget | null = browserOnlineTarget()): Readable<boolean> {
  return readable(target?.onLine ?? true, (set) => {
    if (!target) return;
    const online = () => set(true);
    const offline = () => set(false);
    target.addEventListener("online", online);
    target.addEventListener("offline", offline);
    return () => {
      target.removeEventListener("online", online);
      target.removeEventListener("offline", offline);
    };
  });
}

function browserOnlineTarget(): OnlineEventTarget | null {
  if (typeof window === "undefined" || typeof navigator === "undefined") return null;
  return { onLine: navigator.onLine, addEventListener: (type, listener) => window.addEventListener(type, listener), removeEventListener: (type, listener) => window.removeEventListener(type, listener) };
}
