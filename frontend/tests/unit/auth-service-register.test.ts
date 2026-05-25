import { authService } from '@/services/auth';
import apiClient from '@/services/api';

jest.mock('@/services/api', () => ({
  __esModule: true,
  default: {
    post: jest.fn(),
  },
}));

describe('authService.registerTenant', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('sends required Terms acceptance fields during tenant registration', async () => {
    (apiClient.post as jest.Mock).mockResolvedValue({ tenant: { id: 'tenant-1' } });

    await authService.registerTenant({
      businessName: 'Ada Cafe',
      email: 'owner@example.com',
      password: 'Password123!',
      ownerProfile: {
        firstName: 'Ada',
        lastName: 'Owner',
      },
      consents: ['analytics'],
      termsAccepted: true,
      termsVersion: '1.0.0',
    });

    expect(apiClient.post).toHaveBeenCalledWith('/api/tenants/register', {
      business_name: 'Ada Cafe',
      email: 'owner@example.com',
      password: 'Password123!',
      first_name: 'Ada',
      last_name: 'Owner',
      consents: ['analytics'],
      terms_accepted: true,
      terms_version: '1.0.0',
    });
  });

  it('sends verification resend requests with email only', async () => {
    (apiClient.post as jest.Mock).mockResolvedValue({
      message: 'If an unverified account exists, a verification email has been sent.',
    });

    await authService.resendVerificationEmail('owner@example.com');

    expect(apiClient.post).toHaveBeenCalledWith('/api/auth/resend-verification', {
      email: 'owner@example.com',
    });
  });
});
