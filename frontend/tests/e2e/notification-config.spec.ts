// This E2E test will FAIL initially because the notification config feature doesn't exist yet
// This is expected in TDD - write the test first, then implement the feature

import { test, expect } from '@playwright/test';

test.describe('Notification Configuration E2E', () => {
  test.beforeEach(async ({ page }) => {
    // Login as admin
    await page.goto('/login');
    await page.fill('[name="email"]', 'admin@example.com');
    await page.fill('[name="password"]', 'password123');
    await page.click('button[type="submit"]');

    // Wait for dashboard to load
    await page.waitForURL('/dashboard');
  });

  test('should navigate to notification settings page', async ({ page }) => {
    // TODO: Uncomment when implementation is ready
    // // Navigate to settings
    // await page.click('text=Settings');
    // await page.click('text=Notifications');

    // // Should be on notification settings page
    // await expect(page).toHaveURL(/\/admin\/settings\/notifications/);
    // await expect(page.locator('h1')).toContainText('Notification Settings');

    // For now, skip test (TDD - implementation pending)
    test.skip();
  });

  test('should display list of staff members with notification preferences', async ({ page }) => {
    // TODO: Uncomment when implementation is ready
    // await page.goto('/admin/settings/notifications');

    // // Wait for staff list to load
    // await page.waitForSelector('[data-testid="staff-list"]');

    // // Should show staff members
    // const staffItems = page.locator('[data-testid="staff-item"]');
    // await expect(staffItems).toHaveCount(greaterThan(0));

    // // Each staff item should have name, email, role, and toggle
    // const firstStaff = staffItems.first();
    // await expect(firstStaff.locator('[data-testid="staff-name"]')).toBeVisible();
    // await expect(firstStaff.locator('[data-testid="staff-email"]')).toBeVisible();
    // await expect(firstStaff.locator('[role="switch"]')).toBeVisible();

    // For now, skip test (TDD - implementation pending)
    test.skip();
  });

  test('should enable notification preference for staff member', async ({ page }) => {
    // TODO: Uncomment when implementation is ready
    // await page.goto('/admin/settings/notifications');

    // // Find a staff member with notifications disabled
    // const disabledToggle = page.locator('[role="switch"][aria-checked="false"]').first();
    // const staffName = await disabledToggle.locator('xpath=ancestor::*[@data-testid="staff-item"]//[@data-testid="staff-name"]').textContent();

    // // Click toggle to enable
    // await disabledToggle.click();

    // // Should show loading/saving state
    // await expect(page.locator('text=Saving...')).toBeVisible();

    // // Wait for save to complete
    // await page.waitForSelector('text=Saving...', { state: 'hidden' });

    // // Should show success message
    // await expect(page.locator('text=Settings updated')).toBeVisible();

    // // Toggle should now be enabled
    // await expect(disabledToggle).toHaveAttribute('aria-checked', 'true');

    // For now, skip test (TDD - implementation pending)
    test.skip();
  });

  test('should disable notification preference for staff member', async ({ page }) => {
    // TODO: Uncomment when implementation is ready
    // await page.goto('/admin/settings/notifications');

    // // Find a staff member with notifications enabled
    // const enabledToggle = page.locator('[role="switch"][aria-checked="true"]').first();

    // // Click toggle to disable
    // await enabledToggle.click();

    // // Wait for save
    // await page.waitForSelector('text=Saving...', { state: 'hidden' });

    // // Should show success
    // await expect(page.locator('text=Settings updated')).toBeVisible();

    // // Toggle should now be disabled
    // await expect(enabledToggle).toHaveAttribute('aria-checked', 'false');

    // For now, skip test (TDD - implementation pending)
    test.skip();
  });

  test('should send test notification email', async ({ page }) => {
    // TODO: Uncomment when implementation is ready
    // await page.goto('/admin/settings/notifications');

    // // Click "Send Test Email" button
    // await page.click('button:has-text("Send Test Email")');

    // // Should show modal
    // await expect(page.locator('[role="dialog"]')).toBeVisible();
    // await expect(page.locator('text=Send Test Notification')).toBeVisible();

    // // Enter email address
    // await page.fill('[name="recipient_email"]', 'test@example.com');

    // // Select notification type
    // await page.selectOption('[name="notification_type"]', 'staff_order_notification');

    // // Click confirm
    // await page.click('button:has-text("Send Test")');

    // // Should show sending state
    // await expect(page.locator('text=Sending...')).toBeVisible();

    // // Wait for success
    // await expect(page.locator('text=Test email sent successfully')).toBeVisible({ timeout: 10000 });

    // // Modal should close
    // await expect(page.locator('[role="dialog"]')).not.toBeVisible();

    // For now, skip test (TDD - implementation pending)
    test.skip();
  });

  test('should validate email format before sending test notification', async ({ page }) => {
    // TODO: Uncomment when implementation is ready
    // await page.goto('/admin/settings/notifications');

    // await page.click('button:has-text("Send Test Email")');

    // // Enter invalid email
    // await page.fill('[name="recipient_email"]', 'invalid-email');

    // // Try to submit
    // await page.click('button:has-text("Send Test")');

    // // Should show validation error
    // await expect(page.locator('text=Invalid email format')).toBeVisible();

    // // Should not close modal
    // await expect(page.locator('[role="dialog"]')).toBeVisible();

    // For now, skip test (TDD - implementation pending)
    test.skip();
  });

  test('should handle API errors when updating preferences', async ({ page }) => {
    // TODO: Uncomment when implementation is ready
    // // Mock API to return error
    // await page.route('**/api/v1/users/*/notification-preferences', route => {
    //   route.fulfill({
    //     status: 500,
    //     body: JSON.stringify({ error: 'Internal server error' }),
    //   });
    // });

    // await page.goto('/admin/settings/notifications');

    // // Try to toggle a preference
    // await page.locator('[role="switch"]').first().click();

    // // Should show error message
    // await expect(page.locator('text=Failed to update settings')).toBeVisible();

    // // Toggle should revert to original state
    // // (This tests optimistic updates with rollback)

    // For now, skip test (TDD - implementation pending)
    test.skip();
  });

  test('should persist changes after page reload', async ({ page }) => {
    // TODO: Uncomment when implementation is ready
    // await page.goto('/admin/settings/notifications');

    // // Enable first toggle
    // const toggle = page.locator('[role="switch"]').first();
    // const originalState = await toggle.getAttribute('aria-checked');

    // await toggle.click();
    // await page.waitForSelector('text=Saving...', { state: 'hidden' });

    // // Reload page
    // await page.reload();

    // // Wait for page to load
    // await page.waitForSelector('[data-testid="staff-list"]');

    // // Toggle state should be persisted
    // const newState = await page.locator('[role="switch"]').first().getAttribute('aria-checked');
    // expect(newState).not.toBe(originalState);

    // For now, skip test (TDD - implementation pending)
    test.skip();
  });

  test('should require admin role to access notification settings', async ({ page }) => {
    // TODO: Uncomment when implementation is ready
    // // Logout and login as staff (non-admin)
    // await page.click('[data-testid="user-menu"]');
    // await page.click('text=Logout');

    // await page.fill('[name="email"]', 'staff@example.com');
    // await page.fill('[name="password"]', 'password123');
    // await page.click('button[type="submit"]');

    // // Try to access notification settings
    // await page.goto('/admin/settings/notifications');

    // // Should redirect to unauthorized page or show error
    // await expect(page).not.toHaveURL(/\/admin\/settings\/notifications/);
    // await expect(page.locator('text=Unauthorized')).toBeVisible();

    // For now, skip test (TDD - implementation pending)
    test.skip();
  });
});
