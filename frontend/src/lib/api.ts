const API_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:7244';

interface ApiResponse<T> {
	data?: T;
	error?: string;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
	try {
		const res = await fetch(`${API_URL}${path}`, {
			...options,
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json',
				...options.headers,
			},
		});

		const json = await res.json().catch(() => ({}));

		if (!res.ok) {
			return { error: json.error || `HTTP ${res.status}` };
		}

		return json;
	} catch {
		return { error: 'Network error' };
	}
}

export const api = {
	get: <T>(path: string) => request<T>(path),
	post: <T>(path: string, body: unknown) =>
		request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
	patch: <T>(path: string, body: unknown) =>
		request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
	delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
};
