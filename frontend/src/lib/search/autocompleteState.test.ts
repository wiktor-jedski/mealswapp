import { describe, expect, it } from 'bun:test';
import { ApiClientError } from '../api/client';
import type { RankedAutocomplete } from '../api/types';
import { createAutocompleteController, highlightParts } from './autocompleteState';

describe('AutocompleteDropdown state', () => {
  it('debounces autocomplete API calls and preserves deterministic option order', async () => {
    const timers: Array<() => void> = [];
    const calls: string[] = [];
    const controller = createAutocompleteController({
      api: { autocomplete: async (query) => {
        calls.push(query);
        return options();
      } },
      setTimeoutFn: (callback) => {
        timers.push(callback);
        return callback;
      },
      clearTimeoutFn: () => undefined
    });

    controller.setQuery('to');
    controller.setQuery('tofu');
    expect(calls.length).toBe(0);
    await timers[timers.length - 1]?.();
    await Promise.resolve();
    await Promise.resolve();

    expect(calls).toEqual(['tofu']);
    expect(controller.getState().options.map((option) => option.label)).toEqual(['Tofu', 'Tofu firm']);
    expect(controller.getState().selectedIndex).toBe(0);
  });

  it('supports arrow, tab, shift-tab, enter, and escape behavior', async () => {
    const selected: string[] = [];
    const controller = createAutocompleteController({
      api: { autocomplete: async () => options() },
      onSelect: (option) => selected.push(option.label)
    });
    controller.setQuery('tofu');
    await controller.fetchOptions();

    expect(controller.handleKey('ArrowDown')).toBe(true);
    expect(controller.getState().selectedIndex).toBe(1);
    expect(controller.handleKey('Tab', true)).toBe(true);
    expect(controller.getState().selectedIndex).toBe(0);
    expect(controller.handleKey('Tab')).toBe(true);
    expect(controller.getState().selectedIndex).toBe(1);
    expect(controller.handleKey('Enter')).toBe(true);
    expect(selected).toEqual(['Tofu firm']);
    expect(controller.getState().isOpen).toBe(false);

    controller.setQuery('tofu');
    await controller.fetchOptions();
    expect(controller.handleKey('Escape')).toBe(true);
    expect(controller.getState().isOpen).toBe(false);
  });

  it('supports blur dismissal and direct selection', async () => {
    const selected: string[] = [];
    const controller = createAutocompleteController({
      api: { autocomplete: async () => options() },
      onSelect: (option) => selected.push(option.label)
    });
    controller.setQuery('tofu');
    await controller.fetchOptions();

    controller.blur();
    expect(controller.getState().isOpen).toBe(false);

    await controller.fetchOptions();
    controller.select(0);
    expect(selected).toEqual(['Tofu']);
    expect(controller.getState().query).toBe('Tofu');
  });

  it('surfaces API errors without throwing', async () => {
    const controller = createAutocompleteController({
      api: { autocomplete: async () => {
        throw new ApiClientError({ category: 'dependency', code: 'dependency_unavailable', message: 'Autocomplete unavailable', retryable: true });
      } }
    });

    controller.setQuery('tofu');
    await controller.fetchOptions();

    expect(controller.getState().isOpen).toBe(true);
    expect(controller.getState().options).toEqual([]);
    expect(controller.getState().error?.code).toBe('dependency_unavailable');
  });

  it('highlights matched text segments case-insensitively', () => {
    expect(highlightParts('Red Lentils', 'lent')).toEqual([
      { text: 'Red ', highlighted: false },
      { text: 'Lent', highlighted: true },
      { text: 'ils', highlighted: false }
    ]);
  });
});

function options(): RankedAutocomplete[] {
  return [
    { itemId: '1', label: 'Tofu', exactMatch: true, levenshteinDistance: 0, length: 4, rank: 1 },
    { itemId: '2', label: 'Tofu firm', exactMatch: false, levenshteinDistance: 5, length: 9, rank: 2 }
  ];
}
