# Feature Specification: QRIS Guest Ordering System

**Feature Branch**: `003-guest-qris-ordering`  
**Created**: 2025-12-03  
**Status**: Draft  
**Input**: User description: "Customers open a tenant-specific public menu URL, browse/select items, and place an order. At checkout they choose a tenant-allowed delivery_type (pickup | delivery | dine_in), provide contact and address if delivery, and proceed to QRIS payment (Midtrans). Midtrans will notify our /payments/midtrans/notification endpoint when payment completes. After PAID, tenant staff handle courier ordering outside the system and then update order status in the admin dashboard to COMPLETE when delivery finished."

## Clarifications

### Session 2025-12-03

- Q: Should delivery fee calculation be optional per tenant configuration, or mandatory for all delivery orders? → A: Optional - Tenant can enable/disable automatic delivery fee calculation. If disabled, delivery_fee = 0 in checkout, tenant collects manually
- Q: When inventory reservation TTL expires (default 15 minutes), should the system allow the guest to immediately try checking out again with the same cart, or should there be a cooldown period? → A: Immediate retry allowed - Guest can checkout again immediately, system creates new reservation if inventory still available
- Q: When a guest's payment fails or is cancelled, what happens to their existing order reference number and cart? → A: Keep order reference and cart - Guest can retry payment using same order reference number and cart contents
- Q: When a guest tries to add 5 items to cart but only 3 are available, how should the system respond? → A: Add maximum available - Automatically add 3 items with notification "Only 3 available, added to cart"
- Q: When a guest's browser session expires while they're in the middle of checkout (e.g., filling out address form), what should happen to their cart and progress? → A: Restore cart from session storage - Cart persists client-side, guest can continue (checkout form needs re-entry)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse and Add Items to Cart (Priority: P1)

A guest customer visits a tenant's public menu URL on their mobile device, browses available products without logging in, and adds items to their cart.

**Why this priority**: This is the foundation of the guest ordering flow. Without the ability to browse and select items, no transaction can occur. This represents the core value proposition of making ordering accessible without authentication barriers.

**Independent Test**: Can be fully tested by accessing a tenant-specific public menu URL, viewing product listings with prices and descriptions, and successfully adding/removing items from a guest cart. Delivers immediate value by allowing tenants to showcase their offerings to potential customers.

**Acceptance Scenarios**:

1. **Given** a guest visits a tenant's public menu URL, **When** they load the page, **Then** they see a mobile-optimized product catalog without any login prompt
2. **Given** a guest is viewing the product catalog, **When** they select a product, **Then** they see product details including name, description, price, and availability status
3. **Given** a guest is viewing a product, **When** they add it to cart with specified quantity, **Then** the item is added to their guest cart and cart total updates
4. **Given** a guest requests quantity exceeding available inventory, **When** they attempt to add to cart, **Then** system automatically adds maximum available quantity with notification message (e.g., "Only 3 available, added to cart")
5. **Given** a guest has items in their cart, **When** they modify quantities or remove items, **Then** cart updates immediately and shows updated total
4. **Given** a guest has items in their cart, **When** they navigate away and return to the same tenant URL within a session, **Then** their cart contents are preserved
5. **Given** a product has limited inventory during cart modification, **When** guest increases quantity beyond available, **Then** system adjusts to maximum available with notification

---

### User Story 2 - Select Delivery Type and Provide Details (Priority: P1)

A guest proceeds to checkout and selects from tenant-allowed delivery types (pickup, delivery, dine-in), providing necessary contact and address information based on their selection.

**Why this priority**: This is essential for order fulfillment. Different delivery types require different information and have different operational flows. Without this step, the tenant cannot process the order appropriately.

**Independent Test**: Can be fully tested by proceeding to checkout, seeing available delivery options based on tenant configuration, selecting each type, and verifying appropriate form fields appear. For delivery type, verify address capture and validation occurs.

**Acceptance Scenarios**:

1. **Given** a guest proceeds to checkout, **When** the checkout page loads, **Then** they see only the delivery types enabled by the tenant (pickup, delivery, and/or dine-in)
2. **Given** a guest selects "pickup", **When** the form updates, **Then** they are prompted for name and phone number only
3. **Given** a guest selects "dine-in", **When** the form updates, **Then** they are prompted for name, phone number, and optional table number or seating preference
4. **Given** a guest selects "delivery", **When** the form updates, **Then** they are prompted for name, phone number, and complete delivery address
5. **Given** a guest enters a delivery address, **When** they proceed, **Then** system geocodes the address and validates it falls within tenant's service area
6. **Given** a guest's delivery address is outside service area, **When** validation runs, **Then** guest sees error message indicating delivery is not available to that location
7. **Given** a guest's delivery address is valid AND tenant has automatic delivery fees enabled, **When** validation completes, **Then** system calculates delivery fee based on distance or zone and displays it in order total
8. **Given** tenant has disabled automatic delivery fees, **When** guest completes checkout, **Then** delivery_fee is set to 0 and tenant collects delivery fee manually outside the system
8. **Given** a guest enters invalid contact information, **When** they attempt to proceed, **Then** system shows validation errors for specific fields

---

### User Story 3 - Complete Midtrans QRIS Payment (Priority: P1)

A guest completes their order by paying through Midtrans QRIS payment gateway, with the system handling payment notifications asynchronously.

**Why this priority**: This is the critical transaction completion flow. Without payment processing, the ordering system cannot generate revenue. Midtrans integration ensures secure payment handling and proper order status updates.

**Independent Test**: Can be fully tested by proceeding from checkout to payment, being redirected to Midtrans QRIS page, completing payment, and verifying system receives webhook notification and updates order status correctly.

**Acceptance Scenarios**:

1. **Given** a guest has completed checkout form, **When** they confirm the order, **Then** system creates a pending order with temporary inventory reservation and initiates Midtrans payment request
2. **Given** Midtrans payment is initiated, **When** guest is redirected, **Then** they see Midtrans QRIS payment page with QR code and order amount
3. **Given** a guest scans the QR code and completes payment, **When** payment succeeds, **Then** Midtrans sends notification to /payments/midtrans/notification endpoint
4. **Given** system receives Midtrans notification, **When** webhook is processed, **Then** system verifies signature, validates payment amount matches order, and updates order status to PAID
5. **Given** order status is updated to PAID, **When** inventory is processed, **Then** temporary reservation is converted to permanent inventory reduction
6. **Given** payment is successful, **When** guest is redirected back, **Then** they see order confirmation page with order reference number and delivery type details
7. **Given** payment fails or is cancelled, **When** Midtrans sends notification or guest returns, **Then** order status remains PENDING, inventory reservation is maintained, and guest sees error message with option to retry payment using same order reference and cart contents

---

### User Story 4 - Tenant Staff Manages Order Fulfillment (Priority: P1)

Tenant staff view paid orders in admin dashboard, arrange courier delivery outside the system, and update order status to COMPLETE when delivery is finished.

**Why this priority**: This closes the order lifecycle loop. Without manual status updates by staff, orders would remain in PAID status indefinitely and tenants would have no way to track completion in the system.

**Independent Test**: Can be fully tested by logging into admin dashboard as tenant staff, viewing list of PAID orders, selecting an order, and updating its status to COMPLETE with optional completion notes.

**Acceptance Scenarios**:

1. **Given** a tenant staff member is logged into admin dashboard, **When** they navigate to orders section, **Then** they see a list of orders filtered by status (PENDING, PAID, COMPLETE, CANCELLED)
2. **Given** staff is viewing order list, **When** they select a PAID order, **Then** they see full order details including items, delivery type, contact info, address (if delivery), and payment status
3. **Given** staff is viewing a PAID order with delivery type, **When** they arrange courier outside the system, **Then** they can add notes about courier or tracking information
4. **Given** staff has arranged delivery, **When** delivery is completed, **Then** they update order status to COMPLETE in the dashboard
5. **Given** staff updates order status to COMPLETE, **When** the update is saved, **Then** order appears in completed orders list and timestamp is recorded
6. **Given** an order is marked COMPLETE, **When** guest checks their order status via reference number, **Then** they see completed status with completion timestamp

---

### User Story 5 - Handle Inventory Availability with Reservations (Priority: P2)

The system manages product availability with time-limited inventory reservations to prevent overselling while allowing abandoned carts to free up inventory automatically.

**Why this priority**: Prevents customer disappointment and operational issues from overselling. While critical for business operations, it's prioritized after basic ordering flow to ensure MVP can process orders first.

