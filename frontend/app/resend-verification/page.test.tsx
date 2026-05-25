import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import ResendVerificationPage from './page';
import authService from '@/services/auth';

jest.mock('next/navigation', () => ({
  useSearchParams: () => new URLSearchParams(),
}));

jest.mock('@/components/layout/PublicLayout', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

jest.mock('@/i18n/provider', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const messages: Record<string, string> = {
        'auth.resendVerification.title': 'Resend Verification Email',
        'auth.resendVerification.instructions': 'Enter the email you used to register your business account.',
        'auth.resendVerification.email': 'Email',
        'auth.resendVerification.emailPlaceholder': 'you@example.com',
        'auth.resendVerification.submit': 'Send Verification Email',
        'auth.resendVerification.sending': 'Sending...',
        'auth.resendVerification.successMessage': 'If an unverified account exists, a verification email has been sent.',
        'auth.resendVerification.error': 'Failed to resend verification email. Please try again.',
        'auth.resendVerification.emailRequired': 'Email is required',
        'auth.resendVerification.emailInvalid': 'Invalid email format',
        'auth.resendVerification.backToLogin': 'Back to Login',
        'auth.resendVerification.createAccount': 'Create a new account',
      };
      return messages[key] || key;
    },
  }),
}));

jest.mock('@/services/auth', () => ({
  __esModule: true,
  default: {
    resendVerificationEmail: jest.fn(),
  },
}));

describe('ResendVerificationPage', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('validates the email before submitting', () => {
    render(<ResendVerificationPage />);

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'invalid-email' },
    });
    fireEvent.submit(screen.getByRole('button', { name: /send verification email/i }).closest('form')!);

    expect(screen.getByText('Invalid email format')).toBeInTheDocument();
    expect(authService.resendVerificationEmail).not.toHaveBeenCalled();
  });

  it('submits the normalized email and shows generic success', async () => {
    jest.mocked(authService.resendVerificationEmail).mockResolvedValue();
    render(<ResendVerificationPage />);

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: ' Owner@Example.COM ' },
    });
    fireEvent.submit(screen.getByRole('button', { name: /send verification email/i }).closest('form')!);

    await waitFor(() => {
      expect(authService.resendVerificationEmail).toHaveBeenCalledWith('owner@example.com');
    });
    expect(await screen.findByText('If an unverified account exists, a verification email has been sent.')).toBeInTheDocument();
  });
});
