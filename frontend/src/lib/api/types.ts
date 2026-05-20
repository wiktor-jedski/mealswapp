import type { ThemePreference } from '../theme/theme';

export type UUID = string;
export type SearchMode = 'single' | 'replacement' | 'diet';
export type TagFilterKind = 'diet' | 'allergen' | 'functionality' | 'curation';
export type ErrorCategory = 'validation' | 'auth' | 'entitlement' | 'network' | 'timeout' | 'server' | 'dependency' | 'unknown';

export interface AppError {
  category: ErrorCategory;
  code: string;
  message: string;
  retryable: boolean;
  requestId?: string;
  fields?: unknown;
  cause?: unknown;
}

export interface Envelope<T> {
  data?: T;
  error?: AppError;
  meta?: { requestId?: string };
  success: boolean;
}

export interface IngredientInput {
  itemId?: UUID;
  name: string;
  quantity?: number;
  unit?: string;
}

export interface TagFilter {
  tagId: UUID;
  kind: TagFilterKind;
  include: boolean;
}

export interface SearchRequest {
  query: string;
  mode: SearchMode;
  page: number;
  filters?: TagFilter[];
  ingredients?: IngredientInput[];
  sourceItemId?: UUID | string;
  enabledMacros?: MacroToggles;
  dietaryTagIds?: UUID[];
  allergenTagIds?: UUID[];
  sourceProviders?: string[];
}

export interface MacroToggles {
  protein: boolean;
  carbs: boolean;
  fat: boolean;
}

export interface MacroValues {
  protein: number;
  carbs: number;
  fat: number;
}

export interface MacroSummary extends MacroValues {
  unitBasis: '100g' | '100ml' | 'serving';
}

export interface SimilarityBadge {
  score: number;
  tier: 'excellent' | 'good' | 'fair' | 'poor';
  colorHex: string;
  imageUrl: string;
}

export interface FoodItemViewModel {
  id: UUID;
  name: string;
  imageUrl?: string;
  macros: MacroSummary;
  calories?: number;
  matchingQuantity?: number;
  tags: string[];
  similarity?: SimilarityBadge;
}

export interface SearchResponse {
  items: FoodItemViewModel[];
  totalCount: number;
  page: number;
  pageSize: number;
  similarityScores: number[];
  warnings: string[];
}

export interface RankedAutocomplete {
  itemId: UUID;
  label: string;
  exactMatch: boolean;
  levenshteinDistance: number;
  length: number;
  rank: number;
}

export interface UserProfile {
  id?: UUID;
  userId?: UUID;
  email?: string;
  emailVerified?: boolean;
  displayName?: string;
  unitSystem?: 'metric' | 'imperial';
  themePreference?: ThemePreference;
  dietarySettings?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
}

export type SubscriptionTier = 'free' | 'trial' | 'paid' | 'admin';

export interface Entitlement {
  userId: UUID;
  tier: SubscriptionTier;
  status: 'active' | 'expired' | 'past_due' | 'cancelled';
  searchLimitPer24h: number;
  allowedModes: SearchMode[];
  allowedFeatures?: string[];
  expiresAt?: string;
  stripeCustomerId?: string;
  stripeSubscriptionId?: string;
}

export interface SubscriptionStatus {
  entitlement: Entitlement;
  billingState: string;
  plans?: SubscriptionPlan[];
}

export interface SubscriptionPlan {
  id: string;
  tier: SubscriptionTier;
  interval: string;
  priceCents: number;
  searchLimitPer24h?: number;
  allowedModes?: SearchMode[];
}

export interface CheckoutSession {
  id: string;
  url: string;
}

export interface CustomerPortalSession {
  url: string;
}

export type JobStatus = 'queued' | 'processing' | 'completed' | 'failed' | 'cancelled';

export interface DietOptimizationRequest {
  originalMeals: DietOptimizationMealInput[];
  targetMacros: MacroValues;
  excludedIds: UUID[];
  tolerancePercent: number;
}

export interface DietOptimizationMealInput {
  id: UUID | string;
  name: string;
  quantity: number;
  macros?: MacroValues;
  calories?: number;
}

export interface DietAlternativeMeal {
  itemId: UUID | string;
  quantity: number;
}

export interface DietAlternative {
  meals: DietAlternativeMeal[];
  macros: MacroValues;
  calories: number;
  similarityScore: number;
}

export interface OptimizationJob {
  jobId: UUID;
  userId: UUID;
  request: DietOptimizationRequest;
  status: JobStatus;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
  error?: string;
  progress?: number;
  result?: DietAlternative[];
}

export interface OptimizationSubmitResponse {
  jobId: UUID;
  pollUrl: string;
  status: JobStatus;
}

export type ExternalProvider = 'usda' | 'openfoodfacts' | 'all';

export interface ExternalSearchRequest {
  query: string;
  provider: ExternalProvider;
  page: number;
}

export interface ExternalCandidate {
  provider: string;
  externalId: string;
  name: string;
  macrosPer100: MacroValues;
  imageUrl?: string;
  raw: Record<string, unknown>;
}

export interface CuratedItemDraft {
  sourceProvider?: string;
  externalId?: string;
  name: string;
  physicalState: 'solid' | 'liquid';
  macrosPer100: MacroValues;
  categoryTagIds: UUID[];
  functionalityTagIds: UUID[];
  imageUrl?: string;
}