**Independent Test**: Can be fully tested by attempting to order items with various stock levels, verifying that unavailable items cannot be added to cart, confirming temporary reservations during checkout, and validating reservations expire correctly.

**Acceptance Scenarios**:

1. **Given** a product has zero available inventory, **When** a guest views the product catalog, **Then** the product is marked as unavailable and cannot be added to cart
2. **Given** a guest has items in their cart, **When** inventory becomes unavailable before checkout, **Then** system notifies guest and prevents order submission until cart is adjusted
3. **Given** a guest initiates checkout, **When** order is created, **Then** system creates a temporary inventory reservation with configurable TTL (default: 15 minutes)
4. **Given** a temporary inventory reservation exists, **When** payment is not completed before TTL expires, **Then** reservation is released and inventory becomes available again; guest can immediately retry checkout with same cart if inventory remains available
5. **Given** payment is completed successfully, **When** order status changes to PAID, **Then** temporary reservation is converted to permanent inventory reduction
6. **Given** multiple guests attempt to order the last unit, **When** first guest creates reservation, **Then** subsequent guests see item as unavailable

---

### User Story 6 - Calculate Delivery Fees Based on Location (Priority: P2)

For delivery orders where tenant enables automatic fee calculation, the system geocodes addresses, validates serviceability, and calculates delivery fees based on tenant-configured rules. Tenants can also disable automatic calculation to handle delivery fees manually.

**Why this priority**: Important for tenants who want automated delivery pricing. However, pickup and dine-in orders can function without this, and some tenants prefer manual delivery fee collection, making it P2.

**Independent Test**: Can be fully tested by: (1) entering addresses when tenant has automatic fees enabled, verifying geocoding and fee calculation, and (2) verifying that when tenant disables automatic fees, delivery_fee remains 0 and tenant can collect manually.

**Acceptance Scenarios**:

1. **Given** a guest enters a delivery address, **When** they move to next field or click proceed, **Then** system geocodes the address to obtain coordinates
2. **Given** address is successfully geocoded, **When** coordinates are obtained, **Then** system checks if location falls within tenant's configured service area
3. **Given** location is within service area AND tenant has automatic delivery fees enabled, **When** serviceability is confirmed, **Then** system calculates delivery fee using tenant's distance-based or zone-based pricing rules
4. **Given** delivery fee is calculated (or set to 0 if disabled), **When** checkout summary updates, **Then** itemized costs show subtotal, delivery fee, and total clearly
5. **Given** address cannot be geocoded AND tenant has automatic fees enabled, **When** validation fails, **Then** guest sees error message requesting address clarification
6. **Given** tenant has multiple pricing zones AND automatic fees enabled, **When** address falls in specific zone, **Then** correct zone-based delivery fee is applied
7. **Given** tenant has disabled automatic delivery fee calculation, **When** guest selects delivery type, **Then** delivery_fee is always 0 regardless of address or distance

---

### User Story 7 - Access Tenant-Specific Public Menu (Priority: P3)

Each tenant in the multi-tenant system has its own unique public menu URL that displays only that tenant's products and branding.

**Why this priority**: Essential for multi-tenant architecture but can be implemented after core ordering flow is proven. Initial MVP could work with a single tenant or default tenant selection.

**Independent Test**: Can be fully tested by accessing different tenant URLs and verifying that each displays unique product catalogs, branding, and tenant-specific configurations (delivery options, service areas).

**Acceptance Scenarios**:

1. **Given** a multi-tenant system with multiple tenants, **When** a guest accesses a tenant-specific URL, **Then** they see only products from that specific tenant
2. **Given** a guest is viewing a tenant's public menu, **When** the page loads, **Then** tenant name and branding are displayed prominently
3. **Given** an invalid or inactive tenant URL is accessed, **When** the page loads, **Then** guest sees a friendly error message indicating the store is unavailable
4. **Given** a guest has items from Tenant A in their cart, **When** they navigate to Tenant B's URL, **Then** they start a separate cart for Tenant B and cart from Tenant A is preserved separately
5. **Given** different tenants have different delivery type configurations, **When** guest proceeds to checkout, **Then** they see only delivery types enabled for that specific tenant

---

### Edge Cases

