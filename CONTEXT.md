# Mealswapp

Mealswapp helps people find nutritionally similar food substitutions and meal alternatives while preserving practical culinary use.

## Language

### Food Objects

**Food Object**:
Any searchable nutritional object. A Food Object is either a Food Item or a Meal.

**Catalog Food Object**:
A curated Food Object available to all users.

**Private Food Object**:
A Food Object created by one user and visible only to that user.

**Curation**:
The administrator review and refinement of a Food Object before publishing it as a Catalog Food Object.

**Import**:
The act of bringing external food data into Curation.

**Food Item**:
A basic ingredient or standalone food, such as olive oil or chicken breast.

**Meal**:
A Food Object representing a prepared dish or serving-level composition. Its Ingredients may be unknown when only aggregate nutrition is available. A Meal may be a Composite Meal when its Ingredient Entries are explicit, but a Meal never contains another Meal.

**Composite Meal**:
A Meal defined by explicit Ingredient Entries. A prepared dish with only aggregate nutrition is a Meal, not a Composite Meal.

**Ingredient**:
A Food Item used as part of a Composite Meal.

**Ingredient Entry**:
One occurrence of an Ingredient within a Composite Meal, with its own Food Quantity and position. The same Food Item may appear in multiple Ingredient Entries.

**Ingredient List**:
A user-selected collection of Food Items. An Ingredient List does not imply that the Food Items form a prepared Meal, and it may be used as a source of Substitution Inputs.

**Daily Diet**:
A collection of Meals representing one day of eating.
_Avoid_: Diet, meal set, Meal List except when naming the UI search mode

**Substitute**:
A Food Object proposed in place of Substitution Inputs because it is nutritionally similar and satisfies applicable Search constraints. In single-input searches, shared Culinary Roles may improve ordering.
_Avoid_: Substitution, alternative, similarity result

**Alternative Daily Diet**:
A generated substitute for an entire Daily Diet.

**Culinary Role**:
A classification applying to a Food Object and describing how it is used in cooking or eating, such as frying fat, sweetener, or crunchy snack. A Food Object may have multiple Culinary Roles. Explicit Search filters may include Culinary Roles; selected includes must all match. A Culinary Role shared with one Substitution Input is only an ordering signal. Excluded Culinary Roles are expressed through Exclusion Rules.
_Avoid_: Functionality Tag

**Food Category**:
A classification applying to a Food Object and describing what it is, such as meat, fish, dairy, vegetable, or pasta dish. A Food Object may have multiple Food Categories. Explicit Search filters may include Food Categories; selected includes must all match. Excluded Food Categories are expressed through Exclusion Rules.
_Avoid_: Category Tag

**Exclusion Rule**:
A hard Search constraint that excludes Food Categories, Food Objects, or Allergens. If an Exclusion Rule targets an Allergen, Food Objects marked Contains Allergen, May Contain Allergen, or Unknown Allergen Status for that Allergen are excluded. If an Exclusion Rule directly conflicts with explicit Search constraints before results are computed, the Search is rejected with user-facing feedback; otherwise the Search returns any valid remaining results.
_Avoid_: Dietary Rule

**Dietary Preset**:
A user-facing bundle, such as vegetarian or pescatarian, that produces one or more Exclusion Rules.

**Allergen**:
A named allergenic substance or group, such as peanuts, milk, or tree nuts. Food Objects declare applicable Allergens independently from Food Categories.

**Contains Allergen**:
A Food Object is known to contain an Allergen.

**May Contain Allergen**:
A Food Object may contain an Allergen due to possible presence or cross-contamination.

**Unknown Allergen Status**:
A Food Object has insufficient information to determine whether an Allergen is present.

### Users

**User Account**:
A person's durable Mealswapp identity.
_Avoid_: User, account, AuthUser when discussing the domain concept

**Profile**:
User-facing details and preferences attached to a User Account.

**Login Method**:
A credential source linked to a User Account. It may be an email-and-password method or an external provider method such as Google or Apple. A User Account does not require an email-and-password Login Method.

**External Login Identity**:
An identity asserted by an external login provider and linked to a User Account. A User Account may have multiple Login Methods.
_Avoid_: OAuth identity when discussing the domain concept

**Login Method Linking**:
The act of attaching an additional Login Method to an existing User Account. Matching email addresses alone do not authorize linking; the person must authenticate an existing Login Method.

**Session**:
A temporary authenticated relationship between a browser and a User Account. A User Account may have multiple Sessions on different devices.

**Verified Login Method**:
A Login Method whose email ownership has been established. An email-and-password Login Method requires Mealswapp email verification; an External Login Identity relies on verification asserted by its provider.

**Unverified Login Method**:
An email-and-password Login Method that has not completed Mealswapp email verification. It may be used to sign in, but does not unlock paid features.

A User Account unlocks paid features when it has at least one Verified Login Method. Other linked Login Methods may remain unverified.

**Registration Consent**:
The recorded acceptance of the current Privacy Policy and Terms of Service versions required to create a User Account. Later re-acceptance, if introduced, is a separate concept.

