# Software Requirements Specification (SRS)
**Project:** Mealswapp
**Process:** SWE.1 Software Requirements Analysis
**Version:** 1.0 (ASPICE 4.0 Compliant)

## 1. Introduction
This document defines the software-level requirements for the Mealswapp application. It decomposes stakeholder intent into atomic, verifiable, and traceable requirements using the EARS syntax.

---

## 2. Software Requirements

### 2.1 Search Interface & Logic

---
## [SW-REQ-001] Default Search State
**Statement:** WHILE the software is in its initial state, the software shall set the Search Mode to 'Single Item' and enable all macronutrient toggles (Carbohydrates, Fats, Proteins).

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (State initialization check) |

**Notes:** Ensures consistent UX on application launch.
---

## [SW-REQ-002] Search Debounce Timing
**Statement:** WHEN the user provides text input into the search bar, the software shall delay the server-side query execution by 150 milliseconds.

| Attribute | Value |
| :--- | :--- |
| **Type** | Performance |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Timer/Event debounce check) |

**Notes:** Minimizes unnecessary API calls during rapid typing.
---

## [SW-REQ-003] Local Query Caching
**Statement:** The software shall store the 20 most recent unique search queries and their respective result sets in local memory (localStorage).

| Attribute | Value |
| :--- | :--- |
| **Type** | Performance |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Cache limit and LRU logic) |

**Notes:** Provides near-instant feedback for repeated queries.
---

## [SW-REQ-004] Autocomplete Ranking Priority
**Statement:** The software shall rank autocomplete suggestions using textual relevance in the following priority order: 1. Exact matches, 2. Typo distance (Levenshtein), 3. String length.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Sorting algorithm validation) |

**Notes:** Server-side prefix-based search must support these ranking parameters.
---

## [SW-REQ-005] Ingredient List Accumulation
**Statement:** WHILE in 'Ingredient List' mode, WHEN the user presses the 'Enter' key, the software shall add the currently selected autocomplete item to the active ingredient list.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Keyboard event handling) |

**Notes:** Ingredients must appear above the macronutrient toggle bar.
---

## [SW-REQ-006] Search Mode: Meal List
**Statement:** WHILE in 'Meal List' mode, the software shall allow the user to select and aggregate multiple meals into a single collection representing a one-day diet.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (State/List management check) |

**Notes:** This mode differs from the 'Ingredient List' by aggregating high-level meal objects rather than raw ingredients.
---

## [SW-REQ-007] UI Toggle Positioning
**Statement:** The software shall display the search mode toggles above the search bar and positioned vertically on top of the macronutrient toggle bar.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional (UI) |
| **Priority** | Low |
| **Feasibility** | Feasible |
| **Verification** | UI Inspection |

**Notes:** Ensures consistent visual hierarchy as defined in the wireframe intent.
---

## [SW-REQ-008] Search Bar Expansion
**Statement:** WHEN autocomplete suggestions are generated, the software shall expand the search interface container downwards to display the list of suggestions.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional (UI) |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Dynamic expansion check) |

**Notes:** Standard autocomplete behavior to prevent overlapping other UI elements.
---

## [SW-REQ-009] Keyboard Navigation: Autocomplete
**Statement:** WHILE the autocomplete suggestion list is visible, the software shall navigate focus forward through suggestions on 'Tab' input and backward on 'Shift+Tab' input.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional (Accessibility) |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Keyboard event handling) |

**Notes:** Enhances power-user efficiency for search selection.
---

## [SW-REQ-010] Search Result Pagination
**Statement:** The software shall paginate all search results with a maximum limit of 10 items per page.

| Attribute | Value |
| :--- | :--- |
| **Type** | Performance / UI |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (API response count and UI page count) |

**Notes:** Optimizes load times and mobile data usage.
---

## [SW-REQ-011] Search Result Data Fields
**Statement:** The software shall display the image, name, category tags, macronutrients per 100g/100ml, calories, and the similarity score for every item in the search result set.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | UI Inspection |

