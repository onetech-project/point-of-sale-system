# Feature Specification: Indonesian Data Protection Compliance (UU PDP)

**Feature Branch**: `006-uu-pdp-compliance`  
**Created**: January 2, 2026  
**Status**: Draft  
**Input**: User description: "Indonesian data protection compliance (UU PDP No.27 Tahun 2022) with encryption at rest, audit trails, consent management, and data privacy controls"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Platform Owner Compliance Audit (Priority: P1)

As a platform owner, I need to ensure all personally identifiable information (PII) is encrypted at rest and protected from unauthorized access, so that I comply with Indonesian data protection law (UU PDP No.27 Tahun 2022) and avoid legal penalties.

**Why this priority**: Legal compliance is mandatory. Non-compliance can result in fines up to IDR 6 billion and potential criminal liability under UU PDP. This is the foundation for all other privacy features.

**Independent Test**: Can be fully tested by inspecting data storage, verifying encryption is active for all PII fields, checking log outputs for masked data, and confirming no plaintext sensitive data exists in storage or logs.

**Acceptance Scenarios**:

1. **Given** the system is storing user account data, **When** I inspect data storage directly, **Then** all PII fields (email, names, phone, addresses, tokens, API keys) are encrypted and cannot be read as plaintext
2. **Given** a user registration occurs, **When** I review application logs, **Then** sensitive data (email, password, tokens) is masked (e.g., "us**_@example.com", "tok\_**xyz") and never appears in plaintext
3. **Given** a guest order is created, **When** I check customer order information and delivery location data, **Then** customer PII (name, phone, email, address, coordinates, IP address) is encrypted at rest
4. **Given** a tenant configures payment processing, **When** I inspect tenant payment configuration data, **Then** payment gateway API credentials are encrypted and not visible in plaintext
5. **Given** any data backup is created, **When** I examine the backup file, **Then** all PII remains encrypted within the backup

---

### User Story 2 - Tenant Data Management Rights (Priority: P1)

As a tenant (business owner), I need to view, update, and delete my own data and my team members' data, so that I can exercise my data rights under UU PDP and maintain control over my business information.

**Why this priority**: Data subjects have fundamental rights under UU PDP Articles 3-6. Tenants must be able to access and control their data. This is legally required and affects all tenant users.

**Independent Test**: Can be fully tested by logging in as a tenant owner, accessing data management UI, performing CRUD operations on tenant data, and verifying changes persist correctly with proper audit trails.

**Acceptance Scenarios**:

1. **Given** I am logged in as a tenant owner, **When** I navigate to account settings, **Then** I can view all my tenant's data including business profile, team members, and configuration in a readable format
2. **Given** I am viewing my tenant data, **When** I request to update information (business name, contact details, preferences), **Then** the system updates the data and records the change in audit logs
3. **Given** I want to remove a team member, **When** I perform a soft delete on their account, **Then** the user is marked as deleted but data is retained, and I receive confirmation
4. **Given** I need to permanently remove data, **When** I request hard delete with confirmation, **Then** the user data is permanently removed from the system and audit trail records the deletion
5. **Given** I update any tenant data, **When** I check the audit trail, **Then** I see a log entry showing who made the change, what was changed, when, and from which IP address

---

### User Story 3 - Guest Customer Data Access and Deletion (Priority: P1)

As a guest customer who placed an order without creating an account, I need to view my order history with all my personal data and request deletion of my personal information, so that I can exercise my privacy rights without affecting the merchant's order records.

**Why this priority**: Guest customers are data subjects under UU PDP with equal rights. The law requires providing access and deletion rights to all individuals, not just registered users. This affects all e-commerce transactions.

**Independent Test**: Can be fully tested by placing a guest order, accessing the order via unique link, viewing personal data, requesting deletion, and verifying personal data is removed while order history remains for merchant.

**Acceptance Scenarios**:

