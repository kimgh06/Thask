import { test, expect } from '@playwright/test';
import { testEmail, loginViaUI } from './helpers';

const API_URL = 'http://localhost:7244';

test.describe('Authentication', () => {
	test('redirects unauthenticated users to login', async ({ page }) => {
		await page.goto('/dashboard');
		await expect(page).toHaveURL(/\/login/);
	});

	test('registers a new user', async ({ page }) => {
		const email = testEmail();

		await page.goto('/register');
		await page.fill('input[type="email"]', email);
		await page.fill('input[type="password"]', 'TestPass123!');
		await page.getByPlaceholder(/name/i).fill('E2E Tester');
		await page.click('button[type="submit"]');

		await page.waitForURL('**/dashboard**');
		await expect(page.locator('text=E2E Tester')).toBeVisible();
	});

	test('logs in with valid credentials', async ({ page }) => {
		const email = testEmail();
		const password = 'TestPass123!';

		// Register via API
		await page.request.post(`${API_URL}/api/auth/register`, {
			data: { email, password, displayName: 'Login Tester' },
		});

		await loginViaUI(page, email, password);
		await expect(page.locator('text=Login Tester')).toBeVisible();
	});

	test('shows error for wrong password', async ({ page }) => {
		const email = testEmail();

		await page.request.post(`${API_URL}/api/auth/register`, {
			data: { email, password: 'TestPass123!', displayName: 'Err Tester' },
		});

		await page.goto('/login');
		await page.fill('input[type="email"]', email);
		await page.fill('input[type="password"]', 'wrong-password');
		await page.click('button[type="submit"]');

		await expect(page.locator('text=/invalid|error|incorrect/i')).toBeVisible();
	});

	test('logs out successfully', async ({ page }) => {
		const email = testEmail();
		const password = 'TestPass123!';

		await page.request.post(`${API_URL}/api/auth/register`, {
			data: { email, password, displayName: 'Logout Tester' },
		});

		await loginViaUI(page, email, password);
		await page.click('button:has-text("Logout")');
		await expect(page).toHaveURL(/\/login/);
	});
});
