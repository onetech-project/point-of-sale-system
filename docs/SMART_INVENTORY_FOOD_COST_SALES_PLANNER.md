You are a senior software architect and backend engineer working on an existing multi-tenant POS system.

Your task is to design and implement the foundation for a new Inventory, Recipe Costing, Bundle, Discount, and Sales Planning module without breaking the existing product catalog and order flows.

Do not blindly generate code. Start by auditing the repository, understanding the existing architecture, identifying reusable patterns, and producing a concrete implementation plan. Then implement only the approved first phase described below.

==================================================
PROJECT CONTEXT
==================================================

This is an existing POS system, not a greenfield project.

Current capabilities:

- Multi-tenant registration
- Team management
- Product catalog and product management
- Guest online orders
- QRIS payment integration through Midtrans
- Order management
- Business insight dashboard
- GDPR / UU PDP considerations
- Kafka-based event-driven communication
- PostgreSQL
- Redis
- Docker Compose
- CI/CD pipeline
- API Gateway
- Backend services written in Go
- Frontend written with Next.js App Router and TypeScript

Existing backend services may include:

- auth-service
- user-service
- tenant-service
- product-service
- order-service
- notification-service
- api-gateway

Inspect the actual repository before assuming exact folder names, framework conventions, database migration tools, or message formats.

==================================================
BUSINESS GOAL
==================================================

Add a new feature set for F&B and small production businesses:

1. Raw material management
   - Users can register ingredients or raw materials.
   - Each ingredient has a base unit, stock quantity, monetary value, average cost, minimum stock threshold, and active status.
   - Supported units include weight, volume, count, and packaging units:
     - kilogram
     - gram
     - milligram
     - liter
     - milliliter
     - piece
     - portion
     - slice
     - egg / item count
     - tray
     - box
     - sachet
     - bottle
     - pack
     - custom packaging units

2. Product recipe and food-cost calculation
   - When creating or editing a product, users can attach raw materials as a recipe.
   - Each recipe item defines:
     - ingredient
     - quantity
     - selected unit
     - normalized quantity in the ingredient base unit
     - optional waste percentage
   - Each product may define overhead costs.
   - Overhead costs initially support:
     - fixed amount per product
     - percentage of ingredient material cost
   - System calculates:
     - material cost
     - overhead cost
     - total COGS
     - selling price
     - gross profit
     - gross margin percentage

3. Bundle or package management
   - Users can create bundles consisting of multiple products and quantities.
   - Bundle material usage must be calculated automatically by expanding each product recipe.
   - Bundle COGS must be calculated automatically.
   - Do not duplicate product recipes inside bundles.
   - Nested bundles are not required in the MVP.

4. Discount management
   - Users can configure discounts for products or bundles.
   - Discounts support:
     - percentage discount
     - fixed amount discount
     - active date range
     - minimum quantity
     - priority
     - exclusive or stackable policy
   - MVP default: discounts are exclusive unless explicitly configured otherwise.

5. Strategic dashboard and forecasting
   - Users can enter a revenue target for a selected period.
   - The system calculates:
     - monthly target
     - weekly target
     - daily target
     - recommended product or bundle sales mix
     - estimated ingredient usage
     - estimated raw-material budget
     - gross profit
     - gross margin
     - ingredient shortage
   - Alternatively, users can enter an available raw-material budget.
   - The system proposes a realistic product sales mix based on:
     - historical sales
     - product gross margin
     - stock readiness
     - product trend
     - bundle opportunities
   - Do not use AI, LLMs, or machine learning in the initial implementation.
   - Use deterministic calculations and a simple explainable scoring mechanism.

==================================================
IMPORTANT DOMAIN RULES
==================================================

1. Keep inventory and recipe costing separate from the existing product catalog domain.

Recommended boundary:

- product-service:
  - product catalog
  - bundles
  - pricing rules
  - selling prices

- inventory-service:
  - ingredients
  - units of measure
  - ingredient-specific unit conversions
  - recipes
  - recipe versions
  - stock ledger
  - stock adjustments
  - stock consumption
  - waste
  - costing