- What happens when a Midtrans payment notification is received multiple times for the same order (idempotency)?
- How does the system handle network failures during Midtrans payment gateway redirect?
- What happens when a guest abandons their cart before completing payment?
- How does the system handle concurrent orders depleting the last available inventory unit?
- What happens if the Midtrans notification arrives before the guest redirect completes?
- How does the system handle malformed or fraudulent Midtrans payment notifications?
- What happens when a guest tries to order more quantity than available inventory?
- What happens when address geocoding service is unavailable or times out?
- How does the system handle addresses that are on the border of the service area?
- What happens when a guest's session expires during the checkout process?
- How does tenant staff handle orders that need to be cancelled after payment?
- What happens when delivery fee calculation fails or returns invalid results?
- How does the system handle very large cart quantities or unusual product combinations?
- What happens when a guest enters a valid address format but non-existent location?
- How does the system handle partial inventory availability (guest wants 5, only 3 available)?

## Requirements *(mandatory)*

### Functional Requirements

#### Public Menu & Product Browsing
- **FR-001**: System MUST provide a unique public URL for each tenant that displays the tenant's product catalog without requiring authentication
- **FR-002**: System MUST display product information including name, description, price, images, and availability status on the public menu
- **FR-003**: Public menu MUST be optimized for mobile devices with responsive design (target: <2s page load on 3G connections per plan.md performance goals, responsive breakpoints at 320px/768px/1024px)
- **FR-004**: System MUST allow filtering or categorizing products for easier browsing
- **FR-005**: System MUST display tenant branding (name, logo, colors) on the public menu

#### Guest Cart Management
- **FR-006**: System MUST allow guest users to add products to a cart with specified quantities without authentication
- **FR-007**: System MUST allow guests to modify cart contents (update quantities, remove items)
- **FR-008**: System MUST persist guest cart data in browser session storage (localStorage) to survive page refreshes and session interruptions
- **FR-009**: System MUST display real-time cart totals including itemized list and total amount
- **FR-010**: System MUST isolate guest cart data per tenant in multi-tenant scenarios
- **FR-011**: System MUST validate requested quantities against available inventory before adding to cart
- **FR-011a**: When requested quantity exceeds available inventory, system MUST automatically add maximum available quantity and display notification to guest (e.g., "Only X available, added to cart")

#### Delivery Type Selection & Contact Information
- **FR-012**: System MUST allow tenants to configure which delivery types are enabled (pickup, delivery, dine-in)
- **FR-013**: System MUST display only tenant-enabled delivery types at checkout
- **FR-014**: System MUST collect guest name and phone number for all delivery types
- **FR-015**: System MUST validate phone number format before accepting
- **FR-016**: System MUST collect complete delivery address when delivery type is selected
- **FR-017**: System MUST optionally collect table number or seating preference for dine-in orders
- **FR-018**: System MUST validate all required fields based on selected delivery type before allowing payment

#### Address Geocoding & Serviceability
- **FR-019**: System MUST geocode delivery addresses to obtain geographic coordinates
- **FR-020**: System MUST validate that geocoded address falls within tenant's configured service area
- **FR-021**: System MUST display clear error messages when addresses are outside service area (e.g., "Delivery not available to this location. We currently serve areas within [X km/zone name]")
- **FR-022**: System MUST handle geocoding failures gracefully with user-friendly error messages (e.g., "Unable to verify address. Please check spelling or try a nearby landmark")
- **FR-023**: System MUST allow tenants to configure service area boundaries (radius or polygon zones)

#### Delivery Fee Calculation
- **FR-024**: System MUST allow tenants to enable or disable automatic delivery fee calculation
- **FR-025**: When automatic delivery fees are enabled, system MUST support distance-based delivery fee calculation
- **FR-026**: When automatic delivery fees are enabled, system MUST support zone-based delivery fee calculation
- **FR-027**: System MUST display delivery fee separately in order summary before payment (showing 0 when automatic calculation is disabled)
- **FR-028**: System MUST include delivery fee in total amount sent to payment gateway (0 when automatic calculation is disabled)
- **FR-029**: System MUST store delivery fee with order record for reporting purposes (0 when automatic calculation is disabled, actual fee recorded when tenant collects manually)

#### Order Creation & Checkout
- **FR-030**: System MUST create a pending order when guest completes checkout form
- **FR-031**: System MUST generate a unique order reference number for each order
- **FR-032**: System MUST store order with items, quantities, prices at time of order, delivery type, contact info, and address
- **FR-033**: System MUST display an order summary before initiating payment
- **FR-034**: System MUST validate cart contents against current inventory before creating order
- **FR-035**: System MUST prevent order creation if any cart items are no longer available

