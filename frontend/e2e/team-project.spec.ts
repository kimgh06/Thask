import { test, expect } from '@playwright/test';
import { setupAuthenticatedUser } from './helpers';

test.describe('Team & Project Management', () => {
	test('creates a new team', async ({ page }) => {
		await setupAuthenticatedUser(page);

		const teamName = `Team ${Date.now()}`;
		await page.getByPlaceholder(/team name/i).fill(teamName);
		await page.click('button:has-text("Create")');

		await expect(page.locator(`text=${teamName}`)).toBeVisible();
	});

	test('navigates to team page and creates a project', async ({ page }) => {
		await setupAuthenticatedUser(page);

		// Create team
		const teamName = `ProjTeam ${Date.now()}`;
		await page.getByPlaceholder(/team name/i).fill(teamName);
		await page.click('button:has-text("Create")');

		// Navigate to team
		await page.click(`text=${teamName}`);
		await page.waitForURL(/\/dashboard\/.+/);

		// Create project
		const projectName = `Project ${Date.now()}`;
		await page.getByPlaceholder(/project name/i).fill(projectName);
		await page.click('button:has-text("Create")');

		await expect(page.locator(`text=${projectName}`)).toBeVisible();
	});

	test('navigates to project graph editor', async ({ page }) => {
		await setupAuthenticatedUser(page);

		// Create team
		const teamName = `GraphTeam ${Date.now()}`;
		await page.getByPlaceholder(/team name/i).fill(teamName);
		await page.click('button:has-text("Create")');

		// Navigate to team
		await page.click(`text=${teamName}`);
		await page.waitForURL(/\/dashboard\/.+/);

		// Create project
		const projectName = `GraphProj ${Date.now()}`;
		await page.getByPlaceholder(/project name/i).fill(projectName);
		await page.click('button:has-text("Create")');

		// Navigate to project graph
		await page.click(`text=${projectName}`);
		await page.waitForURL(/\/dashboard\/.+\/.+/);

		// Should see the graph toolbar
		await expect(page.locator('button:has-text("+ Node")')).toBeVisible();
		await expect(page.locator('text=/\\d+ nodes/i')).toBeVisible();
	});
});