1. **Given** I placed a guest order, **When** I access the order using my order reference and email/phone, **Then** I can view all my personal data (name, phone, email, delivery address, IP address)
2. **Given** I am viewing my guest order, **When** I click "Request Data Deletion", **Then** I see a clear explanation of what will be deleted (personal info) and what will be retained (order history for merchant)
3. **Given** I confirm data deletion request, **When** the system processes the request, **Then** my personal data (name, phone, email, address, coordinates) is anonymized (replaced with "Deleted User", masked values, or null values)
4. **Given** my personal data is deleted, **When** the merchant views the order, **Then** the order record still exists with order items and amounts but shows anonymized customer info
5. **Given** I request data deletion, **When** the deletion is complete, **Then** an audit log entry records the deletion request timestamp, order reference, and deletion confirmation

---

### User Story 4 - Persistent Audit Trail for Compliance (Priority: P2)

As a platform owner or compliance officer, I need a comprehensive audit trail of all access and modifications to sensitive data, so that I can investigate security incidents, demonstrate compliance to regulators, and track data processing activities.

**Why this priority**: UU PDP requires documentation of data processing activities. Audit trails are essential for regulatory compliance, security investigations, and demonstrating accountability. This supports all other privacy features.

**Independent Test**: Can be fully tested by performing various data operations (create, read, update, delete) and verifying each action is logged with complete context (who, what, when, where, why).

**Acceptance Scenarios**:

1. **Given** any user accesses PII, **When** the access occurs, **Then** an audit log entry is created containing timestamp, user ID, tenant ID, data type accessed, action performed, IP address, and session ID
2. **Given** sensitive data is modified, **When** the modification is saved, **Then** the audit log records the before/after values (encrypted), who made the change, and justification if required
3. **Given** data deletion occurs (soft or hard delete), **When** the deletion is processed, **Then** the audit log permanently records what was deleted, by whom, when, and cannot be modified or deleted
4. **Given** I am a platform owner, **When** I query the audit trail system, **Then** I can filter by user, tenant, date range, action type, and data category to investigate specific incidents
5. **Given** a failed authentication attempt occurs, **When** someone tries to access protected data without authorization, **Then** the failed attempt is logged with IP address, attempted user ID, and timestamp
6. **Given** audit logs are created, **When** the system performs maintenance or backup, **Then** audit logs are immutable and cannot be altered or deleted (append-only storage)

---

### User Story 5 - Consent Collection and Management (Priority: P1)

As the platform, I need to collect explicit consent from tenants during registration and from customers during checkout for data processing activities, so that I comply with UU PDP consent requirements and have legal basis for data processing.

**Why this priority**: UU PDP Article 20 requires explicit consent as legal basis for data processing. Without documented consent, all data processing is potentially illegal. This must be implemented before any data collection.

**Independent Test**: Can be fully tested by registering a new tenant and placing a guest order, verifying consent checkboxes appear, are required for critical operations, optional for others, and consent records are stored with timestamps.

**Acceptance Scenarios**:

1. **Given** I am registering as a new tenant, **When** I reach the registration form, **Then** I see clear consent checkboxes for: (a) tenant operational data processing [required], (b) service analysis and improvement [required], (c) advertising and promotion [optional], and (d) third-party integration (Midtrans) [required]
2. **Given** I am on the registration form, **When** I try to register without checking required consent boxes, **Then** the form prevents submission and shows error messages indicating which consents are mandatory
3. **Given** I check all consent boxes and register, **When** the registration completes, **Then** the system records my consent with timestamp, IP address, user agent, and consent version number
4. **Given** I am a guest customer at checkout, **When** I reach the checkout page, **Then** I see consent checkboxes for: (a) order processing and delivery [required], (b) email communications about my order [required], (c) promotional communications [optional], and (d) payment processing via Midtrans [required]
5. **Given** I am a guest customer, **When** I try to complete checkout without checking required consent boxes, **Then** the checkout is blocked and I receive clear explanation of why each consent is needed
6. **Given** I complete checkout with consent, **When** the order is created, **Then** my consent is recorded with order reference, timestamp, IP address, and linked to the order
7. **Given** I previously gave consent, **When** I access my account settings (tenant) or order details (guest), **Then** I can view my consent history and revoke optional consents

---

### User Story 6 - Privacy Policy Transparency (Priority: P2)

