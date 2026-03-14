import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const BACKEND_URL = env.BACKEND_URL || 'http://localhost:7244';

const handler: RequestHandler = async ({ request, params, url }) => {
	const path = `/api/${params.path}${url.search}`;
	const backendRes = await fetch(`${BACKEND_URL}${path}`, {
		method: request.method,
		headers: request.headers,
		body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : undefined,
		// @ts-expect-error -- duplex required for streaming request bodies
		duplex: 'half'
	});

	return new Response(backendRes.body, {
		status: backendRes.status,
		headers: backendRes.headers
	});
};

export const GET = handler;
export const POST = handler;
export const PATCH = handler;
export const DELETE = handler;
export const OPTIONS = handler;
