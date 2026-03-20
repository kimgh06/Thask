import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND_URL = env.BACKEND_URL || 'http://localhost:7244';

export const GET: RequestHandler = async ({ request, params }) => {
	const projectId = params.projectId;
	const path = `/api/projects/${projectId}/events`;

	const headers = new Headers();
	const cookie = request.headers.get('cookie');
	if (cookie) headers.set('cookie', cookie);
	const auth = request.headers.get('authorization');
	if (auth) headers.set('authorization', auth);

	const backendRes = await fetch(`${BACKEND_URL}${path}`, {
		method: 'GET',
		headers,
		signal: request.signal,
	});

	// Stream SSE directly without buffering
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