- order-service:
  - order lifecycle
  - selected product or bundle
  - applied discounts
  - fulfillment status
  - historical selling-price snapshot
  - historical COGS snapshot

- analytics module or analytics-service:
  - historical profitability dashboard
  - sales target planner
  - raw-material forecasting
  - recommended sales mix

Adapt this boundary if the existing repository structure suggests a better fit, but document the reasoning.

2. Do not store ingredients as a JSON field directly inside the product table.

3. All relevant business tables must include tenant scoping.
   - Use tenant_id consistently.
   - If outlet or warehouse support already exists, use outlet_id or warehouse_id for stock ownership.
   - If it does not exist, design the schema so multi-outlet support can be added cleanly later.

4. Use decimal-safe numeric database types.
   - Never use floating-point types for money or ingredient quantities.
   - Prefer NUMERIC or DECIMAL types.
   - Example:
     - quantities: NUMERIC(18, 6)
     - money: NUMERIC(18, 2) or the existing project convention

5. Unit conversion rules:
   - Allow global conversions only inside the same unit category:
     - 1 kilogram = 1000 gram
     - 1 gram = 1000 milligram
     - 1 liter = 1000 milliliter
   - Do not globally convert weight to volume or count.
   - Packaging conversions must be ingredient-specific:
     - 1 tray of eggs = 30 pieces
     - 1 sack of flour = 25 kilograms
     - 1 cheese block = 100 slices

6. Stock must use a ledger model.
   - A current_stock field may exist as a derived or cached value.
   - The source of truth must be stock movements.

Required movement types:

- INITIAL_STOCK
- PURCHASE_IN
- SALE_RESERVATION
- SALE_RESERVATION_RELEASED
- SALE_CONSUMPTION
- WASTE
- MANUAL_ADJUSTMENT
- RETURN_IN
- RETURN_OUT

7. Historical reporting must be immutable.
   - Ingredient prices and recipes may change later.
   - Historical order profitability must not change retroactively.
   - Save recipe version and COGS snapshot when an order is fulfilled.

8. Inventory event processing must be idempotent.
   - Kafka retries must never deduct ingredients twice.
   - Use event_id or idempotency_key.
   - Enforce uniqueness at the database level where appropriate.

9. Inventory deduction lifecycle:
   - order.created:
     - do not deduct ingredients
   - order.confirmed:
     - optionally reserve ingredients
   - order.fulfilled:
     - consume ingredients
     - write stock ledger entries
     - save COGS snapshots
   - order.cancelled before fulfillment:
     - release reservations
   - refund after fulfillment:
     - do not automatically restore consumed ingredients
     - treat the refund as a financial event unless explicitly configured otherwise

==================================================
TARGET DATA MODEL
==================================================

Use the following as a reference, but inspect existing conventions first and adapt naming consistently.

Core tables:

uoms

- id
- code
- name
- category
- is_base_unit
- is_active
- created_at
- updated_at

ingredient_uom_conversions

- id
- tenant_id
- ingredient_id nullable for global conversions
- from_uom_id
- to_uom_id
- multiplier
- created_at
- updated_at

ingredients

- id
- tenant_id
- outlet_id nullable if not supported yet
- sku
- name
- base_uom_id
- current_stock_base
- average_cost_per_base_unit
- minimum_stock_base
- track_stock
- is_active
- created_at
- updated_at

stock_movements

- id
- tenant_id
- outlet_id nullable
- ingredient_id
- movement_type
- quantity_base
- unit_cost
- total_cost
- reference_type
- reference_id
- idempotency_key
- occurred_at
- created_by
- created_at

recipes

- id
- tenant_id
- product_id
- version
- yield_quantity
- is_active
- effective_from
- created_at
- updated_at

recipe_items

- id
- recipe_id
- ingredient_id
- quantity
- uom_id
- normalized_quantity_base
- waste_percentage
- created_at
- updated_at

recipe_overheads

- id
- recipe_id
- name
- calculation_type
- amount
- created_at
- updated_at

order_item_cost_snapshots

- id
- tenant_id
- order_item_id
- recipe_id
- recipe_version
- raw_material_cost
- overhead_cost
- total_cogs
- final_selling_price
- discount_amount
- gross_profit
- created_at

