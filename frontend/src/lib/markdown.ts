import { marked } from 'marked';
import { browser } from '$app/environment';

const ALLOWED_TAGS = [
	'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'p', 'br', 'hr',
	'strong', 'em', 'del', 'code', 'pre',
	'ul', 'ol', 'li',
	'a', 'blockquote',
	'table', 'thead', 'tbody', 'tr', 'th', 'td',
];

const ALLOWED_ATTR = ['href', 'title', 'target', 'rel'];

let purifyReady = false;
let DOMPurify: typeof import('dompurify').default;

async function ensurePurify() {
	if (purifyReady) return;
	if (!browser) return;
	const mod = await import('dompurify');
	DOMPurify = mod.default;
	DOMPurify.addHook('afterSanitizeAttributes', (node) => {
		if (node.tagName === 'A') {
			node.setAttribute('target', '_blank');
			node.setAttribute('rel', 'noopener noreferrer');
		}
	});
	purifyReady = true;
}

// Eagerly init on import (browser only)
if (browser) {
	ensurePurify();
}

export function renderMarkdown(source: string): string {
	const rawHtml = marked.parse(source, { async: false, breaks: true, gfm: true }) as string;
	if (!browser || !purifyReady || !DOMPurify) {
		// SSR fallback: strip all HTML tags for safety
		return rawHtml.replace(/<[^>]*>/g, '');
	}
	return DOMPurify.sanitize(rawHtml, {
		ALLOWED_TAGS,
		ALLOWED_ATTR,
	});
}
