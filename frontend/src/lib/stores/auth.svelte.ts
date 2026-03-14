import { api } from '$lib/api';
import type { User } from '$lib/types';

class AuthStore {
	user = $state<User | null>(null);
	loading = $state(true);

	get isAuthenticated() {
		return this.user !== null;
	}

	async fetchUser() {
		this.loading = true;
		const res = await api.get<User>('/api/auth/me');
		this.user = res.data ?? null;
		this.loading = false;
	}

	async login(email: string, password: string): Promise<string | null> {
		const res = await api.post<User>('/api/auth/login', { email, password });
		if (res.error) return res.error;
		this.user = res.data ?? null;
		return null;
	}

	async register(email: string, password: string, displayName: string): Promise<string | null> {
		const res = await api.post<User>('/api/auth/register', { email, password, displayName });
		if (res.error) return res.error;
		this.user = res.data ?? null;
		return null;
	}

	async logout() {
		await api.post('/api/auth/logout', {});
		this.user = null;
	}
}

export const authStore = new AuthStore();
