# Feature Specification: Order Email Notifications

**Feature Branch**: `004-order-email-notifications`  
**Created**: 2025-12-11  
**Status**: Draft  
**Input**: User description: "As tenant owner/manager/cashier I want to receive notification via email on every paid order to prevent miss or delay order process. As customer (guest) I want to receive receipt via email (use invoice design but add paid watermark)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Tenant Staff Receives Order Notification Email (Priority: P1)

When a customer completes payment and an order transitions to PAID status, tenant staff members (owner, manager, cashier) immediately receive an email notification containing essential order details to begin fulfillment without delay.

**Why this priority**: This is the critical real-time alert that prevents missed orders and delays in fulfillment. Without immediate notification, staff must manually check the dashboard, creating risk of delayed or forgotten orders. This directly addresses the core business need for timely order processing.

**Independent Test**: Can be fully tested by completing a paid order through the existing guest ordering system and verifying that all configured staff members receive an email within 1 minute containing order number, items, customer details, and delivery type. Delivers immediate value by enabling proactive order fulfillment.

**Acceptance Scenarios**:

1. **Given** a guest completes payment for an order, **When** the order status changes to PAID, **Then** system triggers email notification to all tenant staff members configured for order notifications
2. **Given** an order notification is triggered, **When** email is composed, **Then** it includes order reference number, order date/time, customer name, customer contact info, delivery type, and itemized list with quantities and prices
3. **Given** multiple staff members are configured for notifications, **When** order becomes PAID, **Then** each staff member receives the notification independently within 1 minute
4. **Given** a tenant has staff with different roles (owner, manager, cashier), **When** notification preferences are configured, **Then** tenant can select which roles receive order notifications
5. **Given** staff member has provided valid email address, **When** notification is sent, **Then** email is delivered successfully and system logs delivery status
6. **Given** email delivery fails, **When** system detects failure, **Then** system retries sending up to 3 times with exponential backoff
7. **Given** a notification email is received, **When** staff member opens it, **Then** email displays properly on both mobile and desktop email clients with clear formatting and readable content
8. **Given** multiple orders are paid within short time, **When** notifications are sent, **Then** each order generates a separate email to avoid confusion

---

### User Story 2 - Guest Receives Email Receipt (Priority: P1)

After successfully paying for an order, the guest customer receives an email receipt that serves as proof of payment and order confirmation, using the existing invoice design with a visible "PAID" watermark.

**Why this priority**: This provides immediate payment confirmation to customers, reducing anxiety about payment success and providing a reference for order tracking. It's a standard e-commerce practice that builds trust and reduces customer support inquiries about payment status.

**Independent Test**: Can be fully tested by completing a guest order with payment and verifying the customer receives an email receipt with invoice format, clear PAID marking, order details, and payment information. Delivers value by providing customers with immediate transaction confirmation.

**Acceptance Scenarios**:

1. **Given** a guest completes payment for an order, **When** payment is confirmed as successful, **Then** system sends email receipt to the email address provided during checkout
2. **Given** receipt email is being composed, **When** content is generated, **Then** it uses the existing invoice design/template with all order details formatted consistently
3. **Given** invoice design is applied, **When** receipt is rendered, **Then** a prominent "PAID" watermark is visible on the invoice to distinguish it from unpaid invoices
4. **Given** receipt is being sent, **When** email content is prepared, **Then** it includes order reference number, payment date/time, payment method (QRIS via Midtrans), transaction ID, itemized order details with quantities and prices, subtotal, delivery fee (if applicable), and total amount paid
5. **Given** guest provided email during checkout, **When** receipt is sent, **Then** email is delivered within 2 minutes of payment confirmation
6. **Given** receipt email is received, **When** customer opens it, **Then** email displays properly on mobile and desktop email clients with invoice formatting intact
7. **Given** email delivery fails, **When** system detects failure, **Then** system retries sending up to 3 times and logs failure for manual follow-up
8. **Given** customer wants proof of payment, **When** they open the email receipt, **Then** the PAID watermark and payment details provide clear confirmation of successful payment

---

### User Story 3 - Configure Email Notification Preferences (Priority: P2)

Tenant administrators configure which staff members receive order notification emails and customize notification settings based on their operational needs.

**Why this priority**: Different tenants have different organizational structures and notification needs. Some may want only managers notified, others may want all staff. This flexibility ensures the notification system adapts to various business workflows without creating email overload.

**Independent Test**: Can be fully tested by logging into tenant admin settings, adding/removing staff members from notification list, configuring notification preferences, and verifying changes take effect on next paid order.