**Notes:** Items without a specific image must use the category-based placeholder.
---

## [SW-REQ-012] Category-Based Placeholders
**Statement:** IF an item record does not contain an image URL, THEN the software shall display a placeholder image associated with the item’s primary category tag (e.g., meat, dairy, gluten).

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional (UI) |
| **Priority** | Low |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Empty image source handling) |

**Notes:** Maintains a high-quality visual experience even when external API images are missing.
---

## [SW-REQ-013] Activity Sidebar
**Statement:** WHILE in the search view, the software shall provide a collapsible sidebar on the left side of the screen containing the user's search history and favorites.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional (UI) |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI Test (State toggle and visibility check) |

**Notes:** Provides a unified area for managing personal activity without leaving the main search context.
---

## [SW-REQ-014] Responsive Web Interface
**Statement:** The software shall provide a web-based user interface that renders responsively on both desktop and mobile web browsers (Chrome, Firefox).

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | UI Inspection (Cross-device browser testing) |

**Notes:** This is the primary delivery platform for the current phase.
---

## [SW-REQ-015] Light/Dark Mode Toggle
**Statement:** The software shall provide a toggleable theme switcher in the collapsible sidebar allowing users to select between light mode and dark mode, persisting the preference across sessions.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional (UI) |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Theme toggle and persistence check) |

**Notes:** Default theme should respect the user's system preference (prefers-color-scheme). User selection overrides system preference.
---

## [SW-REQ-089] Style Guide
**Statement:** The interface shall follow the Style Guide for layout, typography, and color consistency.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional (UI) |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | UI Inspection |

**Notes:** Style Guide saved in 02_STYLE_GUIDE.md,
---

### 2.2 Similarity Algorithm & Filtering

---
## [SW-REQ-016] Cosine Similarity Calculation
**Statement:** The software shall calculate item similarity based on the Cosine Similarity of a three-dimensional vector containing: Protein, Carbohydrates, Fat.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Mathematical correctness vs reference set) |

**Notes:** Core logic for meal/ingredient replacement.
---

## [SW-REQ-017] Similarity Threshold Filtering
**Statement:** The software shall exclude all items with a calculated Cosine Similarity score of less than 0.40 (40%) from the result set.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Filter logic validation) |

**Notes:** Prevents irrelevant results from appearing.
---

## [SW-REQ-018] Visual Similarity Indicators
**Statement:** The software shall assign visual indicators to results based on similarity score: Green/🌟 for ≥85%, Light Green/✨ for 70-84%, Yellow/👍 for 55-69%, and Red/👎 for <55%. To make sure that UI is stored properly, save emojis as images on server.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Low |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (UI rendering logic) |

**Notes:** Applies to all search modes.
---

## [SW-REQ-019] Tag-Based Filtering (Whitelist/Blacklist)
**Statement:** WHERE user-defined tag preferences exist, the software shall filter all search results to include only 'whitelisted' tags and exclude all 'blacklisted' tags.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Query parameter application) |

**Notes:** Critical for users with allergies or dietary restrictions (e.g., Gluten, Dairy).
---

## [SW-REQ-020] Real-time Macronutrient Scaling
**Statement:** WHEN the user modifies the numeric quantity input of a search result, the software shall recalculate and update all displayed macronutrient values in real-time.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Scaling logic correctness) |

**Notes:** Default quantity is 100g.
---

## [SW-REQ-021] Linear Programming Macro Matching
**Statement:** WHILE generating diet alternatives, the software shall use a Linear Programming (LP) algorithm to identify meal combinations that match the target Protein, Carbohydrate, and Fat profiles of the original diet.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Validation of LP solver output against macro targets) |

**Notes:** The solver must handle multiple constraints (P, C, F) simultaneously to find valid intersections.
---

