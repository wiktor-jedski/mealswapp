import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'bun:test';

describe('App scaffold', () => {
  it('defines the empty search shell expected by the bootstrap task', () => {
    const appSource = readFileSync(join(import.meta.dir, 'App.svelte'), 'utf8');
    const searchViewSource = readFileSync(join(import.meta.dir, 'lib/components/SearchView.svelte'), 'utf8');
    const sidebarSource = readFileSync(join(import.meta.dir, 'lib/components/SidebarComponent.svelte'), 'utf8');
    const source = `${appSource}\n${searchViewSource}\n${sidebarSource}`;

    expect(source).toContain('Mealswapp');
    expect(source).toContain('Search food');
    expect(source).toContain('Single item');
  });
});
