import { afterEach, beforeEach, expect, jest, test } from "bun:test";
import { createDebouncer } from "./debounce";

beforeEach(() => jest.useFakeTimers());
afterEach(() => jest.useRealTimers());

// Implements DESIGN-001 AutocompleteDropdown 150ms debounce verification.
test("runs once 150ms after the final keystroke", () => {
  const values: string[] = [];
  const debouncer = createDebouncer(150, (value) => values.push(value));
  debouncer.schedule("a");
  jest.advanceTimersByTime(100);
  debouncer.schedule("ap");
  jest.advanceTimersByTime(149);
  expect(values).toEqual([]);
  jest.advanceTimersByTime(1);
  expect(values).toEqual(["ap"]);
  debouncer.cancel();
});