## [SW-REQ-022] Calorie Optimization Priority
**Statement:** WHILE executing the Linear Programming optimization, the software shall define the "lowest total calorie count" as the primary objective function for the optimal solution.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Compare multiple valid solutions to ensure lowest calorie set is selected) |

**Notes:** Ensures that when multiple combinations meet macro targets, the most calorie-efficient one is presented first.
---

## [SW-REQ-023] Meal Set Diversity
**Statement:** The software shall minimize the inclusion of identical meal IDs between the original diet and the generated alternatives through a "best-effort" weight penalty in the optimization model.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Verify meal ID overlap counts across 3 generated alternatives) |

**Notes:** Prevents the system from simply suggesting the same diet back to the user.
---

## [SW-REQ-024] Implicit Similarity Search Trigger
**Statement:** IF the search bar is empty AND the active ingredient list contains two or more items, THEN the software shall automatically trigger a meal similarity search.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Logical trigger condition check) |

**Notes:** This is the primary automated logic for ingredient-set matching.
---

## [SW-REQ-025] Explicit Similarity Search Redundancy
**Statement:** The software shall provide a dedicated search button on the right-hand extremity of the search bar to allow the user to manually trigger the meal similarity search.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional (UI) |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI Inspection / Integration Test |

**Notes:** Provides a fallback for users who do not rely on the implicit trigger defined in SW-REQ-024.
---

## [SW-REQ-026] Result Sorting Order
**Statement:** The software shall return all search results sorted in descending order based on the calculated Cosine Similarity score.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Sorting algorithm validation) |

**Notes:** Ensures the most relevant matches appear at the top of the list.
---

## [SW-REQ-027] Recipe-Based Macro Summation
**Statement:** For recipe-based meals, the software shall calculate the similarity score based on the sum total of the macronutrients of all constituent ingredients.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Aggregation logic validation) |

**Notes:** Ensures complex meals are comparable to single-item entries.
---

## [SW-REQ-028] Comparative Quantity Calculation
**Statement:** WHILE displaying a replacement result, the software shall calculate and display the specific quantity of the replacement item required to match either the calorie count or the protein count of the original item.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Mathematical calculation check) |

**Notes:** This provides actionable data for users swapping ingredients in a recipe.
---

## [SW-REQ-029] Preparation Time Filtering
**Statement:** WHERE a 'Time to Prepare' filter is active, the software shall exclude all meals or recipes whose metadata field for preparation time exceeds the user-defined limit.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Filter application check) |

**Notes:** Requires the "Time to Prepare" metadata field to be populated in the data model.
---

## [SW-REQ-030] Alternative Diet Result Volume
**Statement:** The software shall generate a maximum of three (3) alternative meal combinations for every individual diet search request.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Verify that the result array size is ≤ 3) |

**Notes:** This limit ensures performance stability and prevents user choice paralysis. If the Linear Programming model (SW-REQ-021) finds fewer than 3 valid combinations meeting the diversity constraints, it may return 1 or 2 results.
---

## [SW-REQ-031] Context-Aware Replacements
**Statement:** WHERE a replacement search is executed, the software shall prioritize result items that share the same 'Functionality Tag' as the original item.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Search result relevance check) |

**Notes:** Prevents suggesting a "crunchy snack" as a replacement for "fat for frying" even if macros are similar.
---

### 2.3 Data Model & Unit Conversion

---
## [SW-REQ-032] Unit Conversion Logic (Imperial)
**Statement:** WHILE the application is set to 'US Imperial' mode, the software shall convert grams to ounces (1g ≈ 0.035oz) and milliliters to fluid ounces (1ml ≈ 0.033fl oz) for all displays.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Conversion accuracy) |

**Notes:** Storage remains in Metric per SW-REQ-033.
---

## [SW-REQ-033] Standardized Storage Units
**Statement:** The software shall store all item macronutrient values normalized to 100 grams (for solids) or 100 milliliters (for liquids).

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Data ingestion validation) |