**Acceptance Scenarios**:

1. **Given** a tenant administrator is logged into admin dashboard, **When** they navigate to notification settings, **Then** they see a list of all staff members with toggle switches to enable/disable order notifications for each
2. **Given** notification settings are displayed, **When** administrator views staff list, **Then** they see staff member name, email address, role, and current notification status with clear visual indicators (enabled/disabled state)
3. **Given** administrator selects staff members for notifications, **When** they save settings, **Then** system validates all selected staff have valid email addresses and displays inline error messages for any invalid entries
4. **Given** notification preferences are saved, **When** next order becomes PAID, **Then** only selected staff members receive notification emails
5. **Given** administrator adds a new staff member, **When** they save the staff profile, **Then** administrator can immediately enable order notifications for that staff member from the notification settings page
6. **Given** a staff member's email address is updated, **When** change is saved, **Then** future notifications are sent to the new email address
7. **Given** administrator wants to test notifications, **When** they click "Send Test Email" button and confirm, **Then** system sends a sample order notification email with placeholder order data (order reference "TEST-001", sample items, sample customer info) to selected staff members and displays success/failure feedback message
8. **Given** test email is sent, **When** recipient opens email, **Then** email clearly indicates it is a test notification with "[TEST]" prefix in subject line

---

### User Story 4 - View Email Notification History (Priority: P3)

Tenant administrators view a log of all email notifications sent for orders, including delivery status, to troubleshoot delivery issues and confirm notifications were sent.

**Why this priority**: This provides transparency and accountability for email notifications. While less critical than sending the notifications themselves, having a history helps diagnose issues when staff claims they didn't receive a notification or when investigating missed orders.

**Independent Test**: Can be fully tested by navigating to notification history in admin dashboard, viewing list of sent notifications with timestamps, recipients, and delivery status, and filtering/searching the history.

**Acceptance Scenarios**:

1. **Given** a tenant administrator is logged into admin dashboard, **When** they navigate to notification history, **Then** they see a list of all order notifications sent with order reference number, date/time sent, recipient email addresses, and delivery status
2. **Given** notification history is displayed, **When** administrator reviews entries, **Then** they can see which emails were delivered successfully, which failed, and which are pending retry
3. **Given** administrator wants to investigate a specific order, **When** they search by order reference number, **Then** system displays all notification emails sent for that order
4. **Given** administrator suspects email delivery issues, **When** they filter by delivery status, **Then** system shows all failed or pending notifications
5. **Given** a notification failed to deliver, **When** administrator views the history entry, **Then** they see the failure reason and can manually resend the notification
6. **Given** administrator wants to verify receipt notifications, **When** they view notification history, **Then** they can see both staff notifications and customer receipts separately

---

### Edge Cases