As a tenant or customer, I need to read a clear privacy policy that explains what data is collected, why it's collected, how it's used, how long it's retained, and how to exercise my rights, so that I understand my privacy rights under UU PDP.

**Why this priority**: UU PDP Article 6 requires transparency about data processing. Privacy policy must exist before collecting any data. This is legally required and builds user trust.

**Independent Test**: Can be fully tested by accessing the privacy policy page, verifying all required disclosures are present, links work, contact information is provided, and policy is available in Indonesian language.

**Acceptance Scenarios**:

1. **Given** I am on the registration or checkout page, **When** I look for privacy information, **Then** I see a prominent link to the Privacy Policy page
2. **Given** I click the Privacy Policy link, **When** the page loads, **Then** I see sections covering: (a) what data is collected, (b) purposes of data processing, (c) legal basis (consent), (d) data retention periods, (e) third-party sharing (Midtrans only), (f) data security measures, (g) my rights (access, correction, deletion), and (h) how to contact the data controller
3. **Given** I am reading the privacy policy, **When** I review the data usage section, **Then** I see clearly categorized purposes: (a) Tenant Operations - managing orders, inventory, users, (b) Service Analysis - improving features and performance, (c) Advertising & Promotion - marketing communications [optional], (d) Third-Party Integration - Midtrans payment processing with link to Midtrans privacy policy
4. **Given** I want to exercise my rights, **When** I read the "Your Rights" section, **Then** I see instructions for how to access, update, or delete my data, with specific email contact and expected response time (14 days per UU PDP Article 6)
5. **Given** the privacy policy exists, **When** regulations or practices change, **Then** the policy includes version number, last updated date, and mechanism to notify users of material changes

---

### User Story 7 - Revoke Optional Consent (Priority: P3)

As a tenant or customer, I need to revoke optional consents (analytics, advertising) at any time, so that I can change my privacy preferences and control how my data is used beyond mandatory operational purposes.

**Why this priority**: While consent collection (P1) is critical, consent revocation is required by UU PDP Article 21 but is typically used less frequently. Users must be able to withdraw consent, but this can be implemented after core consent collection.

**Independent Test**: Can be fully tested by giving initial consent during registration/checkout, then accessing account/order settings, revoking optional consents, and verifying the system respects the revocation and stops processing for those purposes.

**Acceptance Scenarios**:

1. **Given** I am a tenant with active account, **When** I navigate to Privacy Settings, **Then** I see my current consent status for: operational data (cannot revoke), analytics (can revoke), advertising (can revoke), payment integration (cannot revoke while using payment)
2. **Given** I am viewing consent settings, **When** I uncheck "Analytics and Service Improvement" consent, **Then** the system saves my preference, stops collecting analytics data for my account, and shows confirmation message
3. **Given** I am a guest customer, **When** I access my order via order reference and email, **Then** I see a Privacy Settings section showing my consents: order processing (cannot revoke), promotional emails (can revoke)
4. **Given** I revoke promotional email consent, **When** the system processes the revocation, **Then** my email is removed from marketing lists, preference is saved with revocation timestamp, and audit log records the action
5. **Given** I revoked consent for analytics, **When** I later want to opt back in, **Then** I can re-enable the consent and the system records the new consent grant with timestamp

---

### User Story 8 - Data Retention and Automated Cleanup (Priority: P3)

As the platform owner, I need automated cleanup of expired temporary data (tokens, sessions, reservations) and enforcement of data retention policies, so that I don't retain personal data longer than necessary per UU PDP principles of data minimization.

**Why this priority**: UU PDP requires limiting data retention to what's necessary. While important for compliance, automated cleanup can be implemented after core encryption and consent features are working. Manual processes can serve as interim solution.

**Independent Test**: Can be fully tested by creating temporary data with expiration dates, waiting for expiration, running cleanup processes, and verifying expired data is properly removed while respecting retention policies.

**Acceptance Scenarios**:

1. **Given** verification tokens are created, **When** they expire (24 hours), **Then** an automated cleanup process removes expired tokens within 48 hours
2. **Given** password reset tokens are used, **When** the token is consumed, **Then** it is immediately marked as used and removed within 24 hours
3. **Given** guest sessions are created, **When** sessions exceed 30 days with no activity, **Then** session records are automatically purged while preserving audit trail entries
4. **Given** a tenant account is soft-deleted, **When** the 90-day grace period expires, **Then** the system prompts for final confirmation and proceeds with hard delete if no response
5. **Given** guest orders are completed, **When** the legal retention period (5 years for tax purposes in Indonesia) expires, **Then** order records are automatically hard-deleted with audit trail entry

---

### Edge Cases

- What happens when encryption key needs to be rotated? System must re-encrypt all sensitive data with new key without downtime or data loss.
- How does system handle audit log storage reaching capacity? Implement log rotation and archival to long-term storage with integrity verification.
- What if a user requests data deletion but has pending legal obligations (unpaid orders, disputes)? System must block deletion and inform user of outstanding obligations.
- What happens when a guest customer loses their order reference link? Provide alternative verification method using email + phone verification code.
- How does system handle GDPR requests from international users? Privacy policy clarifies Indonesian law applies, but system should support similar data export/deletion requests.
- What if tenant revokes payment integration consent but has active payment configurations? System must warn about service disruption and require confirmation before processing revocation.
- How are deleted tenant's customer data (guest orders) handled? Customer data belongs to transaction records and must be retained per retention policy, but tenant access is removed.
- How does audit log handle system actions (automated processes) vs user actions? System actions are logged with service account identifier and automation reason.
- What if multiple users access same sensitive data concurrently during encryption migration? System must ensure data consistency and prevent race conditions.
- How does system handle partial consent (user consents to some but not all optional items)? System stores individual consent preferences and respects each one independently.

## Requirements _(mandatory)_

## 2. Functional Requirements

### Terminology

- **Soft Delete**: Setting status='deleted' flag while retaining data for grace period (90 days for tenants)
- **Hard Delete**: Permanent removal of all data from database (no recovery possible)
- **Anonymization**: Replacing PII with generic values ("Deleted User") while preserving non-PII transaction data
- **PII (Personal Identifiable Information)**: Email, name, phone, address, IP, verification tokens, session IDs
- **Masking**: Obfuscating sensitive data in logs (e.g., "us**\*@example.com", "\*\*\*\***1234")

### Data Encryption at Rest

- **FR-001**: System MUST encrypt all PII in user account data at rest: email, first name, last name, verification tokens
- **FR-002**: System MUST encrypt all PII in customer order information at rest: customer name, customer phone, customer email, IP address, session identifier
- **FR-003**: System MUST encrypt all PII in delivery location data at rest: address text, geographic coordinates, geocoded addresses
- **FR-004**: System MUST encrypt password reset tokens at rest
- **FR-005**: System MUST encrypt sensitive data in user invitations at rest: email addresses, invitation tokens
- **FR-006**: System MUST encrypt sensitive data in user sessions at rest: IP addresses, session identifiers
- **FR-007**: System MUST encrypt sensitive data in notification records at rest: recipient information, message body, metadata containing PII
- **FR-008**: System MUST encrypt payment gateway credentials at rest: API keys, merchant identifiers, authentication tokens
- **FR-009**: System MUST store encryption keys securely outside primary data storage with restricted access controls
- **FR-010**: System MUST ensure each encryption operation uses unique cryptographic parameters to prevent pattern detection
- **FR-011**: System MUST handle encryption/decryption transparently in application layer without requiring changes to business logic
- **FR-012**: System MUST maintain data integrity verification for all encrypted data

#### Log Data Masking

- **FR-013**: System MUST mask email addresses in logs by showing only first 2 characters and domain (e.g., "us\*\*\*@example.com")
- **FR-014**: System MUST mask phone numbers in logs by showing only last 4 digits (e.g., "**\*\***1234")
- **FR-015**: System MUST mask tokens in logs by showing only first 3 and last 3 characters (e.g., "abc\*\*\*xyz")
- **FR-016**: System MUST mask IP addresses in logs by showing only first two segments (e.g., "192.168._._")
- **FR-017**: System MUST mask API keys/credentials in logs completely (e.g., "**_REDACTED_**")
- **FR-018**: System MUST mask personal names in logs using first name initial only (e.g., "J**_ D_**")
- **FR-019**: System MUST apply masking to all log levels (debug, info, warn, error) before writing to log output
- **FR-020**: System MUST NOT log password hashes or any password-related data even in masked form