**Notes:** Ensures consistency across different data sources (USDA/OpenFoodFacts).
---

## [SW-REQ-034] Meal Object Composition
**Statement:** The software shall support meal objects defined as either a single dish with standalone macronutrient data or a recipe object composed of multiple ingredients with relative quantities.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Data schema validation) |

**Notes:** This allows for both atomic database entries and dynamically calculated recipe-based meals.
---

## [SW-REQ-035] Item State & Unit Metadata
**Statement:** The software shall store a 'Physical State' flag (Solid/Liquid) and a 'Time to Prepare' duration (in minutes) for every item in the database.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Schema integrity check) |

**Notes:** Solid items default to grams (g); Liquid items default to milliliters (ml).
---

## [SW-REQ-036] Per-Unit Calculation Logic
**Statement:** The software shall store the average weight/volume of an item to enable the dynamic calculation and display of macronutrients 'per unit' (e.g., "per 1 apple").

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Weight-to-unit math validation) |

**Notes:** Allows users to switch views between 100g/ml and discrete unit counts.
---

## [SW-REQ-037] Functionality Tagging
**Statement:** The software shall assign one or more 'Functionality Tags' (e.g., "fat for frying," "sweetener," "crunchy snack") to every item to define its culinary role.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI/Data Inspection |

**Notes:** Crucial for providing replacements that serve the same purpose in a recipe.
---

## [SW-REQ-038] Micronutrient Storage & Algorithm Exclusion
**Statement:** The software shall allow each item to store a variable-length collection (from 0 to *n* entries) of micronutrient key-value pairs that are explicitly excluded from the Cosine Similarity calculation.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Low |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (1. Ensure items can save 0, 1, and multiple micronutrients. 2. Ensure similarity scores remain identical regardless of these values). |

**Notes:** This data is stored as a dictionary/hash-table mapping for supplemental user information only.
---

## [SW-REQ-039] Metric Unit Preference Selection
**Statement:** WHILE the application is set to Metric mode, the software shall allow the user to select either "per 100g/ml" or "per unit" as the global display preference.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Verify numerical display changes based on selection) |

**Notes:** "Per unit" relies on the weight data stored in SW-REQ-036.
---

## [SW-REQ-040] Imperial Unit Preference Selection
**Statement:** WHILE the application is set to US Imperial mode, the software shall allow the user to select either "oz / fl oz" or "per unit" as the global display preference.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Verify numerical conversion and label changes) |

**Notes:** When "oz / fl oz" is selected, the conversion logic from SW-REQ-032 is applied to the base 100g/ml storage.
---

## [SW-REQ-041] Global Preference Application
**Statement:** WHEN a user updates their unit preference in the settings, the software shall update all displayed macronutrient values across the search interface, history sidebar, and saved lists in real-time.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (End-to-end setting propagation) |

**Notes:** Ensures a consistent experience across the session without requiring manual page refreshes.
---

### 2.4 User Accounts & Subscriptions

---
## [SW-REQ-042] Free Tier Search Limitation
**Statement:** IF a user is on the 'Free' tier, THEN the software shall restrict the user to a maximum of 3 searches per 24-hour period.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Counter increment and lock) |

**Notes:** Enforced via backend middleware.
---

## [SW-REQ-043] Private Item Visibility
**Statement:** The software shall ensure that custom items created by a user are only accessible and visible to that specific authenticated user.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Security Audit / Integration Test (Cross-user access check) |

**Notes:** Critical for data privacy.
---

## [SW-REQ-044] Secure Credential Tokenization
**Statement:** The software shall use Stripe Elements for the capture of payment data to ensure that raw credit card information is tokenized at the client-side and never processed or stored by the application server.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security / Compliance (PCI-DSS) |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Security Audit (Network trace to ensure no PAN data reaches application backend) |

**Notes:** Mandatory for PCI-DSS Scope reduction.
---

