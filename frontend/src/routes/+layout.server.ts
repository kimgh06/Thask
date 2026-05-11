import type { LayoutServerLoad } from './$types';
import type { User } from '$lib/types';

export const load: LayoutServerLoad = async ({ fetch }) => {
	try {
		const res = await fetch('/api/auth/me');
		if (res.ok) {
			const user: User = await res.json();
			return { user };
		}
	} catch {
		// network error or parse error — treat as unauthenticated
	}
	return { user: null };
};