#### Tenant Data Management

- **FR-021**: System MUST provide tenant owners with a data management interface to view all their tenant's data
- **FR-022**: System MUST allow tenant owners to update their business profile data (business name, contact info, preferences)
- **FR-023**: System MUST allow tenant owners to view and manage team member accounts
- **FR-024**: System MUST allow tenant owners to soft delete team member accounts (marks as deleted but retains data)
- **FR-025**: System MUST allow tenant owners to request hard delete of team member accounts with explicit confirmation
- **FR-026**: System MUST prevent tenant owners from deleting their own account if they are the last owner
- **FR-027**: System MUST record all data access and modifications in audit trail with tenant context

#### Guest Customer Data Rights

- **FR-028**: System MUST verify guest identity using order reference AND (email OR phone) before allowing data access or deletion. Verification logic: (order_reference matches) AND ((email matches) OR (phone matches)) - either contact method is sufficient.
- **FR-029**: System MUST display guest's PII in a dedicated "Your Data" section: name, phone, email, delivery address, order history
- **FR-030**: System MUST provide a "Request Data Deletion" button on guest order details page
- **FR-031**: System MUST display clear explanation of data deletion consequences before confirming: "Personal info will be deleted, but order record will be retained for merchant with anonymized customer info"
- **FR-032**: System MUST anonymize guest PII upon deletion request: name becomes "Deleted User", phone/email/IP become null
- **FR-033**: System MUST anonymize delivery address data: address becomes "Address Deleted", coordinates become null
- **FR-034**: System MUST retain order transaction data (order reference, items, amounts, status, timestamps) for merchant records
- **FR-035**: System MUST record data deletion in audit trail with order reference, timestamp, and deletion method

#### Audit Trail Logging

- **FR-036**: System MUST create comprehensive audit log entries containing: timestamp, tenant identifier, user identifier, actor type (user/system/guest), action, resource type, resource identifier, IP address, session identifier, before/after values (encrypted), and metadata
- **FR-037**: System MUST log all PII access operations: read, create, update, delete with full context
- **FR-038**: System MUST log authentication events: login success, login failure, logout, token refresh, session expiry
- **FR-039**: System MUST log data export requests and results
- **FR-040**: System MUST log consent changes: grant, revoke, update with consent type and version
- **FR-041**: System MUST log encryption key operations: key rotation, key access (without logging the key itself) **[POST-MVP - deferred to separate key management feature]**
- **FR-042**: System MUST implement audit logs as append-only (no updates or deletes permitted)
- **FR-043**: System MUST store audit logs with restricted access permissions separate from operational data
- **FR-044**: System MUST retain audit logs for minimum 7 years per Indonesian record retention requirements
- **FR-045**: System MUST provide audit log query interface for platform administrators with filtering by tenant, user, date range, action type

#### Consent Collection and Management

- **FR-046**: System MUST create consent records containing: tenant identifier, user identifier, order reference (for guests), consent type, purpose, granted status, grant timestamp, revocation timestamp, IP address, user agent, consent version
- **FR-047**: System MUST display consent checkboxes on tenant registration form for: (a) operational_data [required], (b) analytics [required], (c) advertising [optional], (d) third_party_midtrans [required]
- **FR-048**: System MUST prevent tenant registration submission until all required consents are checked
- **FR-049**: System MUST display consent checkboxes on guest checkout form for: (a) order_processing [required], (b) order_communications [required], (c) promotional_communications [optional], (d) payment_processing_midtrans [required]
- **FR-050**: System MUST prevent guest order submission until all required consents are checked
- **FR-051**: System MUST record consent grant with timestamp, IP address, user agent, and consent version number
- **FR-052**: System MUST link consent records to tenant or guest order
- **FR-053**: System MUST provide consent management interface in tenant account settings showing current consent status
- **FR-054**: System MUST allow users to revoke optional consents (analytics, advertising, promotional) at any time
- **FR-055**: System MUST prevent revocation of mandatory consents (operational, order processing) while account/order is active
- **FR-056**: System MUST record consent revocation with timestamp and update audit trail
- **FR-057**: System MUST stop processing data for revoked consent purposes immediately upon revocation