## [SW-REQ-045] Payment Status Synchronization
**Statement:** WHEN a subscription transaction is initiated, the software shall synchronize the user's entitlement status (Free vs Paid) by processing asynchronous 'Payment Intent' success or failure events via Stripe webhooks.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Webhook simulation and database state check) |

**Notes:** Ensures reliable subscription status even if the user closes the browser during processing.
---

## [SW-REQ-046] Social Authentication
**Statement:** The software shall authenticate users via Google and Apple social login providers.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional / Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (OAuth2 flow validation) |

**Notes:** Ensures a low-friction onboarding process for new users.
---

## [SW-REQ-047] User Data Persistence
**Statement:** The software shall allow the user to save customized ingredient lists and diets to their authenticated profile.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Database persistence and retrieval check) |

**Notes:** Saved items must be private to the user per SW-REQ-043.
---

## [SW-REQ-048] Search History Retention
**Statement:** The software shall store and display the five (5) most recent unique search queries performed by the user, stored in client-side store (localStorage).

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Low |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (History stack limit check) |

**Notes:** Older history must be automatically purged as new searches are added.
---

## [SW-REQ-049] Search Favorites Management
**Statement:** The software shall allow the user to mark specific searches or results as 'Favorites' for quick retrieval.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Toggle state and retrieval check) |

**Notes:** Favorites differ from history as they are explicitly pinned by the user.
---

## [SW-REQ-050] Subscription Pricing Tiers
**Statement:** The software shall offer two paid subscription options: a monthly plan priced at $3.00 and an annual plan priced at $25.00.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Stripe product/price mapping check) |

**Notes:** Prices are inclusive of the yearly discount for the annual plan.
---

## [SW-REQ-051] Promotional Trial Logic
**Statement:** WHEN a new user authenticates via social login for the first time, the software shall grant a 7-day free trial of all Paid tier features.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Trial timestamp and entitlement check) |

**Notes:** Trial status must be tracked in the user profile and expire automatically after 168 hours.
---

## [SW-REQ-052] Paid Tier Exclusive Features
**Statement:** WHERE the user has an active Paid subscription or trial, the software shall enable access to 'Ingredient List' searches, 'Meal List' searches, and 'Diet Alternative' generation.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Entitlement-based feature activation) |

**Notes:** These are the primary value-add features for the subscription.
---

## [SW-REQ-053] Free Tier Functional Scope
**Statement:** IF the user is on the Free tier, THEN the software shall restrict search operations to single meal or single ingredient replacements only.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Verify blockage of list/diet search for free users) |

**Notes:** Combined with SW-REQ-042 (Daily limit), this defines the Free tier constraints.
---

## [SW-REQ-054] Administrative Access Control
**Statement:** The software shall provide a restricted Administration Panel accessible only to authenticated users with the 'Admin' role.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Role-Based Access Control check) |

**Notes:** Protects administrative functions from standard users.
---

## [SW-REQ-055] External Data Curation
**Statement:** WHEN an administrator selects an entry from an external API source (USDA or OpenFoodFacts), the software shall allow the administrator to edit, tag, and import that entry into the local database.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Import workflow validation) |

**Notes:** Essential for maintaining data quality and functionality tags.
---

## [SW-REQ-056] Manual Item Entry
**Statement:** The software shall allow administrators to manually create, update, and delete custom items, including their associated macronutrients, images, and category tags.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (CRUD operations on items) |

**Notes:** Used for items not found in external databases.
---

## [SW-REQ-057] Global Tag Management
**Statement:** The software shall provide tools within the admin panel to create and manage the global list of Category Tags and Functionality Tags.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Tag management UI check) |

**Notes:** Ensures consistency across the item database.
---

## [SW-REQ-058] Email and Password Registration
**Statement:** The software shall allow users to create an account by providing a unique email address and a password.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (User creation in DB) |

**Notes:** This acts as the primary alternative to Social Login (SW-REQ-046).
---

