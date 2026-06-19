// Implements DESIGN-001 AutocompleteDropdown deterministic debounce scheduling.
export function createDebouncer(delayMs: number, callback: (value: string) => void) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  return {
    schedule(value: string) {
      if (timer !== undefined) clearTimeout(timer);
      timer = setTimeout(() => { timer = undefined; callback(value); }, delayMs);
    },
    cancel() {
      if (timer !== undefined) clearTimeout(timer);
      timer = undefined;
    }
  };
}