#### Midtrans QRIS Payment Integration
- **FR-036**: System MUST integrate with Midtrans payment gateway for QRIS payment processing
- **FR-037**: System MUST create Midtrans payment request server-side with order details (amount, order reference)
- **FR-038**: System MUST redirect guests to Midtrans QRIS payment page
- **FR-039**: System MUST provide secure webhook endpoint at /payments/midtrans/notification to receive payment notifications (secured via signature verification per FR-040, FR-073, FR-076)
- **FR-040**: System MUST verify Midtrans notification signatures to ensure authenticity
- **FR-041**: System MUST validate that payment amount in notification matches order amount
- **FR-042**: System MUST handle notification idempotency to prevent duplicate order processing
- **FR-043**: System MUST update order status to PAID when successful payment notification is received
- **FR-044**: System MUST handle payment failures, cancellations, and expiration statuses from Midtrans
- **FR-045**: System MUST log all Midtrans API interactions and notifications for audit purposes
- **FR-046**: System MUST handle both Midtrans notifications and browser redirects for payment status

#### Payment Status & Order Confirmation
- **FR-047**: System MUST redirect guest back to tenant's site after payment attempt
- **FR-048**: System MUST display order confirmation with order reference number after successful payment
- **FR-049**: System MUST display order summary including delivery type, address (if applicable), and expected fulfillment details
- **FR-050**: System MUST display clear error messages for failed or cancelled payments (e.g., "Payment unsuccessful. Please try again or contact support if issue persists. Order #[ref]")
- **FR-051**: System MUST allow guests to retry payment from failed payment screen using same order reference and cart contents
- **FR-051a**: System MUST maintain order status as PENDING and preserve inventory reservation during payment retry attempts
- **FR-052**: System MUST allow guests to check order status using order reference number

#### Inventory Management with Reservations
- **FR-053**: System MUST create temporary inventory reservations when pending order is created
- **FR-054**: Temporary inventory reservations MUST have a configurable TTL (default: 15 minutes)
- **FR-055**: System MUST release expired inventory reservations automatically via background job
- **FR-055a**: System MUST allow guests to immediately retry checkout after reservation expiration without cooldown period, creating new reservation if inventory still available
- **FR-056**: System MUST convert temporary reservations to permanent inventory reduction when order status changes to PAID
- **FR-057**: System MUST release inventory reservations only when payment expires beyond retry window (not on initial failure or cancellation)
- **FR-058**: System MUST prevent overselling by enforcing inventory constraints during checkout
- **FR-059**: System MUST handle race conditions when multiple guests attempt to order the last available inventory unit
- **FR-060**: System MUST mark products as unavailable on public menu when available inventory (minus active reservations) reaches zero
- **FR-061**: System MUST adjust cart quantities to maximum available when guest requests exceed inventory, displaying clear notification of adjustment

#### Tenant Staff Order Management
- **FR-062**: System MUST provide admin dashboard for tenant staff to view orders
- **FR-063**: Admin dashboard MUST allow filtering orders by status (PENDING, PAID, COMPLETE, CANCELLED)
- **FR-064**: Admin dashboard MUST display full order details including items, delivery type, contact info, address, payment status
- **FR-065**: System MUST allow tenant staff to update order status from PAID to COMPLETE
- **FR-066**: System MUST record timestamp when order status is changed to COMPLETE
- **FR-067**: System MUST allow tenant staff to add notes to orders (e.g., courier info, delivery tracking)
- **FR-068**: System MUST allow tenant staff to cancel orders with reason documentation
- **FR-069**: System SHOULD send notifications to guests when order status changes (future enhancement)

#### Security & Data Protection
- **FR-070**: System MUST validate and sanitize all guest inputs to prevent injection attacks
- **FR-071**: System MUST use secure HTTPS connections for all public menu pages and API calls
- **FR-072**: System MUST implement rate limiting on public endpoints to prevent abuse
- **FR-073**: System MUST verify Midtrans notification signatures to prevent fraudulent order confirmations
- **FR-074**: System MUST not store sensitive payment information (card details, QRIS credentials)
- **FR-075**: System MUST generate cryptographically secure order reference numbers that are not easily guessable
- **FR-076**: System MUST protect webhook endpoint from unauthorized access using signature verification
- **FR-077**: System MUST log security events including failed signature verifications and suspicious activity

