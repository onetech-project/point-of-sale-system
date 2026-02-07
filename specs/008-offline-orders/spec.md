# Feature Specification: Offline Order Management

**Feature Branch**: `008-offline-orders`  
**Created**: February 7, 2026  
**Status**: Draft  
**Input**: User description: "as a tenant's owner i want to manually record order in order management (offline order) with these condition: follow checkout form for customer data and item order, provide down payment / installments scheme / payment term for offline order, offline order can be completed only if all payment has been settled, can edit the offline order anytime (with recording to audit trail), all role can add and edit offline order (audit trailed - CREATE, UPDATE, READ, ACCESS), only owner and manager can delete the offline order (audit trailed - DELETE), make sure don't ruin online order flow, keep comply with implemtation of UU PDP and GDPR, make sure the data can be include on dashboard (analytics service)"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Record Basic Offline Order (Priority: P1)

A store staff member records a walk-in customer's purchase that happened in person, capturing customer details and items purchased.

**Why this priority**: This is the core functionality - the minimum viable capability to record offline transactions that occur outside the digital system. Without this, there's no offline order feature at all.

**Independent Test**: Can be fully tested by having a user with any role create an offline order with customer information and line items, then verify the order appears in the order management system with "offline" designation and all data is properly audit-trailed.

**Acceptance Scenarios**:

1. **Given** a staff member is logged in, **When** they navigate to order management and select "Record Offline Order", **Then** they see a form similar to the checkout form with fields for customer data and order items
2. **Given** the offline order form is displayed, **When** staff enters customer name, contact information, and adds at least one product with quantity, **Then** the system validates all required fields and calculates the total
3. **Given** all order details are entered with full payment, **When** staff submits the order, **Then** the order is created with status "completed", assigned a unique order ID, and logged to the audit trail with CREATE action
4. **Given** an offline order is created, **When** viewing the order list, **Then** offline orders are clearly distinguished from online orders (e.g., labeled or tagged)

---

### User Story 2 - Manage Payment Terms and Installments (Priority: P2)

Store owner records an offline order where the customer pays a down payment with remaining balance to be paid in installments.

**Why this priority**: Enables businesses to offer flexible payment options to walk-in customers, which is common in B2B or high-value transactions. Builds on P1 by adding payment flexibility without which the feature would be too rigid for many real-world use cases.

**Independent Test**: Can be fully tested by creating an offline order with a down payment of 30% and two installment payments, verifying the order status remains "pending" until all payments are recorded, then marking individual payments as received until the order status changes to "completed".

**Acceptance Scenarios**:

1. **Given** staff is recording an offline order, **When** they select "Partial Payment" option, **Then** they can specify down payment amount and define installment schedule (number of payments, amounts, due dates)
2. **Given** an offline order with payment terms is created, **When** only the down payment is recorded, **Then** the order status is "pending payment" and shows outstanding balance
3. **Given** an offline order has pending installments, **When** staff records each subsequent payment, **Then** the system updates the outstanding balance and logs each payment to audit trail
4. **Given** an offline order with installments, **When** the final payment is recorded and total paid equals order total, **Then** the order status automatically changes to "completed"
5. **Given** an offline order with pending payments, **When** viewing the order details, **Then** the payment history shows all payments received with dates, amounts, and remaining balance

---

### User Story 3 - Edit Offline Orders with Audit Trail (Priority: P3)

Staff member needs to correct customer information or modify order items after an offline order has been recorded.

**Why this priority**: Handles error correction and changes, which is essential for data accuracy but not critical for the initial order recording functionality. This adds operational flexibility.

**Independent Test**: Can be fully tested by creating an offline order, then editing customer phone number and adding one more item, verifying that both changes are reflected in the order details and each modification is logged to the audit trail with UPDATE action including timestamp, user, and changed fields.

**Acceptance Scenarios**:

1. **Given** an existing offline order, **When** any role user accesses the order, **Then** they see an "Edit Order" option
2. **Given** user clicks "Edit Order", **When** they modify customer data, line items, quantities, or payment information, **Then** the system allows changes and validates data
3. **Given** changes are saved, **When** the update is submitted, **Then** the system records the changes and logs to audit trail with UPDATE action, including user ID, timestamp, and specific fields modified
4. **Given** an offline order has been edited multiple times, **When** viewing the audit trail, **Then** all modifications are visible with complete history of what changed, who changed it, and when

---

### User Story 4 - Role-Based Order Deletion (Priority: P3)

A manager or owner needs to remove an erroneous or fraudulent offline order from the system.

**Why this priority**: Provides administrative control for data quality and fraud prevention. Lower priority because deletion should be rare compared to creation and editing.

**Independent Test**: Can be fully tested by verifying that staff and cashier roles cannot delete offline orders (delete button hidden or action blocked), while manager and owner roles can delete orders with DELETE action logged to audit trail including reason for deletion.

**Acceptance Scenarios**:

1. **Given** a user with staff or cashier role views an offline order, **When** they access order actions, **Then** no delete option is available
2. **Given** a user with owner or manager role views an offline order, **When** they access order actions, **Then** a "Delete Order" option is available
3. **Given** owner/manager selects delete, **When** they confirm the deletion with optional reason, **Then** the order is marked as deleted (soft delete recommended for audit) and logged to audit trail with DELETE action
4. **Given** an offline order is deleted, **When** viewing the audit trail, **Then** the deletion event shows who deleted it, when, and any reason provided

---

### User Story 5 - View Offline Orders in Analytics (Priority: P3)

Business owner reviews sales performance including both online and offline orders in the analytics dashboard.

**Why this priority**: Enables data-driven decision making by providing visibility into all revenue streams. Important for business intelligence but not critical for the operational order recording functionality.

**Independent Test**: Can be fully tested by creating 5 online orders and 3 offline orders with various amounts, then verifying the analytics dashboard shows accurate totals for both order types, revenue breakdowns, and offline orders are included in all relevant reports (daily sales, product performance, customer analytics).

**Acceptance Scenarios**:

1. **Given** offline orders exist in the system, **When** viewing the analytics dashboard, **Then** offline orders are included in total revenue, order count, and average order value metrics
2. **Given** the dashboard displays order trends, **When** filtering or segmenting data, **Then** users can distinguish between online and offline orders
3. **Given** analytics data is exported, **When** generating reports, **Then** offline order data is included with proper classification
4. **Given** offline orders have different payment statuses, **When** viewing revenue reports, **Then** only completed offline orders (all payments settled) contribute to revenue totals

---

### Edge Cases

