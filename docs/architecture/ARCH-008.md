# [ARCH-008] - User Profile Module

**Description:** Service managing user preferences, saved data, search history, favorites, and data export/deletion for GDPR compliance.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | ProfileController, PreferenceManager, SavedDataRepository, DataExporter, AccountDeleter |
| **Dependencies** | ARCH-005 (Data Repository), ARCH-006 (Authentication) |
| **Traceability** | SW-REQ-043, SW-REQ-047, SW-REQ-048, SW-REQ-049, SW-REQ-072, SW-REQ-073, SW-REQ-074 |

**Dynamic Behavior:**

- **Data Isolation:** Enforces user-scoped queries for all custom items and saved data. Cross-user access prevented at repository level.
- **Preference Propagation:** Updates to unit preferences trigger real-time recalculation across all displayed data (SW-REQ-041).
- **Data Export:** Generates JSON and CSV exports containing all user PII, saved items, diets, and history.
- **Account Deletion:** Permanently removes all PII and associated data from production database. Cascades to all related records.

**Interface Definition:**

- `Input`: User ID context, preference updates, export/delete requests
- `Output`: User profiles, exported data files, deletion confirmations

**Alternative Analysis (BP6):**

- *Chosen Approach:* Server-side profile storage with client-side history caching
- *Alternative Considered:* Fully client-side profile storage (localStorage only)
- *Trade-off:* Server-side storage enables cross-device sync and proper GDPR compliance (SW-REQ-072, SW-REQ-073). Pure client-side would lose data on device change and complicate data export requests. Hybrid approach uses localStorage for recent history (SW-REQ-048) while server stores persistent data.
