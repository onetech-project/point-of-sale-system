import apiClient from './api';

export interface RegisterTenantData {
  businessName: string;
  email: string;
  password: string;
  ownerProfile?: {
    firstName?: string;
    lastName?: string;
  };
  consents?: string[];
}

export interface LoginCredentials {
  email: string;
  password: string;
}

export interface UserSession {
  user: {
    id: string;
    email: string;
    firstName?: string;
    lastName?: string;
    role: string;
  };
  tenantId: string;
  token: string;
}

export const authService = {
  async registerTenant(data: RegisterTenantData): Promise<any> {
    try {
      const response = await apiClient.post('/api/tenants/register', {
        business_name: data.businessName,
        email: data.email,
        password: data.password,
        first_name: data.ownerProfile?.firstName || '',
        last_name: data.ownerProfile?.lastName || '',
        consents: data.consents || [],
      });

      return response;
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Failed to register tenant. Please try again.');
    }
  },

  async login(credentials: LoginCredentials): Promise<UserSession> {
    try {
      const response = await apiClient.post<UserSession>('/api/auth/login', {
        email: credentials.email,
        password: credentials.password,
      });

      // Token is stored in HTTP-only cookie by backend, no need to store manually
      // Just store user info in localStorage for quick access
      if (typeof window !== 'undefined' && response.user) {
        localStorage.setItem('user', JSON.stringify(response.user));
      }

      return response;
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Failed to login. Please try again.');
    }
  },

  async getSession(): Promise<UserSession | null> {
    try {
      const response = await apiClient.get<UserSession>('/api/auth/session');
      return response;
    } catch (error) {
      return null;
    }
  },

  async logout(): Promise<void> {
    try {
      await apiClient.post('/api/auth/logout');
      // Clear auth data from API client and localStorage
      apiClient.clearAuth();
      if (typeof window !== 'undefined') {
        localStorage.removeItem('user');
      }
    } catch (error) {
      console.error('Logout error:', error);
      // Clear auth data even if logout request fails
      apiClient.clearAuth();
      if (typeof window !== 'undefined') {
        localStorage.removeItem('user');
      }
      throw new Error('Failed to logout');
    }
  },

  async validateSession(): Promise<boolean> {
    try {
      const session = await this.getSession();
      return !!session;
    } catch (error) {
      return false;
    }
  },

  async requestPasswordReset(email: string): Promise<void> {
    try {
      await apiClient.post('/api/auth/password-reset/request', { email });
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Failed to request password reset. Please try again.');
    }
  },

  async resetPassword(token: string, newPassword: string): Promise<void> {
    try {
      await apiClient.post('/api/auth/password-reset/reset', {
        token,
        new_password: newPassword,
      });
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Failed to reset password. Please try again.');
    }
  },

  async verifyAccount(token: string): Promise<void> {
    try {
      await apiClient.post('/api/auth/verify-account', { token });
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Failed to verify account. Please try again.');
    }
  },
}

export default authService;