#### Session & State Management
- **FR-078**: System MUST maintain guest session state throughout the browsing and ordering process
- **FR-079**: System MUST handle session expiration gracefully with appropriate user messaging
- **FR-079a**: When browser session expires during checkout, system MUST restore cart contents from session storage (localStorage), but checkout form data must be re-entered
- **FR-080**: System MUST preserve order state across Midtrans payment gateway redirects
- **FR-081**: System MUST clean up abandoned guest carts after a configurable retention period (default: 24 hours)
- **FR-082**: System MUST clean up expired inventory reservations in background job
- **FR-083**: System MUST store minimal guest data (no account creation) and clean up after order completion or expiration

### Key Entities

- **Tenant**: Represents a business entity in the multi-tenant system. Contains: tenant identifier, business name, public menu URL, active status, contact information, branding (logo, colors), enabled delivery types, service area boundaries, delivery fee pricing rules, and associated product catalog.

- **Product**: Represents an item available for purchase. Contains: product identifier, name, description, price, images, category, current inventory count, availability status, and tenant association.

- **Guest Cart**: Represents a temporary shopping cart for an unauthenticated user. Contains: cart identifier, session identifier, tenant association, list of cart items with quantities, creation timestamp, last updated timestamp, and expiration time. Persisted in browser session storage (localStorage) to survive page refreshes and session interruptions.

- **Guest Order**: Represents a purchase transaction initiated by a guest user. Contains: order reference number, order status (PENDING, PAID, COMPLETE, CANCELLED), tenant association, list of ordered items with quantities and prices at time of order, subtotal amount, delivery fee, total amount, delivery type (pickup, delivery, dine-in), customer contact info (name, phone), delivery address with coordinates (if delivery type), optional table number or notes, timestamps for creation/payment/completion, Midtrans transaction reference.

- **Inventory Reservation**: Represents a temporary hold on product inventory during checkout process. Contains: reservation identifier, order reference, product identifier, reserved quantity, creation timestamp, expiration timestamp (based on TTL), reservation status (active, expired, converted, released).

- **Payment Transaction**: Represents an interaction with Midtrans payment gateway. Contains: transaction identifier, order reference, Midtrans transaction ID, amount, payment method (QRIS), payment status, notification payload, signature verification status, timestamps for initiation/notification/completion.

- **Delivery Address**: Represents a validated delivery location for delivery orders. Contains: order reference, address text, geocoded coordinates (latitude, longitude), service area zone, calculated delivery fee, geocoding status, validation timestamp.

- **Tenant Configuration**: Represents operational settings for a tenant. Contains: tenant identifier, enabled delivery types array, automatic delivery fee calculation flag (enabled/disabled), service area definition (radius or polygon coordinates), delivery fee pricing structure (distance tiers or zone mappings, null when automatic calculation disabled), inventory reservation TTL, business hours, contact information.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Guests can complete the entire ordering flow from browsing to payment confirmation in under 5 minutes on a mobile device
- **SC-002**: System successfully processes 95% of Midtrans payment notifications within 3 seconds of receipt
- **SC-003**: Inventory overselling occurs in less than 0.1% of concurrent order scenarios
- **SC-004**: Public menu pages load and display products in under 2 seconds on 3G mobile connections
- **SC-005**: 90% of guests who initiate checkout successfully complete payment without errors
- **SC-006**: System handles at least 100 concurrent guest users browsing and ordering without performance degradation
- **SC-007**: Midtrans notification signature verification succeeds for 100% of legitimate notifications
- **SC-008**: Address geocoding completes within 2 seconds for 95% of delivery orders
- **SC-009**: Delivery fee calculation completes within 1 second for 98% of orders
- **SC-010**: Abandoned cart cleanup and expired inventory reservation release processes run automatically every 5 minutes without manual intervention
- **SC-011**: Zero successful fraudulent orders from forged Midtrans notifications
- **SC-012**: Tenant staff can update order status from PAID to COMPLETE in under 30 seconds
- **SC-013**: System correctly validates 100% of addresses against tenant service area boundaries
- **SC-014**: Inventory reservations expire and release automatically within 1 minute of TTL expiration
- **SC-015**: Each tenant's public menu displays only their products with 100% accuracy in multi-tenant scenarios
