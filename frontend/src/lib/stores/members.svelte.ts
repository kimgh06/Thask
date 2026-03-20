import { api } from '$lib/api';
import type { TeamMember, TeamRole } from '$lib/types';

class MembersStore {
	members = $state<TeamMember[]>([]);
	loading = $state(false);
	error = $state('');

	get myRole(): TeamRole | null {
		// Will be set externally based on current user
		return this._myRole;
	}

	private _myRole = $state<TeamRole | null>(null);

	setMyRole(role: TeamRole | null) {
		this._myRole = role;
	}

	async load(teamSlug: string) {
		this.loading = true;
		this.error = '';
		const res = await api.get<TeamMember[]>(`/api/teams/${teamSlug}/members`);
		if (res.data) {
			this.members = res.data;
		} else {
			this.error = res.error ?? 'Failed to load members';
		}
		this.loading = false;
	}

	async invite(teamSlug: string, email: string, role: TeamRole) {
		const res = await api.post(`/api/teams/${teamSlug}/members`, { email, role });
		if (res.error) return res.error;
		await this.load(teamSlug);
		return null;
	}

	async updateRole(teamSlug: string, userId: string, role: TeamRole) {
		const res = await api.patch(`/api/teams/${teamSlug}/members/${userId}`, { role });
		if (res.error) return res.error;
		await this.load(teamSlug);
		return null;
	}

	async remove(teamSlug: string, userId: string) {
		const res = await api.delete(`/api/teams/${teamSlug}/members/${userId}`);
		if (res.error) return res.error;
		await this.load(teamSlug);
		return null;
	}

	async leave(teamSlug: string) {
		const res = await api.post(`/api/teams/${teamSlug}/leave`, {});
		if (res.error) return res.error;
		return null;
	}

	async transfer(teamSlug: string, targetUserId: string) {
		const res = await api.post(`/api/teams/${teamSlug}/transfer`, { userId: targetUserId });
		if (res.error) return res.error;
		await this.load(teamSlug);
		return null;
	}
}

export const membersStore = new MembersStore();
