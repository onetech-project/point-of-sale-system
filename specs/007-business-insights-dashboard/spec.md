# Feature Specification: Business Insights Dashboard

**Feature Branch**: `007-business-insights-dashboard`  
**Created**: 2026-01-31  
**Status**: Draft  
**Input**: User description: "as a tenant's owner i need a dashboard that provide business insight, like, total sales current month, top 5 best seller vs top 5 non perform product by quantity and total sales current month, total inventory/product value in cash, net profit current month. then task that i need to immediately do (ToDo) like, 5 order above 15 minutes, restock of out of stock/low stock product. Quick Action: Invite Team Members, Settings"

## Clarifications

### Session 2026-01-31

- Q: What time granularities should be supported for chart visualization and filtering? → A: Daily, weekly, and monthly views with current month as default
- Q: How far back should the date range filter allow viewing data? → A: Multi-level filter approach - user selects time series first (daily/weekly/monthly/quarterly/yearly), then appropriate range (daily=last 30 days, monthly=last 3 months, quarterly=last 4 quarters, yearly=last 3 years)
- Q: Which specific metrics should be displayed as charts? → A: Sales revenue, order count, and product performance, plus top 5 best spender customers (unique by phone number)
- Q: Which chart type best suits the visualization needs? → A: Mixed chart types - different types for different metrics (e.g., bars for orders, lines for sales trends)
- Q: How should customer uniqueness be determined for top spenders when both phone and email exist? → A: Phone number priority as it is mandatory for guest orders

## User Scenarios & Testing _(mandatory)_

### User Story 1 - View Current Month Sales Performance (Priority: P1)

Tenant owners need immediate visibility into their current month's sales performance to make informed operational decisions. They view total sales revenue, identify which products are selling well versus underperforming, understand their profitability, and see their top spending customers to adjust pricing, promotions, and inventory strategies.

**Why this priority**: This is the core value proposition of the dashboard - providing real-time business insights that directly impact decision-making. Without sales and performance data, owners cannot effectively manage their business.

**Independent Test**: Can be fully tested by logging in as a tenant owner, viewing the dashboard, and verifying that sales metrics (total sales, top/bottom performers, net profit, top customers) are displayed accurately for the current month. Delivers immediate value by showing business performance at a glance.

**Acceptance Scenarios**:

1. **Given** a tenant owner is logged in, **When** they access the dashboard, **Then** they see total sales revenue for the current month displayed prominently
2. **Given** the dashboard is loaded, **When** the owner views the product performance section, **Then** they see the top 5 best-selling products ranked by quantity sold and total sales value
3. **Given** the dashboard is loaded, **When** the owner views the product performance section, **Then** they see the 5 worst-performing products ranked by quantity sold and total sales value
4. **Given** the dashboard is loaded, **When** the owner views financial metrics, **Then** they see net profit calculated for the current month
5. **Given** the dashboard is loaded, **When** the owner views customer insights, **Then** they see the top 5 best spending customers identified by phone number with total spending amounts
6. **Given** products have been sold this month, **When** the dashboard loads, **Then** product performance metrics update to reflect current data
7. **Given** it is a new month, **When** the owner accesses the dashboard, **Then** all current month metrics reset and show data from the new month only

---

### User Story 2 - Monitor Inventory Health and Value (Priority: P2)

Tenant owners need to understand the total value of their inventory investment to manage cash flow and make purchasing decisions. They view the total monetary value of all products in stock to assess capital tied up in inventory and identify optimization opportunities.

**Why this priority**: Inventory represents a significant capital investment. Understanding inventory value helps owners optimize their working capital, plan purchases, and identify slow-moving stock that ties up funds.

**Independent Test**: Can be fully tested by viewing the dashboard's inventory section and verifying that the total inventory value (sum of all products' cost × quantity) is calculated and displayed correctly. Delivers value by showing capital allocation.

**Acceptance Scenarios**:

1. **Given** a tenant owner is logged in, **When** they view the dashboard, **Then** they see the total inventory value displayed as a monetary amount
2. **Given** inventory quantities or costs change, **When** the dashboard refreshes, **Then** the total inventory value updates to reflect current stock levels and costs
3. **Given** products are out of stock, **When** calculating inventory value, **Then** those products contribute zero to the total value
4. **Given** products have different cost prices, **When** viewing inventory value, **Then** the calculation accounts for each product's specific cost and quantity

---

### User Story 3 - Manage Urgent Operational Tasks (Priority: P1)

Tenant owners need immediate alerts about time-sensitive operational issues to maintain service quality and customer satisfaction. They view and act on critical tasks such as delayed orders and inventory shortages that require immediate attention.

**Why this priority**: Delayed orders directly impact customer satisfaction and reputation. Stock-outs lead to lost sales. These are urgent issues that need immediate visibility and action, making this a critical feature for day-to-day operations.

**Independent Test**: Can be fully tested by creating orders that exceed 15 minutes in processing time and setting products to low/out of stock levels, then verifying these appear in the task list. Delivers immediate operational value by highlighting urgent issues.

**Acceptance Scenarios**:

1. **Given** orders exist that have been unprocessed for over 15 minutes, **When** the owner views the dashboard, **Then** they see a task alert showing the count of delayed orders
2. **Given** delayed order alerts are displayed, **When** the owner clicks on the alert, **Then** they can view the list of specific delayed orders with order IDs and time elapsed
3. **Given** products are out of stock or below a low stock threshold, **When** the owner views the dashboard, **Then** they see a task alert for restock needs
4. **Given** restock alerts are displayed, **When** the owner clicks on the alert, **Then** they see a list of products requiring restocking with current quantities
5. **Given** an order is processed within 15 minutes, **When** the dashboard refreshes, **Then** that order is removed from the delayed orders count
6. **Given** a product is restocked above the threshold, **When** the dashboard refreshes, **Then** that product is removed from the restock alert list

---

### User Story 4 - Quick Access to Common Actions (Priority: P3)

Tenant owners need quick access to frequently performed administrative tasks without navigating through multiple menus. They use quick action shortcuts to invite team members and access settings efficiently.

**Why this priority**: While useful for efficiency, this is a convenience feature that enhances user experience but is not critical for core business operations. The same actions can be performed through regular navigation.

**Independent Test**: Can be fully tested by clicking quick action buttons and verifying they navigate to the correct pages or open the correct dialogs. Delivers value through improved workflow efficiency.

**Acceptance Scenarios**:

1. **Given** a tenant owner is on the dashboard, **When** they click the "Invite Team Members" quick action, **Then** they are directed to the team invitation interface
2. **Given** a tenant owner is on the dashboard, **When** they click the "Settings" quick action, **Then** they are directed to the tenant settings page
3. **Given** quick actions are displayed, **When** the owner hovers over them, **Then** they see clear labels indicating what each action does

---

### User Story 5 - Analyze Trends with Data Visualizations (Priority: P1)

Tenant owners need to visualize business trends over time to identify patterns, compare performance across periods, and make data-driven strategic decisions. They view interactive charts showing sales revenue, order counts, and product performance with flexible time-based filtering.

**Why this priority**: Visual trend analysis is critical for understanding business trajectory, identifying seasonal patterns, and making informed forecasting decisions. Charts make complex data immediately interpretable.

**Independent Test**: Can be fully tested by accessing the dashboard, selecting different time series (daily/weekly/monthly), adjusting date ranges, and verifying that charts update to show accurate historical trends. Delivers value by revealing patterns not visible in static numbers.

**Acceptance Scenarios**:

1. **Given** a tenant owner is on the dashboard, **When** they view the default state, **Then** they see charts displaying current month data
2. **Given** the owner wants to change time granularity, **When** they select a time series option (daily/weekly/monthly/quarterly/yearly), **Then** the appropriate date range filter appears (daily=last 30 days, monthly=last 3 months, quarterly=last 4 quarters, yearly=last 3 years)
3. **Given** a time series is selected, **When** the owner adjusts the date range filter, **Then** all charts update to show data for the selected period
4. **Given** charts are displayed, **When** the owner views them, **Then** sales revenue is shown in an appropriate chart type for trend visualization
5. **Given** charts are displayed, **When** the owner views them, **Then** order count is shown in an appropriate chart type
6. **Given** charts are displayed, **When** the owner views them, **Then** product performance trends are visualized in an appropriate chart format
7. **Given** multiple metrics are charted, **When** viewing different chart types, **Then** each metric uses the most suitable visualization (mixed chart approach)
8. **Given** the owner selects a historical period with no data, **When** charts load, **Then** empty states are displayed with appropriate messaging
9. **Given** charts display time-series data, **When** the owner hovers over data points, **Then** they see tooltips with exact values and dates