## [SW-REQ-059] Password Security (Hashing)
**Statement:** WHEN storing user passwords, the software shall use a cryptographically secure one-way hashing algorithm (e.g., Argon2 or bcrypt) with a unique salt per user.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Security Audit (Database record inspection) |

**Notes:** Raw passwords must never be stored in plain text to comply with security best practices and GDPR.
---

## [SW-REQ-060] Duplicate Email Prevention
**Statement:** IF a user attempts to register with an email address that already exists in the system, THEN the software shall prevent the account creation and display a "User already exists" error message.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional / Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Constraint validation) |

**Notes:** Prevents account hijacking and data duplication.
---

## [SW-REQ-061] Manual Authentication Login
**Statement:** WHEN a user provides a valid email and matching password, the software shall authenticate the session and grant access to the user profile.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Login success flow) |

**Notes:** Session management must remain consistent with social login sessions.
---

## [SW-REQ-062] API Authentication via JWT
**Statement:** The software shall authenticate API requests using JSON Web Tokens (JWT) stored in HttpOnly, Secure cookies with SameSite=Strict attribute.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Security Audit (Cookie attribute inspection and token validation) |

**Notes:** Prevents XSS-based token theft while maintaining stateless authentication benefits.
---

## [SW-REQ-063] JWT Token Expiration and Refresh
**Statement:** The software shall issue access tokens with a maximum validity of 15 minutes and refresh tokens with a maximum validity of 7 days, implementing automatic token rotation on refresh.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Token expiration validation) |

**Notes:** Short-lived access tokens minimize exposure window if compromised.
---

## [SW-REQ-064] Brute Force Protection
**Statement:** The software shall implement rate limiting that restricts failed login attempts to a maximum of 10 attempts per IP address per 10-minute window.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Rate limit trigger and reset validation) |

**Notes:** Protects against credential stuffing and brute force attacks.
---

## [SW-REQ-065] Account Lockout Policy
**Statement:** IF a user account experiences 5 consecutive failed login attempts, THEN the software shall temporarily lock the account for 15 minutes and notify the user via email.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Lockout trigger and duration validation) |

**Notes:** Provides per-account protection complementary to IP-based rate limiting.
---

## [SW-REQ-066] Session Timeout
**Statement:** The software shall automatically invalidate user sessions after 30 minutes of inactivity and require re-authentication.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Session expiration behavior) |

**Notes:** Reduces risk of session hijacking on shared or unattended devices.
---

## [SW-REQ-067] CSRF Protection
**Statement:** The software shall implement Cross-Site Request Forgery protection using synchronizer tokens for all state-changing operations.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Security Audit (CSRF token validation on form submissions) |

**Notes:** SameSite=Strict cookies provide additional CSRF mitigation.
---

## [SW-REQ-068] Security Headers
**Statement:** The software shall include the following HTTP security headers on all responses: Content-Security-Policy, X-Frame-Options (DENY), X-Content-Type-Options (nosniff), Referrer-Policy (strict-origin-when-cross-origin), and Permissions-Policy.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Security Audit (Header inspection) |

**Notes:** Mitigates clickjacking, XSS, and other client-side attacks.
---

## [SW-REQ-069] Password Reset Flow
**Statement:** WHEN a user requests a password reset, the software shall send a time-limited (1 hour) single-use reset link to the registered email address.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Reset token generation, expiration, and invalidation) |

**Notes:** Reset tokens must be cryptographically random and invalidated after use or expiration.
---

## [SW-REQ-070] Email Verification
**Statement:** WHEN a user registers with email and password, the software shall send a verification email and restrict access to paid features until the email address is verified.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional / Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Verification flow and feature restriction check) |

**Notes:** Prevents account creation with invalid or non-owned email addresses.
---

### 2.5 Compliance & Legal

---
## [SW-REQ-071] Medical Disclaimer Display
**Statement:** The software shall display a mandatory legal disclaimer stating that the application does not provide medical advice on the initial login screen and within the application 'About' section.

