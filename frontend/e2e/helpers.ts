import { type Page } from '@playwright/test';

const API_URL = 'http://localhost:7244';

/** Generate a unique email for test isolation */
export function testEmail(): string {
	return `test_${Date.now()}_${Math.random().toString(36).slice(2, 8)}@test.local`;
}

/** Register a user via API and return credentials */
export async function registerUser(page: Page) {
	const email = testEmail();
	const password = 'TestPass123!';
	const displayName = 'E2E User';

	const res = await page.request.post(`${API_URL}/api/auth/register`, {
		data: { email, password, displayName },
	});

	if (!res.ok()) {
		throw new Error(`Registration failed: ${res.status()} ${await res.text()}`);
	}

	return { email, password, displayName };
}

/** Login via the UI */
export async function loginViaUI(page: Page, email: string, password: string) {
	await page.goto('/login');
	await page.fill('input[type="email"]', email);
	await page.fill('input[type="password"]', password);
	await page.click('button[type="submit"]');
	await page.waitForURL('**/dashboard**');
}

/** Register + login, returning credentials */
export async function setupAuthenticatedUser(page: Page) {
	const creds = await registerUser(page);
	await loginViaUI(page, creds.email, creds.password);
	return creds;
}
