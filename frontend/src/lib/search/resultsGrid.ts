import type { FoodItemViewModel, MacroSummary, SearchResponse } from '../api/types';

export interface PaginationState {
  page: number;
  pageSize: number;
  totalCount: number;
  totalPages: number;
  canPrevious: boolean;
  canNext: boolean;
}

const placeholderByTag: Record<string, string> = {
  meat: '/static/placeholders/meat.svg',
  dairy: '/static/placeholders/dairy.svg',
  gluten: '/static/placeholders/gluten.svg',
  vegan: '/static/placeholders/plant.svg',
  vegetarian: '/static/placeholders/plant.svg'
};

export function paginationState(response: Pick<SearchResponse, 'page' | 'pageSize' | 'totalCount'>): PaginationState {
  const pageSize = Math.max(1, response.pageSize || 10);
  const totalPages = Math.max(1, Math.ceil(response.totalCount / pageSize));
  const page = Math.min(Math.max(1, response.page || 1), totalPages);
  return {
    page,
    pageSize,
    totalCount: response.totalCount,
    totalPages,
    canPrevious: page > 1,
    canNext: page < totalPages
  };
}

export function placeholderImage(tags: string[]): string {
  const normalized = tags.map((tag) => tag.toLowerCase());
  for (const tag of normalized) {
    if (placeholderByTag[tag]) {
      return placeholderByTag[tag];
    }
  }
  return '/static/placeholders/food.svg';
}

export function imageForItem(item: FoodItemViewModel): string {
  return item.imageUrl && item.imageUrl.trim() !== '' ? item.imageUrl : placeholderImage(item.tags);
}

export function scaleMacros(macros: MacroSummary, quantity = 100): MacroSummary {
  const multiplier = quantity / 100;
  return {
    ...macros,
    protein: roundOne(macros.protein * multiplier),
    carbs: roundOne(macros.carbs * multiplier),
    fat: roundOne(macros.fat * multiplier)
  };
}

export function similarityPercent(item: FoodItemViewModel): string {
  if (!item.similarity) {
    return 'N/A';
  }
  return `${Math.round(item.similarity.score * 100)}%`;
}

export function quantityLabel(item: FoodItemViewModel): string {
  const quantity = item.matchingQuantity ?? 100;
  return `${roundOne(quantity)} g`;
}

export function gridColumns(widthPx: number): 1 | 2 | 3 {
  if (widthPx < 640) {
    return 1;
  }
  if (widthPx < 1024) {
    return 2;
  }
  return 3;
}

function roundOne(value: number): number {
  return Math.round(value * 10) / 10;
}