#### Privacy Policy

- **FR-058**: System MUST provide a publicly accessible Privacy Policy page at `/privacy-policy` route
- **FR-059**: Privacy Policy MUST disclose all categories of PII collected with specific examples
- **FR-060**: Privacy Policy MUST explain purposes for data processing: Tenant Operations, Service Analysis, Advertising & Promotion, Third-Party Integration (Midtrans)
- **FR-061**: Privacy Policy MUST state legal basis for processing (consent per UU PDP Article 20)
- **FR-062**: Privacy Policy MUST disclose data retention periods: active accounts (indefinite), closed accounts (90 days grace + hard delete), guest orders (5 years for tax compliance)
- **FR-063**: Privacy Policy MUST list all third-party data processors: Midtrans Payment Gateway with link to their privacy policy
- **FR-064**: Privacy Policy MUST describe security measures: encryption at rest, access controls, audit logging
- **FR-065**: Privacy Policy MUST explain data subject rights: access, correction, deletion, consent revocation, complaint process
- **FR-066**: Privacy Policy MUST provide contact information for data protection inquiries: email address and expected response time (14 days)
- **FR-067**: Privacy Policy MUST be available in Indonesian language (Bahasa Indonesia)
- **FR-068**: Privacy Policy MUST include version number and last updated date
- **FR-069**: System MUST show link to Privacy Policy on registration, checkout, and footer of all pages

#### Data Retention and Cleanup

- **FR-070**: System MUST implement automated cleanup job that runs daily to remove expired temporary data
- **FR-071**: System MUST delete expired verification tokens (older than 48 hours)
- **FR-072**: System MUST delete used password reset tokens (consumed and older than 24 hours)
- **FR-073**: System MUST delete expired invitations (expired status and older than 30 days)
- **FR-074**: System MUST delete expired sessions (expired and older than 7 days)
- **FR-075**: System MUST implement soft delete grace period of 90 days for deleted tenant accounts
- **FR-076**: System MUST send notification to tenant 30 days before hard delete occurs
- **FR-077**: System MUST retain completed guest order data for 5 years from completion date (Indonesian tax law requirement)
- **FR-078**: System MUST hard delete guest order data automatically after 5 year retention period expires
- **FR-079**: System MUST log all automated cleanup actions in audit trail

### Key Entities _(include if feature involves data)_

- **Consent Record**: Tracks user consent for data processing purposes, links to tenant or guest order, records grant/revoke timestamps, IP address, consent version. Supports compliance with UU PDP consent requirements.

- **Audit Log Entry**: Immutable record of all data access and modifications, contains actor info, action type, resource affected, before/after values (protected), IP address, timestamp. Supports compliance investigations and security incident response.

- **Encrypted Data Field**: Sensitive information protected at rest through encryption, stored with integrity verification, decrypted transparently by application layer. Protects PII from unauthorized access.

- **Privacy Policy Version**: Versioned privacy policy document with effective date, content in Indonesian, tracks acceptance by users. Supports transparency obligations under UU PDP.

