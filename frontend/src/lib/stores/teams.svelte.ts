import { api } from '$lib/api';
import type { Team } from '$lib/types';

class TeamsStore {
	teams = $state<Team[]>([]);
	error = $state('');

	async load() {
		this.error = '';
		const res = await api.get<Team[]>('/api/teams');
		if (res.data) { this.teams = res.data; }
		else { this.error = 'Failed to load teams.'; }
	}
}

export const teamsStore = new TeamsStore();
