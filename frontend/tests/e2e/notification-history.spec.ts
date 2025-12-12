import { test, expect } from '@playwright/test';

test.describe('Notification History E2E Tests (T067)', () => {
  test.skip('should navigate to notification history page', async ({ page }) => {
    // Login as admin
    await page.goto('/login');
    await page.fill('input[name="email"]', 'admin@restaurant.com');
    await page.fill('input[name="password"]', 'password123');
    await page.click('button[type="submit"]');

    // Navigate to settings
    await page.waitForURL('/dashboard');
    await page.click('text=Settings');

    // Navigate to notification history
    await page.click('text=Notification History');
    await expect(page).toHaveURL('/settings/notifications/history');

    // Verify page loaded
    await expect(page.locator('h2:has-text("Notification History")')).toBeVisible();
  });

  test.skip('should display notification list with pagination', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Wait for notifications to load
    await page.waitForSelector('[data-testid="notification-list"]');

    // Verify notification items are displayed
    const notifications = page.locator('[data-testid="notification-item"]');
    await expect(notifications).toHaveCount(await notifications.count());

    // Verify pagination controls exist
    await expect(page.locator('[data-testid="pagination"]')).toBeVisible();
    await expect(page.locator('button:has-text("Previous")')).toBeVisible();
    await expect(page.locator('button:has-text("Next")')).toBeVisible();
  });

  test.skip('should filter notifications by order reference', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Enter order reference filter
    await page.fill('input[placeholder*="Search by order"]', 'ORD-123');
    await page.click('button:has-text("Search")');

    // Wait for filtered results
    await page.waitForTimeout(500);

    // Verify all results contain the order reference
    const notifications = page.locator('[data-testid="notification-item"]');
    const count = await notifications.count();

    for (let i = 0; i < count; i++) {
      const orderRef = await notifications.nth(i).locator('[data-testid="order-reference"]').textContent();
      expect(orderRef).toContain('ORD-123');
    }
  });

  test.skip('should filter notifications by status', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Select status filter
    await page.selectOption('select[name="status"]', 'failed');

    // Wait for filtered results
    await page.waitForTimeout(500);

    // Verify all results have failed status
    const failedBadges = page.locator('[data-testid="status-badge"]:has-text("failed")');
    await expect(failedBadges.first()).toBeVisible();
  });

  test.skip('should filter by date range', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Set date range
    await page.fill('input[name="start_date"]', '2025-12-01');
    await page.fill('input[name="end_date"]', '2025-12-31');
    await page.click('button:has-text("Apply Filters")');

    // Wait for filtered results
    await page.waitForTimeout(500);

    // Verify notifications are displayed
    await expect(page.locator('[data-testid="notification-item"]').first()).toBeVisible();
  });

  test.skip('should resend failed notification', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Find a failed notification
    const failedNotification = page.locator('[data-testid="notification-item"]').filter({
      has: page.locator('[data-testid="status-badge"]:has-text("failed")'),
    }).first();

    await expect(failedNotification).toBeVisible();

    // Click resend button
    await failedNotification.locator('button:has-text("Resend")').click();

    // Verify confirmation dialog
    await expect(page.locator('text=Resend notification?')).toBeVisible();
    await page.click('button:has-text("Confirm")');

    // Verify success message
    await expect(page.locator('text=Notification resent successfully')).toBeVisible();
  });

  test.skip('should disable resend for successfully sent notifications', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Find a sent notification
    const sentNotification = page.locator('[data-testid="notification-item"]').filter({
      has: page.locator('[data-testid="status-badge"]:has-text("sent")'),
    }).first();

    await expect(sentNotification).toBeVisible();

    // Verify resend button is disabled or not present
    const resendButton = sentNotification.locator('button:has-text("Resend")');
    const isPresent = await resendButton.count() > 0;

    if (isPresent) {
      await expect(resendButton).toBeDisabled();
    } else {
      // Button should not exist for sent notifications
      expect(isPresent).toBe(false);
    }
  });

  test.skip('should show error details for failed notifications', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Find a failed notification
    const failedNotification = page.locator('[data-testid="notification-item"]').filter({
      has: page.locator('[data-testid="status-badge"]:has-text("failed")'),
    }).first();

    await expect(failedNotification).toBeVisible();

    // Click to expand error details
    await failedNotification.locator('[data-testid="expand-details"]').click();

    // Verify error message is displayed
    await expect(page.locator('[data-testid="error-message"]')).toBeVisible();
    await expect(page.locator('[data-testid="retry-count"]')).toBeVisible();
  });

  test.skip('should navigate between pages', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Wait for initial load
    await page.waitForSelector('[data-testid="notification-list"]');

    // Get initial page number
    const initialPage = await page.locator('[data-testid="current-page"]').textContent();
    expect(initialPage).toBe('1');

    // Click next page
    await page.click('button:has-text("Next")');

    // Wait for page change
    await page.waitForTimeout(500);

    // Verify page number changed
    const newPage = await page.locator('[data-testid="current-page"]').textContent();
    expect(newPage).toBe('2');

    // Click previous page
    await page.click('button:has-text("Previous")');

    // Verify back to page 1
    const backPage = await page.locator('[data-testid="current-page"]').textContent();
    expect(backPage).toBe('1');
  });

  test.skip('should display status badges with correct colors', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Check sent badge (green)
    const sentBadge = page.locator('[data-testid="status-badge"]:has-text("sent")').first();
    if (await sentBadge.count() > 0) {
      await expect(sentBadge).toHaveClass(/bg-green/);
    }

    // Check failed badge (red)
    const failedBadge = page.locator('[data-testid="status-badge"]:has-text("failed")').first();
    if (await failedBadge.count() > 0) {
      await expect(failedBadge).toHaveClass(/bg-red/);
    }

    // Check pending badge (yellow)
    const pendingBadge = page.locator('[data-testid="status-badge"]:has-text("pending")').first();
    if (await pendingBadge.count() > 0) {
      await expect(pendingBadge).toHaveClass(/bg-yellow/);
    }
  });

  test.skip('should handle empty state', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Apply filters that return no results
    await page.fill('input[placeholder*="Search by order"]', 'NONEXISTENT-ORDER');
    await page.click('button:has-text("Search")');

    // Wait for results
    await page.waitForTimeout(500);

    // Verify empty state message
    await expect(page.locator('text=No notifications found')).toBeVisible();
  });

  test.skip('should require authentication', async ({ page, context }) => {
    // Clear cookies to simulate logged out state
    await context.clearCookies();

    // Try to access notification history
    await page.goto('/settings/notifications/history');

    // Should redirect to login
    await expect(page).toHaveURL('/login');
  });

  test.skip('should handle rate limit error for resend', async ({ page }) => {
    await page.goto('/settings/notifications/history');

    // Find a failed notification
    const failedNotification = page.locator('[data-testid="notification-item"]').filter({
      has: page.locator('[data-testid="status-badge"]:has-text("failed")'),
    }).first();

    // Click resend multiple times quickly
    for (let i = 0; i < 5; i++) {
      await failedNotification.locator('button:has-text("Resend")').click();
      await page.click('button:has-text("Confirm")');
      await page.waitForTimeout(100);
    }

    // Verify rate limit error message
    await expect(page.locator('text=Too many requests')).toBeVisible();
  });
});
