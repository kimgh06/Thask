const BASE = 'https://thask.kimgh06.com';

const paths = [
	'/',
	'/login',
	'/register',
	'/docs',
	'/dashboard',
	'/dashboard/settings'
];

export const prerender = true;

export const GET = async () => {
	const now = new Date().toISOString().slice(0, 10);
	const urls = paths
		.map(
			(p) =>
				`  <url><loc>${BASE}${p}</loc><lastmod>${now}</lastmod><changefreq>weekly</changefreq><priority>${p === '/' ? '1.0' : '0.7'}</priority></url>`
		)
		.join('\n');
	const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls}
</urlset>`;
	return new Response(xml, {
		headers: { 'Content-Type': 'application/xml; charset=utf-8' }
	});
};
