import apiClient from './api';

export interface Invitation {
  id: string;
  email: string;
  role: string;
  status: string;
  expiresAt: string;
  invitedBy: string;
  createdAt: string;
}

export interface TeamMember {
  id: string;
  email: string;
  role: 'manager' | 'cashier';
  status: 'active' | 'suspended';
  firstName?: string;
  lastName?: string;
  createdAt: string;
  updatedAt: string;
}

export interface InvitationRequest {
  email: string;
  role: string;
}

export interface TeamMemberUpdateRequest {
  role?: 'manager' | 'cashier';
  status?: 'active' | 'suspended';
}

export interface AcceptInvitationRequest {
  firstName: string;
  lastName: string;
  password: string;
  consents?: string[];
}

class UserService {
  async inviteUser(data: InvitationRequest): Promise<Invitation> {
    return apiClient.post<Invitation>('/api/invitations', data);
  }

  async getInvitations(): Promise<Invitation[]> {
    return apiClient.get<Invitation[]>('/api/invitations');
  }

  async resendInvitation(invitationId: string): Promise<Invitation> {
    return apiClient.post<Invitation>(`/api/invitations/${invitationId}/resend`, {});
  }

  async revokeInvitation(invitationId: string): Promise<Invitation> {
    return apiClient.post<Invitation>(`/api/invitations/${invitationId}/revoke`, {});
  }

  async getTeamMembers(): Promise<TeamMember[]> {
    return apiClient.get<TeamMember[]>('/api/team/users');
  }

  async updateTeamMember(userId: string, data: TeamMemberUpdateRequest): Promise<TeamMember> {
    return apiClient.patch<TeamMember>(`/api/team/users/${userId}`, data);
  }

  async acceptInvitation(token: string, data: AcceptInvitationRequest): Promise<any> {
    return apiClient.post(`/api/invitations/${token}/accept`, data);
  }
}

export default new UserService();
