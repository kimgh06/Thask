/**
 * Returns a human-readable relative time string for an ISO date string.
 * Examples: "3 days ago", "2 months ago", "just now"
 */
export function relativeDate(iso: string): string {
	try {
		const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
		const diffMs = new Date(iso).getTime() - Date.now();
		const diffSec = Math.round(diffMs / 1000);
		const diffMin = Math.round(diffSec / 60);
		const diffHour = Math.round(diffMin / 60);
		const diffDay = Math.round(diffHour / 24);
		const diffMonth = Math.round(diffDay / 30);
		const diffYear = Math.round(diffDay / 365);

		if (Math.abs(diffSec) < 60) return rtf.format(diffSec, 'second');
		if (Math.abs(diffMin) < 60) return rtf.format(diffMin, 'minute');
		if (Math.abs(diffHour) < 24) return rtf.format(diffHour, 'hour');
		if (Math.abs(diffDay) < 30) return rtf.format(diffDay, 'day');
		if (Math.abs(diffMonth) < 12) return rtf.format(diffMonth, 'month');
		return rtf.format(diffYear, 'year');
	} catch {
		return iso;
	}
}
