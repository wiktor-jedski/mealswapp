import { describe, expect, it } from 'bun:test';
import type { FoodItemViewModel } from '../api/types';
import { gridColumns, imageForItem, paginationState, quantityLabel, scaleMacros, similarityPercent } from './resultsGrid';

describe('ResultsGrid helpers', () => {
  it('builds pagination metadata with max page bounds', () => {
    expect(paginationState({ page: 2, pageSize: 10, totalCount: 25 })).toEqual({
      page: 2,
      pageSize: 10,
      totalCount: 25,
      totalPages: 3,
      canPrevious: true,
      canNext: true
    });
    expect(paginationState({ page: 99, pageSize: 10, totalCount: 11 }).page).toBe(2);
  });

  it('uses item images and category placeholders', () => {
    expect(imageForItem({ ...food(), imageUrl: 'https://example.test/tofu.jpg' })).toBe('https://example.test/tofu.jpg');
    expect(imageForItem({ ...food(), tags: ['Dairy'] })).toBe('/static/placeholders/dairy.svg');
    expect(imageForItem({ ...food(), tags: [] })).toBe('/static/placeholders/food.svg');
  });

  it('scales displayed macro values by quantity', () => {
    expect(scaleMacros({ protein: 10, carbs: 20, fat: 5, unitBasis: '100g' }, 150)).toEqual({
      protein: 15,
      carbs: 30,
      fat: 7.5,
      unitBasis: '100g'
    });
  });

  it('formats similarity and matching quantity labels', () => {
    expect(similarityPercent({ ...food(), similarity: { score: 0.86, tier: 'excellent', colorHex: '#16A34A', imageUrl: '/static/similarity/green.svg' } })).toBe('86%');
    expect(similarityPercent(food())).toBe('N/A');
    expect(quantityLabel({ ...food(), matchingQuantity: 42.25 })).toBe('42.3 g');
    expect(quantityLabel(food())).toBe('100 g');
  });

  it('selects responsive grid columns', () => {
    expect(gridColumns(360)).toBe(1);
    expect(gridColumns(800)).toBe(2);
    expect(gridColumns(1200)).toBe(3);
  });
});

function food(): FoodItemViewModel {
  return {
    id: 'food-1',
    name: 'Tofu',
    macros: { protein: 10, carbs: 2, fat: 4, unitBasis: '100g' },
    calories: 120,
    tags: ['vegan']
  };
}
