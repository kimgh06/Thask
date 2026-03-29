import { browser } from '$app/environment';

type Theme = 'dark' | 'light';

class ThemeStore {
	current = $state<Theme>('dark');

	constructor() {
		if (browser) {
			const saved = localStorage.getItem('thask-theme') as Theme | null;
			this.current = saved || 'dark';
			this.apply();
		}
	}

	toggle() {
		this.current = this.current === 'dark' ? 'light' : 'dark';
		this.apply();
	}

	private apply() {
		if (!browser) return;
		document.documentElement.setAttribute('data-theme', this.current);
		localStorage.setItem('thask-theme', this.current);
	}
}

export const themeStore = new ThemeStore();
