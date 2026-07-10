# [ARCH-015] - Compliance Module

**Description:** Service handling legal and regulatory requirements including GDPR compliance, consent management, and disclaimer display.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | ConsentManager, DisclaimerRenderer, DataRetentionPolicy, BackupManager |
| **Dependencies** | ARCH-005 (Data Repository), ARCH-008 (User Profile) |
| **Traceability** | SW-REQ-071, SW-REQ-072, SW-REQ-073, SW-REQ-074, SW-REQ-083 |

**Dynamic Behavior:**

- **Consent Capture:** Blocks registration completion until Privacy Policy and ToS checkboxes are explicitly checked.
- **Disclaimer Display:** Publishes medical-disclaimer information in the Terms of Service and, when implemented, the About section.
- **Data Retention:** Enforces 30-day backup retention with point-in-time recovery capability.
- **Erasure Processing:** Coordinates complete data deletion across primary database and schedules backup purge.

**Interface Definition:**

- `Input`: Consent status, deletion requests, backup schedules
- `Output`: Consent records, disclaimer content, backup status

**Alternative Analysis (BP6):**

- *Chosen Approach:* Integrated compliance module with automated retention policies
- *Alternative Considered:* Manual compliance processes with external legal review
- *Trade-off:* Automated compliance ensures consistent enforcement of GDPR requirements (SW-REQ-072, SW-REQ-073, SW-REQ-074) without human error. Manual processes would require dedicated staff and risk non-compliance. Automation also enables faster response to data subject requests.