**Account Erasure**:
The irreversible workflow that permanently removes a User Account's personal data and private data from active production systems while retaining only legally justified pseudonymous evidence.
_Avoid_: Account deletion, data deletion, GDPR deletion

**Erasure Receipt**:
A pseudonymous record that an Account Erasure was requested and completed or failed, without retaining the deleted person's identity or account data.

**Account Export**:
A machine-readable package of a User Account's personal data, saved data, Private Food Objects, and search history.
_Avoid_: Export bundle

**Saved Item**:
A user-owned saved reference to a Food Object, Ingredient List, or Daily Diet. Use a more specific term, such as Favorite, when behavior differs.

**Favorite**:
A bookmarked Food Object. Ingredient Lists and Daily Diets may be saved, but they are not Favorites.

**Search History Entry**:
One completed search retained for a User Account after valid results are returned. Repeated identical searches remain separate entries.

Anonymous visitors may have temporary browser-local search state, but Saved Items, Search History Entries, Profiles, and Private Food Objects belong only to a User Account.

### Search

**Search**:
A request to retrieve Food Objects or Substitutes according to a selected Search Mode, query, and applicable constraints.

**Search Mode**:
One of the domain search operations: Catalog Search, Substitution Search, or Daily Diet Alternative Search. The UI may expose these as a continuous flow rather than literal mode toggles.

**Rejected Search**:
A Search that is not executed because its inputs are invalid or contradictory, with feedback explaining the reason.

**Catalog Search**:
A Search that retrieves Food Objects by name text and applicable constraints. Food Categories and Culinary Roles are explicit constraints rather than text-matching fields. Catalog Search may be used to discover inputs for a Substitution Search.

**Catalog Search Result**:
A Food Object returned by Catalog Search. Results include both Food Items and Meals and identify which subtype each result represents. A result may be browsed directly or selected as an input to a Substitution Search.

**Substitution Input**:
A Food Object and Food Quantity supplied to a Substitution Search. When first selected, it uses one known Serving when available; otherwise it uses the Food Object's Nutrition Basis. The quantity may be edited. Duplicate selections of the same Food Object are merged by summing their quantities.

**Substitution Search**:
A Search that accepts one or more Substitution Inputs and returns Food Objects as Substitutes. Results may be Food Items or Meals regardless of input subtype; constraints and ranking determine suitability. Input Food Objects are excluded from results. Adding further Substitution Inputs refines the requested substitution rather than changing to a separate Search Mode. With one Substitution Input, shared Culinary Roles improve ordering without changing Nutritional Similarity. With multiple Substitution Inputs, their Food Quantities determine the combined Macro Profile used for Nutritional Similarity and their individual Culinary Roles do not affect ordering.

**Food Object Type Filter**:
An optional Catalog Search or Substitution Search constraint limiting results to Food Items or Meals. By default, both Food Object types are included.

**Daily Diet Alternative Search**:
A Search that accepts a Daily Diet and generates Alternative Daily Diets. Unlike a Substitution Search, it returns collections of Meals rather than Food Objects.

Catalog Search, Substitution Search, and Daily Diet Alternative Search all apply Exclusion Rules.

Contradictory Search constraints are rejected before results are computed. The rejection must explain the conflict to the user.

A valid Search may return no results. Empty results are not a rejection when the constraints are consistent.

### Nutrition

**Macro Profile**:
The protein, carbohydrate, and fat composition of a Food Object. Nutritional similarity compares Macro Profiles; calories are derived data and are not part of the comparison profile.
_Avoid_: Macros, macro vector, nutritional profile

**Nutritional Similarity**:
How closely two Food Objects match in the proportions of protein, carbohydrates, and fat, independent of serving quantity. Culinary Role affects the ordering of suitable Substitutes, not Nutritional Similarity itself. The selected Matched Quantity target does not affect Nutritional Similarity.
_Avoid_: Similarity score when discussing the domain concept

**Matched Quantity**:
The amount of a Substitute needed to match a selected nutritional target of the Substitution Inputs. The user selects calories, protein, carbohydrates, or fat; calories are the default. A Food Object is excluded when no finite Matched Quantity exists for the selected target.

**Nutrition Basis**:
The standard quantity used to express a Food Object's nutritional values: `100 g` for solids and `100 ml` for liquids.
_Avoid_: Storage basis, normalization basis

**Physical State**:
Whether a Food Object is solid or liquid, determining whether its Nutrition Basis is `100 g` or `100 ml`.

**Food Quantity**:
An amount of a Food Object expressed in a supported unit. It can be converted to the Food Object's Nutrition Basis when enough conversion data exists.

**Serving**:
A Food Object-specific Food Quantity with enough recorded data to convert it to the Nutrition Basis. Descriptive serving text without conversion data must not be used for nutritional calculations.

**Density**:
The grams-per-milliliter conversion factor for a liquid Food Item. Density is required when converting a liquid Ingredient into mass for Composite Meal calculations; it must not be guessed as `1 ml = 1 g`.

**Micronutrient**:
A supplemental nutrient, such as sodium or vitamin C, recorded for display and export but excluded from Nutritional Similarity.