---

### Edge Cases

- What happens when there are no sales in the current month? (Display zero values with appropriate messaging)
- How does the system handle products with no sales ever? (Include in bottom 5 performers if fewer than 5 products have sales)
- What happens when there are fewer than 5 products in inventory? (Display all available products, not necessarily 5)
- How does the system calculate net profit when there are returns or refunds? (Subtract refunds from gross sales, add back refunded COGS)
- What happens on the first day of a new month at midnight? (Metrics should reset cleanly to show new month data)
- How are delayed orders counted when an order contains multiple items at different processing stages? (Order is delayed if any item is unprocessed beyond threshold)
- What is the low stock threshold for restock alerts? (Configurable per tenant in settings, with sensible default like 10 units or 20% of average monthly sales)
- How does the dashboard handle products with zero or negative cost price? (Include in calculations but flag for review)
- What happens when multiple tenant users view the dashboard simultaneously? (Each sees the same data; real-time updates not required, refresh on page load is acceptable)
- How does the system handle very large numbers (millions in sales)? (Format with appropriate abbreviations: 1.5M, 2.3K, etc.)
- How are customers identified as unique for top spenders when they place multiple orders? (Use phone number as primary identifier since it is mandatory for guest orders)
- What happens when a customer has both phone and email but orders under different phone numbers? (Each unique phone number is treated as a separate customer)
- How does the chart handle data points when there are no sales for certain periods? (Display zero values on the chart to show gaps clearly)
- What happens when the selected date range has insufficient data for the chosen granularity? (Display all available data points with a message indicating limited data)
- How does the multi-level filter behave when switching between time series? (Automatically adjusts to the appropriate default range for the new time series)

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST display total sales revenue for the current calendar month, calculated as the sum of all completed order totals
- **FR-002**: System MUST identify and display the top 5 best-selling products ranked by quantity sold in the current month
- **FR-003**: System MUST identify and display the top 5 best-selling products ranked by total sales value in the current month
- **FR-004**: System MUST identify and display the 5 worst-performing products ranked by quantity sold in the current month (or all products if fewer than 5)
- **FR-005**: System MUST identify and display the 5 worst-performing products ranked by total sales value in the current month
- **FR-006**: System MUST calculate and display net profit for the current month as (total sales - total cost of goods sold - operational costs)
- **FR-007**: System MUST calculate and display total inventory value as the sum of (product cost × current quantity) for all products
- **FR-008**: System MUST display a count of orders that have been in processing status for more than 15 minutes
- **FR-009**: System MUST allow owners to view the list of delayed orders with order ID, customer information, and time elapsed
- **FR-010**: System MUST identify products that are out of stock (quantity = 0) or below the low stock threshold
- **FR-011**: System MUST display restock alerts showing product name, current quantity, and low stock threshold
- **FR-012**: System MUST provide a quick action button to navigate to the team invitation interface
- **FR-013**: System MUST provide a quick action button to navigate to tenant settings
- **FR-014**: System MUST calculate metrics using only data belonging to the logged-in owner's tenant (data isolation)
- **FR-015**: System MUST update dashboard metrics when the page is loaded or manually refreshed
- **FR-016**: System MUST display zero or empty states gracefully when no data is available for a metric
- **FR-017**: System MUST format large monetary values with appropriate abbreviations (K for thousands, M for millions)
- **FR-018**: System MUST calculate current month as the period from the 1st to the last day of the current calendar month in the tenant's timezone
- **FR-019**: System MUST only count completed/paid orders in sales calculations (exclude cancelled, pending, or abandoned orders)
- **FR-020**: System MUST allow tenant owners to configure the low stock threshold for restock alerts in settings
- **FR-021**: System MUST provide a time series selector with options for daily, weekly, monthly, quarterly, and yearly views
- **FR-022**: System MUST set current month as the default view when the dashboard loads
- **FR-023**: System MUST provide date range filters that adjust based on selected time series: daily (last 30 days), monthly (last 3 months), quarterly (last 4 quarters), yearly (last 3 years)
- **FR-024**: System MUST display sales revenue trends in chart format using an appropriate visualization type
- **FR-025**: System MUST display order count trends in chart format using an appropriate visualization type
- **FR-026**: System MUST display product performance trends in chart format using an appropriate visualization type
- **FR-027**: System MUST use mixed chart types optimized for each metric's characteristics
- **FR-028**: System MUST identify and display the top 5 best spending customers ranked by total order value
- **FR-029**: System MUST use phone number as the primary identifier for determining unique customers
- **FR-030**: System MUST treat each unique phone number as a separate customer even if email addresses differ
- **FR-031**: System MUST update all charts when time series or date range filters are changed
- **FR-032**: System MUST display tooltips on chart data points showing exact values and corresponding dates
- **FR-033**: System MUST handle periods with no data by displaying zero values or empty state indicators in charts
- **FR-034**: System MUST aggregate data according to the selected time series granularity (daily/weekly/monthly/quarterly/yearly)

