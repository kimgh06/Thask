import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND_URL = env.BACKEND_URL || 'http://localhost:7244';

export const GET: RequestHandler = async ({ request, params }) => {
	const shareToken = params.shareToken;
	const path = `/api/shared/${shareToken}/events`;

	const headers = new Headers();
	const cookie = request.headers.get('cookie');
	if (cookie) headers.set('cookie', cookie);

	const backendRes = await fetch(`${BACKEND_URL}${path}`, {
		method: 'GET',
		headers,
		signal: request.signal,
	});

	return new Response(backendRes.body, {
		status: backendRes.status,
		headers: {
			'Content-Type': 'text/event-stream',
			'Cache-Control': 'no-cache',
			'Connection': 'keep-alive',
			'X-Accel-Buffering': 'no',
		},
	});
};
