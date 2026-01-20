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

export interface InvitationRequest {
  email: string;
  role: string;
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

  async acceptInvitation(token: string, data: AcceptInvitationRequest): Promise<any> {
    return apiClient.post(`/api/invitations/${token}/accept`, data);
  }
}

export default new UserService();
