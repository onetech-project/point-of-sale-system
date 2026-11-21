package repository

import "fmt"

// Orders count as revenue once they are paid; COMPLETE orders were paid before completion.
const revenueStatusFilter = "status IN ('PAID', 'COMPLETE')"

// revenueAt is the moment an order counts as revenue: when it was paid, falling back to
// completion and then creation for orders that are missing the earlier timestamps.
// Matches the expression of idx_guest_orders_revenue_date.
func revenueAt(alias string) string {
	return fmt.Sprintf("COALESCE(%[1]spaid_at, %[1]scompleted_at, %[1]screated_at)", alias)
}

// itemProfitAggregates sums, per order, the gross profit and cost of goods sold of the items that
// have a recorded unit cost, plus the revenue of items without one. Profit only comes from the
// transaction snapshot in order_items, never from the product master. The delivery fee is not
// part of an item, so it is excluded from profit.
const itemProfitAggregates = `
	SELECT
		order_id,
		SUM(total_price - quantity * unit_cost) FILTER (WHERE unit_cost IS NOT NULL) AS gross_profit,
		SUM(quantity * unit_cost) FILTER (WHERE unit_cost IS NOT NULL) AS cost_of_goods,
		SUM(total_price) FILTER (WHERE unit_cost IS NULL) AS uncosted_revenue
	FROM order_items
	GROUP BY order_id`

// localDate converts a TIMESTAMPTZ expression to the calendar date in the given IANA timezone
func localDate(expr, timezone string) string {
	return fmt.Sprintf("(%s AT TIME ZONE '%s')::date", expr, timezone)
}

// localTimestamp converts a TIMESTAMPTZ expression to wall-clock time in the given IANA timezone
func localTimestamp(expr, timezone string) string {
	return fmt.Sprintf("(%s AT TIME ZONE '%s')", expr, timezone)
}