Future tables, not required in the first coding phase:

- bundles
- bundle_items
- pricing_rules
- pricing_rule_targets
- sales_plan_scenarios
- sales_plan_recommendations

==================================================
CALCULATION RULES
==================================================

Use the following formulas:

ingredient_cost =
normalized_quantity_base × average_cost_per_base_unit

recipe_item_cost_with_waste =
ingredient_cost × (1 + waste_percentage / 100)

recipe_material_cost =
sum(recipe_item_cost_with_waste)

fixed_overhead_cost =
sum(overheads where calculation_type = FIXED)

percentage_overhead_cost =
recipe_material_cost × sum(percentage overheads) / 100

menu_cogs =
recipe_material_cost + fixed_overhead_cost + percentage_overhead_cost

gross_profit =
selling_price - menu_cogs

gross_margin_percentage =
selling_price > 0
? gross_profit / selling_price × 100
: 0

For future bundle support:

bundle_cogs =
sum(product_cogs × bundle_item_quantity)

For future forecasting:

target_daily_revenue =
target_period_revenue / operational_days

required_ingredient_quantity =
sum(target_product_quantity × recipe_ingredient_quantity)

purchase_requirement =
max(required_ingredient_quantity - projected_available_stock, 0)

required_purchase_budget =
sum(purchase_requirement × average_cost_per_base_unit)

==================================================
KAFKA EVENTS
==================================================

Inspect the existing Kafka event envelope and reuse the same conventions.

Plan for these events:

Consumed from order-service:

- order.created
- order.confirmed
- order.fulfilled
- order.cancelled

Published by inventory-service:

- inventory.ingredients_reserved
- inventory.ingredients_consumed
- inventory.reservation_released
- inventory.low_stock_detected
- inventory.stock_adjusted
- catalog.recipe_updated

Each event must include:

- event_id
- event_type
- occurred_at
- tenant_id
- aggregate_id
- correlation_id if the existing project supports it
- payload

Do not introduce an incompatible event format if the repository already has an event envelope.

==================================================
IMPLEMENTATION PHASES
==================================================

Do not implement everything in one large change.

Phase 0 — Repository audit and technical design

1. Inspect the repository structure.
2. Identify:
   - backend services
   - framework patterns
   - database migration tooling
   - repository pattern
   - handler or controller conventions
   - validation conventions
   - authentication and tenant resolution
   - Kafka producer and consumer conventions
   - error response format
   - logging convention
   - test convention
   - Docker Compose setup
   - CI pipeline
3. Produce:
   - current architecture summary
   - proposed module placement
   - database ERD in Mermaid
   - API endpoint plan
   - Kafka sequence diagram in Mermaid
   - migration plan
   - implementation checklist
   - risks and assumptions
4. Create a technical-design Markdown document in the repository:
   docs/inventory-recipe-costing-technical-design.md

Phase 1 — Inventory foundation
Implement only:

1. inventory-service bootstrap if a dedicated service does not already exist
2. UoM master data
3. ingredients CRUD
4. ingredient-specific UoM conversions
5. stock ledger
6. stock initialization
7. stock purchase-in endpoint
8. manual stock adjustment endpoint
9. waste endpoint
10. ingredient stock history endpoint
11. low-stock query endpoint
12. tenant isolation
13. validation
14. database migrations
15. automated tests
16. Docker Compose integration
17. README updates

Do not implement recipe costing, order fulfillment integration, bundle management, discounts, or forecasting yet. Only prepare extensible schema boundaries where necessary.

Phase 2 — Recipe costing
Future scope:

- recipes
- recipe versioning
- recipe items
- overheads
- COGS calculation
- gross margin calculation
- product integration

Phase 3 — Order integration
Future scope:

- order.fulfilled consumer
- ingredient consumption
- idempotent processing
- reservation
- cancellation release
- order-item COGS snapshots

Phase 4 — Bundles and discount rules
Future scope:

- bundle management
- bundle cost expansion
- bundle stock usage
- product or bundle pricing rules
- discounts
- pricing snapshots

Phase 5 — Analytics and strategic dashboard
Future scope:

