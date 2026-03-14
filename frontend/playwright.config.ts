import { defineConfig } from '@playwright/test';

export default defineConfig({
	testDir: './e2e',
	timeout: 30_000,
	expect: { timeout: 5_000 },
	fullyParallel: false,
	retries: 0,
	workers: 1,
	reporter: 'list',
	use: {
		baseURL: 'http://localhost:7243',
		trace: 'on-first-retry',
		screenshot: 'only-on-failure',
	},
	projects: [
		{
			name: 'chromium',
			use: { browserName: 'chromium' },
		},
	],
	webServer: {
		command: 'npm run build && node build',
		port: 7243,
		reuseExistingServer: true,
		timeout: 30_000,
		env: {
			PUBLIC_API_URL: 'http://localhost:7244',
			ORIGIN: 'http://localhost:7243',
		},
	},
});
