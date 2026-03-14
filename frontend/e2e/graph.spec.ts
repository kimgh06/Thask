import { test, expect } from '@playwright/test';
import { setupAuthenticatedUser } from './helpers';

const API_URL = 'http://localhost:7244';

/** Helper to create a team + project and navigate to the graph editor */
async function navigateToGraph(page: import('@playwright/test').Page) {
	await setupAuthenticatedUser(page);

	const teamName = `GTeam ${Date.now()}`;
	await page.getByPlaceholder(/team name/i).fill(teamName);
	await page.click('button:has-text("Create")');
	await page.click(`text=${teamName}`);
	await page.waitForURL(/\/dashboard\/.+/);

	const projectName = `GProj ${Date.now()}`;
	await page.getByPlaceholder(/project name/i).fill(projectName);
	await page.click('button:has-text("Create")');
	await page.click(`text=${projectName}`);
	await page.waitForURL(/\/dashboard\/.+\/.+/);

	await expect(page.locator('button:has-text("+ Node")')).toBeVisible();
}

test.describe('Graph Editor', () => {
	test('opens Add Node modal and creates a node', async ({ page }) => {
		await navigateToGraph(page);

		await page.click('button:has-text("+ Node")');
		await expect(page.locator('text=Add Node')).toBeVisible();

		await page.fill('#node-title', 'Login Flow');
		await page.click('button:has-text("FLOW")');
		await page.click('button:has-text("Create Node")');

		// Modal should close, node count should update
		await expect(page.locator('text=Add Node')).not.toBeVisible();
		await expect(page.locator('text=/1 nodes/i')).toBeVisible();
	});

	test('creates a group node via toolbar', async ({ page }) => {
		await navigateToGraph(page);

		await page.click('button:has-text("+ Group")');

		// Should see 1 node (the group)
		await expect(page.locator('text=/1 nodes/i')).toBeVisible();
	});

	test('toolbar buttons are functional', async ({ page }) => {
		await navigateToGraph(page);

		// Verify toolbar controls exist
		await expect(page.locator('button[title="Zoom In"]')).toBeVisible();
		await expect(page.locator('button[title="Zoom Out"]')).toBeVisible();
		await expect(page.locator('button[title="Fit View"]')).toBeVisible();
		await expect(page.locator('button[title="Auto Layout"]')).toBeVisible();

		// Toggle filters
		await page.click('button:has-text("Filter")');
		await expect(page.locator('text=Type:')).toBeVisible();
		await expect(page.locator('text=Status:')).toBeVisible();

		// Close filters
		await page.click('button:has-text("Filter")');
		await expect(page.locator('text=Type:')).not.toBeVisible();
	});

	test('search opens with Ctrl+F', async ({ page }) => {
		await navigateToGraph(page);

		await page.keyboard.press('Control+f');
		await expect(page.locator('input[placeholder="Search nodes..."]')).toBeVisible();

		// Close with Escape
		await page.keyboard.press('Escape');
		await expect(page.locator('input[placeholder="Search nodes..."]')).not.toBeVisible();
	});

	test('impact mode toggles', async ({ page }) => {
		await navigateToGraph(page);

		const impactBtn = page.locator('button:has-text("Impact")');
		await expect(impactBtn).toBeVisible();

		// Toggle on
		await impactBtn.click();
		// The button should change background color (amber)
		await expect(impactBtn).toHaveCSS('background-color', 'rgb(245, 158, 11)');

		// Toggle off
		await impactBtn.click();
		await expect(impactBtn).not.toHaveCSS('background-color', 'rgb(245, 158, 11)');
	});
});