- **Data Deletion Request**: Guest customer request to anonymize personal data while retaining order history for merchant, records request timestamp, processing status, completion timestamp. Supports right to erasure under UU PDP.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All PII fields are encrypted at rest and cannot be read in plaintext when inspecting data storage directly (100% of identified sensitive fields)
- **SC-002**: No sensitive data appears in plaintext in application logs (0 instances of unmasked PII in log files)
- **SC-003**: All tenant registrations and guest checkouts include documented consent with timestamp (100% consent capture rate)
- **SC-004**: Privacy Policy is accessible and loads in under 2 seconds (95th percentile response time)
- **SC-005**: Tenant owners can access and update their data within 3 clicks from dashboard (maximum 3-click depth for data management)
- **SC-006**: Guest customers can request data deletion and receive confirmation within 24 hours (automated processing)
- **SC-007**: Audit trail captures 100% of sensitive data access events with complete context (no gaps in audit logging)
- **SC-008**: System passes external security audit for encryption implementation (verified by third-party security auditor)
- **SC-009**: Data retention policies are enforced automatically with 99% accuracy (less than 1% of expired data remains beyond grace period)
- **SC-010**: Platform can generate compliance report for regulators showing all data processing activities within 1 business day

## Assumptions

- Encryption keys will be managed securely with appropriate access controls and key rotation capabilities
- Data storage system supports encryption operations without significant performance degradation
- Log masking will be centralized to ensure consistent application across all system components
- Standard data retention period for tax compliance in Indonesia is 5 years from transaction date
- Privacy policy content will be reviewed by legal counsel before publication (legal review is out of scope for implementation)
- Consent version 1.0 will be initial version, with versioning strategy to handle future policy changes
- Audit logs will be stored separately from operational data with appropriate retention and archival mechanisms
- Guest order verification uses combination of order reference + (customer email OR customer phone) as authentication method
- Midtrans is the only third-party data processor currently integrated; additional processors require privacy policy updates
- System administrators will have read-only access to audit logs through secure admin interface
- Backup and disaster recovery procedures will maintain protection of sensitive data in backups
- Indonesian language (Bahasa Indonesia) privacy policy is primary, English translation is optional for this phase
- No existing PII requires retroactive encryption migration (or migration will be handled in separate data migration task)
- Performance impact of encryption/decryption operations is acceptable for business requirements

## Out of Scope

- Integration with external data subject request management platforms
- Automated privacy impact assessments (PIA) for new features
- Cookie consent management for frontend tracking (separate feature)
- Data portability export in machine-readable format (GDPR-style data export)
- Cross-border data transfer agreements and mechanisms
- Privacy-by-design architectural review of existing features
- Training materials for tenant users on data protection practices
- Incident response procedures for data breaches (separate security policy)
- Data anonymization for analytics/reporting purposes beyond deletion requests
- Integration with Indonesian regulatory reporting systems (if any)
- Biometric data protection (not currently collected)
- Children's data protection (platform is for business use only)
- Automated consent renewal/re-confirmation workflows
- Privacy preference center with granular consent controls beyond required/optional
- Third-party vendor security assessments and data processing agreements

## Notes

- **Implementation Priority**: Encryption at rest (FR-001 to FR-012) and log masking (FR-013 to FR-020) must be implemented first as foundation. Consent collection (FR-046 to FR-057) and privacy policy (FR-058 to FR-069) must be completed before any new user registrations. Audit trail (FR-036 to FR-045) should be implemented in parallel. Data retention cleanup (FR-070 to FR-079) can be implemented last.

- **Legal Compliance**: This specification addresses technical implementation requirements for UU PDP No.27 Tahun 2022 compliance. Legal review of privacy policy content, consent language, and retention periods should be conducted by qualified Indonesian legal counsel before production deployment.

- **Performance Considerations**: Encryption/decryption operations will add computational overhead. Benchmark testing should be conducted to ensure performance remains within acceptable business limits. Consider caching strategies for frequently accessed sensitive data.

- **Guest Data Access**: Guest customers do not have accounts, so data access relies on order reference + email/phone verification. This creates a security consideration: anyone with these details can access order data. Consider adding additional verification step (email/SMS OTP) for sensitive operations like data deletion.

- **Consent Versioning**: When privacy policy or consent language changes materially, consent version should be incremented and users may need to re-consent. Implement mechanism to track which users have consented to which version and flag accounts needing re-consent.

- **Testing Strategy**: Encryption correctness and log masking must be tested thoroughly. Include test cases for: encryption/decryption round-trip integrity, handling of null values, concurrent data access, key rotation scenarios, backup/restore with encryption, and log output validation for all sensitive data types.