- historical profitability
- top-selling products
- high-margin products
- ingredient forecast
- target revenue simulator
- budget-based simulator
- explainable product recommendation scoring

==================================================
PHASE 1 REQUIRED API ENDPOINTS
==================================================

Adapt route naming to match the existing repository convention.

Suggested endpoints:

UoM:

- GET /api/v1/uoms
- POST /api/v1/uoms
- PUT /api/v1/uoms/:id
- DELETE /api/v1/uoms/:id

Ingredients:

- GET /api/v1/ingredients
- GET /api/v1/ingredients/:id
- POST /api/v1/ingredients
- PUT /api/v1/ingredients/:id
- DELETE /api/v1/ingredients/:id

Ingredient conversions:

- GET /api/v1/ingredients/:id/conversions
- POST /api/v1/ingredients/:id/conversions
- PUT /api/v1/ingredients/:id/conversions/:conversionId
- DELETE /api/v1/ingredients/:id/conversions/:conversionId

Inventory:

- POST /api/v1/inventory/initial-stock
- POST /api/v1/inventory/purchases
- POST /api/v1/inventory/adjustments
- POST /api/v1/inventory/waste
- GET /api/v1/inventory/ingredients/:ingredientId/movements
- GET /api/v1/inventory/low-stock
- GET /api/v1/inventory/valuation

Pagination, filtering, sorting, tenant scoping, and validation must follow existing repository conventions.

==================================================
VALIDATION REQUIREMENTS
==================================================

Implement validation for:

- duplicate UoM code
- invalid UoM category
- negative ingredient quantity where not allowed
- invalid conversion multiplier
- cross-category global conversion
- missing ingredient-specific conversion for packaging units
- missing tenant context
- inactive ingredients
- stock adjustment reason
- duplicate idempotency key
- insufficient stock where an outbound movement would result in invalid negative stock
- database transaction rollback when ledger update fails

Use database transactions for stock mutations.

==================================================
TEST REQUIREMENTS
==================================================

Follow existing project test conventions.

At minimum, add:

1. Unit tests:
   - global unit conversion
   - ingredient-specific conversion
   - stock valuation
   - low-stock detection
   - adjustment validation
   - waste validation

2. Repository or integration tests:
   - create ingredient
   - register initial stock
   - purchase stock
   - deduct waste
   - manual adjustment
   - ledger history
   - tenant isolation
   - duplicate idempotency key rejection
   - rollback on invalid movement

3. API tests if the project already uses them.

4. Run:
   - formatter
   - linter
   - unit tests
   - integration tests
   - build
   - Docker Compose validation if supported

Do not mark work as complete while tests or build are failing. Report failures honestly if an existing repository issue blocks completion.

==================================================
CODING RULES
==================================================

- Follow existing naming, package, and folder conventions.
- Keep changes minimal and modular.
- Do not refactor unrelated code.
- Do not add dependencies unless clearly justified.
- Prefer simple and explicit code.
- Apply DRY, KISS, and YAGNI.
- Add indexes for common tenant-scoped queries.
- Add database constraints where appropriate.
- Use transactions for stock ledger mutations.
- Preserve backward compatibility with existing product and order flows.
- Add comments only when they explain non-obvious business rules.
- Do not add mock logic or placeholder endpoints and call them completed.
- Do not implement nested bundles in MVP.
- Do not implement supplier management, expiry tracking, batch tracking, ML forecasting, or advanced purchase-order workflow yet.

==================================================
EXPECTED OUTPUT FORMAT
==================================================

Work in the following order:

1. Audit the repository.
2. Create:
   docs/inventory-recipe-costing-technical-design.md
3. Show a concise summary containing:
   - architecture findings
   - assumptions
   - files that will be created or modified
   - migration list
   - API list
   - test plan
4. Implement Phase 1 only.
5. Run tests, linting, build, and any relevant validation commands.
6. Show:
   - completed items
   - changed files
   - commands executed
   - test results
   - known limitations
   - recommended next step for Phase 2

If the repository structure conflicts with this prompt, prioritize the actual repository patterns, explain the conflict, and choose the safest compatible design.
