import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, fetch }) => {
	try {
		const res = await fetch(`/api/shared/${params.shareToken}`);
		if (!res.ok) return { project: null };
		const json = await res.json();
		return { project: json.data ?? null };
	} catch {
		return { project: null };
	}
};
