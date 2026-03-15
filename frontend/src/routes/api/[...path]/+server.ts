import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND_URL = env.BACKEND_URL || 'http://localhost:7244';

const HOP_HEADERS = new Set([
	'transfer-encoding',
	'content-encoding',
	'content-length',
	'connection',
	'keep-alive'
]);

const handler: RequestHandler = async ({ request, params, url }) => {
	const path = `/api/${params.path}${url.search}`;

	const headers = new Headers(request.headers);
	headers.delete('host');

	const backendRes = await fetch(`${BACKEND_URL}${path}`, {
		method: request.method,
		headers,
		body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : undefined,
		// @ts-expect-error -- duplex required for streaming request bodies
		duplex: 'half'
	});

	const responseHeaders = new Headers();
	for (const [key, value] of backendRes.headers) {
		if (!HOP_HEADERS.has(key.toLowerCase())) {
			responseHeaders.set(key, value);
		}
	}

	return new Response(backendRes.body, {
		status: backendRes.status,
		headers: responseHeaders
	});
};

export const GET = handler;
export const POST = handler;
export const PATCH = handler;
export const DELETE = handler;
export const OPTIONS = handler;