- What happens when an offline order is created but the customer never completes payment installments? (Order remains in "pending payment" status indefinitely, may need follow-up process or automatic reminder)
- How does the system handle editing an offline order that has partial payments recorded? (Should allow editing order items/amounts but recalculate outstanding balance; payments already recorded remain in history)
- What if a user tries to delete an offline order that has payments recorded? (System should either prevent deletion and require specific permission or soft-delete to preserve financial audit trail)
- How does the system prevent duplicate recording of the same offline transaction? (Staff responsibility to check existing orders; system could provide recent order search by customer name/phone)
- What happens if customer data is incomplete or customer wishes to remain anonymous? (Allow optional customer fields for privacy compliance; minimum required might be phone OR email OR name depending on business rules)
- How does the system handle offline orders for customers who also have online accounts? (Should link to existing customer record if identifiable, maintaining unified customer history)
- What if an item in an offline order is later refunded or returned? (Should support return/refund process similar to online orders, with proper audit trail)
- How does the system handle concurrent edits to the same offline order by multiple users? (Implement optimistic locking or last-write-wins with clear audit trail of changes)

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST provide a form to manually record offline orders that mirrors the checkout form structure, including customer data fields (name, contact information, address) and order items with quantities and prices
- **FR-002**: System MUST support multiple payment options for offline orders: full payment, down payment with balance, and installment payment schedules with configurable amounts and due dates
- **FR-003**: System MUST maintain an order status that reflects payment completion: "pending payment" when payments are outstanding, "completed" only when total paid equals order total
- **FR-004**: System MUST allow all authenticated users (regardless of role) to create and edit offline orders
- **FR-005**: System MUST restrict deletion of offline orders to users with "owner" or "manager" roles only
- **FR-006**: System MUST log all offline order operations (CREATE, READ, UPDATE, DELETE, ACCESS) to the audit trail with user ID, timestamp, action type, and details of changes made
- **FR-007**: System MUST clearly distinguish offline orders from online orders in the order management interface through visual indicators, filters, or order type labels
- **FR-008**: System MUST ensure offline order functionality does not interfere with existing online order processing workflows, checkout forms, or payment integrations
- **FR-009**: System MUST encrypt personally identifiable information (PII) in offline orders following the same data protection standards as online orders to comply with UU PDP and GDPR
- **FR-010**: System MUST allow data subjects to exercise their privacy rights (access, rectification, erasure, data portability) for data collected in offline orders, consistent with UU PDP and GDPR requirements
- **FR-011**: System MUST include offline order data in the analytics service, making it available for dashboard metrics, reports, and business intelligence queries
- **FR-012**: System MUST calculate and display outstanding balance for offline orders with partial payments, showing payment history and remaining amounts due
- **FR-013**: System MUST validate offline order data including required fields, valid product references, positive quantities, and accurate price calculations before allowing order creation
- **FR-014**: System MUST support recording payment receipts for installments, associating each payment with the offline order and updating payment status accordingly
- **FR-015**: System MUST preserve referential integrity between offline orders and related entities (products, customers, tenants) ensuring no orphaned records
- **FR-016**: System MUST maintain a complete audit history for each offline order that is accessible to authorized users and cannot be modified or deleted

### Key Entities _(include if feature involves data)_

- **Offline Order**: Represents a manual order recorded by staff for transactions occurring outside the digital system. Contains customer information, order items, payment terms, status (pending payment/completed), creation timestamp, last modified timestamp, and tenant association. Linked to audit trail entries.
- **Order Line Item**: Represents individual products within an offline order. Contains product reference, quantity, unit price at time of order, subtotal. Multiple line items belong to one offline order.
- **Payment Record**: Represents a payment transaction associated with an offline order. Contains amount paid, payment date, payment method, remaining balance after payment, and reference to the offline order. Multiple payment records can exist for one order supporting installments.
- **Payment Terms**: Defines the payment schedule for an offline order. Contains total order amount, down payment amount (if applicable), number of installments, installment amounts, due dates. Associated with one offline order.
- **Audit Trail Entry**: Records all operations performed on offline orders. Contains action type (CREATE/READ/UPDATE/DELETE/ACCESS), user ID who performed action, timestamp, order ID, specific fields modified (for UPDATE), and previous/new values for changed fields.
- **Customer Data**: Personal information captured for offline orders. Contains name, email, phone, address. Must be encrypted per data protection requirements. May link to existing customer account if available.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Staff can record a complete offline order from start to finish (customer data + items + payment) in under 3 minutes for a typical 5-item order
- **SC-002**: System maintains 100% audit trail coverage for all offline order operations with no gaps in logging
- **SC-003**: Offline order data appears in analytics dashboards within 5 seconds of order creation/completion
- **SC-004**: Online order processing performance and availability remains unchanged (no degradation > 5%) after offline order feature deployment
- **SC-005**: All offline order PII is encrypted at rest and in transit, passing compliance audit with zero violations
- **SC-006**: Users with restricted roles (non-owner/manager) cannot delete offline orders, verified through 100% access control test success rate
- **SC-007**: 95% of offline orders with payment terms have all installments successfully tracked and the order status updates correctly upon final payment
- **SC-008**: Payment reconciliation accuracy for offline orders matches accounting records with 99%+ accuracy
