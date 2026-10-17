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

// localDate converts a TIMESTAMPTZ expression to the calendar date in the given IANA timezone
func localDate(expr, timezone string) string {
	return fmt.Sprintf("(%s AT TIME ZONE '%s')::date", expr, timezone)
}

// localTimestamp converts a TIMESTAMPTZ expression to wall-clock time in the given IANA timezone
func localTimestamp(expr, timezone string) string {
	return fmt.Sprintf("(%s AT TIME ZONE '%s')", expr, timezone)
}
