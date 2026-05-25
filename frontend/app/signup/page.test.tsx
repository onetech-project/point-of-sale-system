import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import SignupPage from './page';
import { authService } from '@/services/auth';

jest.mock('@/components/layout/PublicLayout', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

jest.mock('@/components/consent/ConsentPurposeList', () => ({
  __esModule: true,
  default: ({ onConsentChange }: { onConsentChange: (consents: Record<string, boolean>) => void }) => {
    return (
      <button
        type="button"
        onClick={() => onConsentChange({ operational: true, third_party_midtrans: true })}
      >
        Grant required consents
      </button>
    );
  },
}));

jest.mock('@/i18n/provider', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const messages: Record<string, string> = {
        'auth.signup.title': 'Create Account',
        'auth.signup.subtitle': 'Create your business account',
        'auth.signup.businessName': 'Business Name',
        'auth.signup.businessNamePlaceholder': 'Enter your business name',
        'auth.signup.email': 'Email',
        'auth.signup.emailPlaceholder': 'you@example.com',
        'auth.signup.firstName': 'First Name',
        'auth.signup.lastName': 'Last Name',
        'auth.signup.password': 'Password',
        'auth.signup.confirmPassword': 'Confirm Password',
        'auth.signup.passwordRequirements': 'Password must contain:',
        'auth.signup.passwordMinLength': 'At least 8 characters',
        'auth.signup.passwordLetterNumber': 'Letters and numbers',
        'auth.signup.passwordUpperCase': 'At least one uppercase letter',
        'auth.signup.passwordSpecialChar': 'At least one special character',
        'auth.signup.dataConsent': 'Data Privacy Consent',
        'auth.signup.submit': 'Create Account',
        'auth.signup.submitting': 'Creating account...',
        'auth.signup.haveAccount': 'Already have an account?',
        'auth.signup.signIn': 'Sign in',
        'auth.signup.successTitle': 'Registration Successful!',
        'auth.signup.successMessage': 'We have sent you a verification email.',
        'auth.signup.goToLogin': 'Go to Login',
        'auth.resendVerification.resendButton': 'Resend verification email',
        'auth.resendVerification.sending': 'Sending...',
        'auth.resendVerification.successMessage': 'If an unverified account exists, a verification email has been sent.',
        'auth.resendVerification.error': 'Failed to resend verification email. Please try again.',
      };
      return messages[key] || key;
    },
  }),
}));

jest.mock('@/services/auth', () => ({
  __esModule: true,
  authService: {
    registerTenant: jest.fn(),
    resendVerificationEmail: jest.fn(),
  },
}));

describe('SignupPage', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('keeps the registration success state and exposes verification resend actions', async () => {
    jest.mocked(authService.registerTenant).mockResolvedValue({ tenant: { id: 'tenant-1' } });
    jest.mocked(authService.resendVerificationEmail).mockResolvedValue();

    render(<SignupPage />);

    fireEvent.change(screen.getByLabelText(/business name/i), {
      target: { value: 'Ada Cafe' },
    });
    fireEvent.change(screen.getByLabelText(/^email/i), {
      target: { value: 'Owner@Example.COM' },
    });
    fireEvent.change(screen.getByLabelText(/^password/i), {
      target: { value: 'Password123!' },
    });
    fireEvent.change(screen.getByLabelText(/confirm password/i), {
      target: { value: 'Password123!' },
    });
    fireEvent.click(screen.getByRole('button', { name: /grant required consents/i }));
    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.click(screen.getByRole('button', { name: /create account/i }));

    expect(await screen.findByText('Registration Successful!')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /go to login/i })).toHaveAttribute('href', '/login');

    fireEvent.click(screen.getByRole('button', { name: /resend verification email/i }));

    await waitFor(() => {
      expect(authService.resendVerificationEmail).toHaveBeenCalledWith('owner@example.com');
    });
  });
});
