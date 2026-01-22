# [ARCH-009] - Administration Module

**Description:** Restricted backend service providing administrative functions for data curation, user management, and global tag management. Acts as a proxy for external data searches to enable admin-curated imports.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | AdminController, DataImporter, ItemCurator, TagManager, UserAdminPanel, ExternalSearchProxy |
| **Dependencies** | ARCH-005 (Data Repository), ARCH-006 (Authentication), ARCH-012 (External Data Integration) |
| **Traceability** | SW-REQ-054, SW-REQ-055, SW-REQ-056, SW-REQ-057 |

**Dynamic Behavior:**

- **Access Control:** Validates 'Admin' role on all requests. Returns 403 Forbidden for non-admin users.
- **External Data Search (SW-REQ-055):** Admin UI provides a dedicated search interface that queries external APIs (not the local database). Flow:
  1. Admin enters search term in "External Import" panel
  2. `ExternalSearchProxy` routes request to ARCH-012 (External Data Integration)
  3. ARCH-012 queries USDA and/or OpenFoodFacts APIs
  4. Results displayed in admin UI with "Import" action for each item
  5. Admin selects item, edits fields (name, tags, macros), and confirms import
  6. `DataImporter` saves curated item to local database via ARCH-005
- **Item CRUD:** Full create/update/delete capabilities for food items including macros, images, and tags.
- **Tag Management:** Creates and manages global Category Tags and Functionality Tags used across all items.

**Interface Definition:**

- `Input`: Admin-authenticated requests, external search queries, item definitions
- `Output`: External search results (uncurated), curated items (post-import), tag hierarchies, admin audit logs

**Admin External Search Flow:**

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Admin UI   │────>│  ARCH-009   │────>│  ARCH-012   │────>│ USDA/OFF    │
│ (Search)    │     │ (Proxy)     │     │ (External)  │     │ (APIs)      │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │                   │
       │<──────────────────┴───────────────────┴───────────────────│
       │              Normalized results for curation              │
       │                                                           │
       ▼                                                           │
┌─────────────┐     ┌─────────────┐                               │
│ Edit & Tag  │────>│  ARCH-005   │  (Save curated item)          │
│ (Admin)     │     │ (Repository)│                               │
└─────────────┘     └─────────────┘                               │
```

**Alternative Analysis (BP6):**

- *Chosen Approach:* Integrated admin module within main application backend
- *Alternative Considered:* Separate admin microservice with dedicated database access
- *Trade-off:* Integrated module simplifies deployment and shares data models with main application. Separate microservice would add network latency and deployment complexity for minimal security benefit (RBAC already enforces access). Admin operations are low-frequency and don't require independent scaling.