- What happens when a guest doesn't provide an email address during checkout? → System does not require email for guest orders (current behavior), so receipt email is skipped. Staff notifications still sent.
- What happens when a staff member's email address is invalid or bounces? → System logs the bounce, marks the staff member's email as invalid in notification history, and administrator sees warning in notification settings.
- What happens when email service provider is temporarily unavailable? → System queues the email for retry, attempts up to 3 times with exponential backoff, and logs failure if all attempts fail. Administrator can manually resend from notification history.
- What happens when an order transitions from PENDING to PAID multiple times due to payment retries? → System sends notification only once per unique payment success event, tracked by transaction ID to prevent duplicate notifications.
- What happens when a tenant has no staff members configured for notifications? → System logs warning but does not fail order processing. Administrator sees notification in dashboard alerting them to configure notification recipients.
- What happens when the same email address is used by multiple staff members? → System sends one notification per configured recipient, even if email addresses are duplicated. This is intentional to respect configuration.
- What happens when email content becomes corrupted or fails to generate? → System logs error, falls back to plain text email format with essential order details, and alerts administrator of template rendering failure.
- What happens when a guest uses a disposable/temporary email address? → System sends receipt normally. If email bounces, failure is logged but does not impact order processing.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST trigger email notifications when an order transitions to PAID status, targeting delivery within 1 minute of payment confirmation (99% SLA)
- **FR-002**: System MUST send order notification emails to all tenant staff members who have order notifications enabled in their configuration (new staff members have notifications disabled by default)
- **FR-003**: System MUST send receipt email to the guest customer's email address provided during checkout (if email was provided), targeting delivery within 2 minutes (99% SLA)
- **FR-004**: Order notification email MUST include order reference number, order date/time, customer name, customer phone number, customer email (if provided), delivery type, delivery address (if applicable), itemized list of products with quantities and individual prices, subtotal, delivery fee, and total amount paid
- **FR-005**: Receipt email MUST use the existing invoice template/design with all order details formatted consistently with invoices generated in the system
- **FR-006**: Receipt email MUST include a prominent "PAID" watermark overlaid on the invoice to clearly distinguish it from unpaid invoices
- **FR-007**: Receipt email MUST include payment method (QRIS via Midtrans), payment date/time, and transaction ID from payment provider
- **FR-008**: System MUST validate email addresses before sending and reject invalid formats
- **FR-009**: System MUST retry failed email deliveries up to 3 times with exponential backoff timing (1st retry after 1 minute, 2nd retry after 5 minutes, 3rd retry after 15 minutes)
- **FR-010**: System MUST log all email notification attempts including timestamp, recipient, delivery status, and failure reason (if applicable)
- **FR-011**: Tenant administrators MUST be able to configure which staff members receive order notification emails through admin dashboard settings
- **FR-012**: Tenant administrators MUST be able to view notification history including all sent emails, recipients, delivery status, and timestamps
- **FR-013**: System MUST support resending failed notifications manually from the notification history interface (manual resend respects max_retries limit of 3 total attempts)
- **FR-014**: Email notifications MUST be mobile-responsive and render correctly in common email clients (Gmail web/mobile, Outlook 2016+/365/web, Apple Mail macOS/iOS, default Android/iOS email apps) on viewports 320px-1920px wide, with all order details visible, formatted tables intact, and clickable links functional
- **FR-015**: System MUST prevent duplicate notifications for the same payment event by tracking transaction IDs
- **FR-016**: System MUST continue order processing successfully even if email notifications fail, ensuring email issues do not block order fulfillment
- **FR-017**: System MUST provide a test notification feature allowing administrators to send sample emails to verify configuration
- **FR-018**: System MUST support configuring notification preferences per staff member including ability to enable/disable notifications independently

### Key Entities

- **Order Notification**: Represents an email notification sent to tenant staff about a paid order; includes order reference, recipient list, send timestamp, delivery status, retry count. *Implementation: Stored in `notifications` table with `event_type='order.paid.staff'`*
- **Receipt Email**: Represents a receipt email sent to a guest customer; includes order reference, customer email, invoice content with PAID watermark, send timestamp, delivery status. *Implementation: Stored in `notifications` table with `event_type='order.paid.customer'`*
- **Notification Configuration**: Represents tenant-level settings for email notifications; includes list of staff members enabled for notifications, notification preferences, email template customizations. *Implementation: Stored in `notification_configs` table (new) and `users.receive_order_notifications` field (extended)*
- **Notification History Entry**: Represents a record of a sent notification; includes notification type (staff or customer), order reference, recipient email, send timestamp, delivery status, failure reason (if failed), retry attempts. *Implementation: Query from `notifications` table filtered by `event_type LIKE 'order.paid.%'`*

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Tenant staff receive order notification emails within 1 minute of payment confirmation for 99% of orders
- **SC-002**: Guest customers receive receipt emails within 2 minutes of payment confirmation for 99% of orders where email was provided
- **SC-003**: Email notification delivery success rate exceeds 98% (including retries)
- **SC-004**: Staff can configure notification preferences and see changes take effect on next order without system restart
- **SC-005**: Notification emails display properly (all content visible and formatted) in at least 95% of common email clients tested (Gmail, Outlook, Apple Mail, mobile clients)
- **SC-006**: Zero orders fail to complete due to email notification errors (email failures do not block order processing)
- **SC-007**: Administrators can access complete notification history showing all sent emails for any order within the past 90 days
- **SC-008**: Duplicate notifications for the same payment event occur in less than 0.1% of cases
- **SC-009**: Failed email notifications are automatically retried within the configured backoff schedule with 95% success rate on retry
- **SC-010**: Manual resend of failed notifications from history succeeds in 99% of cases when email address is valid

## Assumptions

- The system already has an existing invoice template/design that can be reused for receipt emails
- The notification-service infrastructure exists and can be extended to handle order notifications
- Staff members have email addresses stored in the user-service database
- The order-service can emit events or call notification endpoints when order status changes to PAID
- An email service provider integration exists or will be implemented (e.g., SMTP, SendGrid, AWS SES)
- Guest checkout flow already captures optional email address from customers
- The system uses Midtrans for payment processing and receives transaction IDs that can be used to prevent duplicates
- Tenant administrators have appropriate permissions to configure notification settings
- The admin dashboard has a settings section where notification configuration can be added
- Notification history can be stored for at least 90 days for troubleshooting and audit purposes
