import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'bun:test';

describe('App scaffold', () => {
  it('defines the empty search shell expected by the bootstrap task', () => {
    const source = readFileSync(join(import.meta.dir, 'App.svelte'), 'utf8');

    expect(source).toContain('Mealswapp');
    expect(source).toContain('Search food');
    expect(source).toContain('Single item');
  });
});