| Attribute | Value |
| :--- | :--- |
| **Type** | Safety / Legal |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Inspection (UI verification) |

**Notes:** Required for liability mitigation.
---

## [SW-REQ-072] Data Portability (Right to Access)
**Statement:** The software shall allow the user to export all personal data, including saved ingredients, diets, and search history, in machine-readable JSON and CSV formats.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional / Regulatory |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Verify export file schema and content completeness) |

**Notes:** Direct mapping to GDPR Article 20.
---

## [SW-REQ-073] Right to Erasure (Account Deletion)
**Statement:** WHEN a user confirms account deletion, the software shall permanently remove all associated Personally Identifiable Information (PII), private custom items, and historical logs from the active production database.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional / Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Database query post-deletion to confirm null records) |

**Notes:** Mapping to GDPR Article 17. Backup cycles must be accounted for in data retention policies.
---

## [SW-REQ-074] Explicit Consent Management
**Statement:** WHILE in the registration flow, the software shall require the user to provide an explicit opt-in (checkbox) for the Privacy Policy and Terms of Service before the account creation is finalized.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional / Regulatory |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Blockage of 'Register' button without active consent) |

**Notes:** Mapping to GDPR Article 7 (Conditions for consent).
---

## [SW-REQ-075] Data Encryption (PII Protection)
**Statement:** The software shall encrypt all Personal Identifiable Information (PII) at rest using AES-256 and during transit using TLS 1.3 or higher.

| Attribute | Value |
| :--- | :--- |
| **Type** | Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Security Audit (Penetration test/Protocol inspection) |

**Notes:** Mapping to GDPR Article 32 (Security of processing).
---

## [SW-REQ-076] API-First Mobile Readiness
**Statement:** The software backend shall expose all search, user, and subscription logic via a versioned RESTful or GraphQL API to facilitate future integration with native iOS and Android applications (e.g., Flutter).

| Attribute | Value |
| :--- | :--- |
| **Type** | Architecture / Scalability |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Inspection (API Documentation and Endpoint testing) |

**Notes:** Ensures no technical debt is created when expanding to native mobile platforms.
---

### 2.6 Error Handling & Resilience

---
## [SW-REQ-077] Network Failure Handling
**Statement:** WHEN a network request fails due to connectivity issues, the software shall display a user-friendly error message and provide a retry option without losing the current application state.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Simulated network failure scenarios) |

**Notes:** Preserves user work and provides clear recovery path.
---

## [SW-REQ-078] API Timeout Behavior
**Statement:** The software shall implement a 10-second timeout for all API requests and display a timeout notification to the user when exceeded.

| Attribute | Value |
| :--- | :--- |
| **Type** | Performance |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Timeout trigger validation) |

**Notes:** Prevents indefinite waiting states that degrade user experience.
---

## [SW-REQ-079] Graceful Degradation
**Statement:** WHEN a non-critical feature or service fails, the software shall continue operating with reduced functionality rather than displaying a full application error.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Service isolation validation) |

**Notes:** Critical features include search and authentication; non-critical include history sync and recommendations.
---

### 2.7 Non-Functional Requirements

---
## [SW-REQ-080] Search Response Time
**Statement:** The software shall return search results within 2 seconds for 95% of requests under normal load conditions.

| Attribute | Value |
| :--- | :--- |
| **Type** | Performance |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Load Test (P95 latency measurement) |

**Notes:** Measured from user input completion to result display.
---

## [SW-REQ-081] System Availability
**Statement:** The software shall maintain a minimum availability of 99.9% uptime, excluding scheduled maintenance windows.

| Attribute | Value |
| :--- | :--- |
| **Type** | Non-Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Monitoring (Uptime tracking over 30-day periods) |

**Notes:** Translates to maximum 8.76 hours of unplanned downtime per year.
---