### Key Entities

- **Sales Metrics**: Represents aggregated sales data including total revenue, order count, and profit for a time period (current month). Related to orders and products.
- **Product Performance**: Represents a product's sales performance including quantity sold, total sales value, and ranking. Derived from orders and product data.
- **Inventory Value**: Represents the monetary value of current inventory, calculated from product cost and quantity. Related to product data.
- **Operational Task**: Represents an urgent action item such as a delayed order or restock need. Includes type (delayed order/restock), count, and related entity references (order IDs or product IDs).
- **Delayed Order**: Represents an order that has exceeded the processing time threshold. Includes order information, customer data, and time elapsed.
- **Restock Alert**: Represents a product requiring restocking. Includes product information, current quantity, and threshold level.
- **Chart Visualization**: Represents time-series data displayed in chart format. Includes metric type (sales/orders/products), time granularity, date range, data points, and chart type.
- **Time Series Filter**: Represents the user's selected time granularity and date range. Includes granularity (daily/weekly/monthly/quarterly/yearly) and corresponding range constraints.
- **Customer Spending**: Represents aggregated spending data for a unique customer identified by phone number. Includes total spending amount, order count, and ranking among all customers.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Tenant owners can view all key business metrics (sales, profit, inventory value) within 2 seconds of loading the dashboard
- **SC-002**: Owners can identify their top performing and worst performing products within 5 seconds of viewing the dashboard
- **SC-003**: Dashboard displays delayed order alerts within 1 minute of an order exceeding the 15-minute threshold
- **SC-004**: Owners can access team invitation and settings features in a single click from the dashboard
- **SC-005**: Dashboard calculations are accurate to within 99.9% of actual values (accounting for rounding)
- **SC-006**: Dashboard supports tenants with up to 10,000 products and 50,000 monthly orders without performance degradation
- **SC-007**: 95% of tenant owners successfully interpret their business metrics without requiring support or documentation
- **SC-008**: Dashboard metrics reflect data changes within 30 seconds of a page refresh
- **SC-009**: Zero data leakage between tenants - owners only see their own tenant's data in 100% of cases
- **SC-010**: Owners can complete the workflow from seeing a restock alert to viewing product details in under 10 seconds
- **SC-011**: Owners can change time series selection and see updated charts within 3 seconds
- **SC-012**: Charts render with at least 95% accuracy compared to raw data values
- **SC-013**: Dashboard supports rendering charts with up to 365 data points (daily for 1 year) without performance degradation
- **SC-014**: Owners can identify top spending customers within 5 seconds of viewing the dashboard
- **SC-015**: Chart tooltips display with sub-second response time when hovering over data points