## [SW-REQ-082] Concurrent User Capacity
**Statement:** The software shall support a minimum of 1000 concurrent active users without performance degradation.

| Attribute | Value |
| :--- | :--- |
| **Type** | Performance / Scalability |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Load Test (Concurrent user simulation) |

**Notes:** Infrastructure should be horizontally scalable to accommodate growth.
---

## [SW-REQ-083] Data Backup and Recovery
**Statement:** The software shall perform automated database backups every 24 hours with a retention period of 30 days, enabling point-in-time recovery.

| Attribute | Value |
| :--- | :--- |
| **Type** | Non-Functional |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Inspection (Backup logs and recovery drill) |

**Notes:** Recovery Time Objective (RTO) should not exceed 4 hours.
---

## [SW-REQ-084] Application Logging
**Statement:** The software shall log all authentication events, API requests, errors, and administrative actions with timestamps and user identifiers to a centralized logging system.

| Attribute | Value |
| :--- | :--- |
| **Type** | Non-Functional / Security |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Inspection (Log completeness audit) |

**Notes:** Logs must be retained for a minimum of 90 days for security audit purposes.
---

### 2.8 Accessibility

---
## [SW-REQ-085] WCAG 2.1 AA Compliance
**Statement:** The software shall conform to Web Content Accessibility Guidelines (WCAG) 2.1 Level AA for all user-facing interfaces.

| Attribute | Value |
| :--- | :--- |
| **Type** | Accessibility |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | Accessibility Audit (Automated and manual testing) |

**Notes:** Includes requirements for color contrast, screen reader compatibility, and focus management.
---

## [SW-REQ-086] Keyboard Navigation
**Statement:** The software shall support full keyboard navigation for all interactive elements, including search, filtering, and result selection.

| Attribute | Value |
| :--- | :--- |
| **Type** | Accessibility |
| **Priority** | High |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Keyboard-only navigation validation) |

**Notes:** Extends SW-REQ-009 to cover all application areas, not just autocomplete.
---

### 2.9 Offline Behavior & Edge Cases

---
## [SW-REQ-087] Connection Loss During Search
**Statement:** IF the user loses network connectivity during an active search operation, THEN the software shall preserve the search query and display a reconnection prompt, automatically retrying upon connectivity restoration.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Integration Test (Network interruption simulation) |

**Notes:** Uses browser online/offline events for detection.
---

## [SW-REQ-088] Offline Cache Behavior
**Statement:** WHILE offline, the software shall serve cached search results and user data from localStorage, displaying a visual indicator that the application is operating in offline mode.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional |
| **Priority** | Low |
| **Feasibility** | Feasible |
| **Verification** | UI Test (Offline mode indicator and cached data display) |

**Notes:** Builds upon the caching mechanism defined in SW-REQ-003.
---

## [SW-REQ-089] Micronutrient Nomenclature Standardization
**Statement:** The software shall validate all micronutrient keys against a predefined standardized vocabulary (e.g., allowing "Sodium" but rejecting "Na") before they are stored or assigned to an item.

| Attribute | Value |
| :--- | :--- |
| **Type** | Functional / Data Integrity |
| **Priority** | Medium |
| **Feasibility** | Feasible |
| **Verification** | Unit Test (Attempt to save valid and invalid micronutrient keys; verify invalid keys throw an error or are rejected). |

**Notes:** The predefined vocabulary/dictionary of allowed micronutrients shall be maintained in a central configuration or database table to ensure data consistency across all items.
---

## 3. Changelog

### 2026-01-18

* Added
- Document created

### 2026-01-20

* Added
- SW-REQ-089 - added Style Guide requirement

* Changed
- SW-REQ-013 - sidebar on the left side
- SW-REQ-015 - mode toggle in sidebar

### 2026-05-18

* Added
- SW-REQ-089 - micronutrient standardization

* Changed
- SW-REQ-038 - now specifically mentions the storage and algo exclusion
