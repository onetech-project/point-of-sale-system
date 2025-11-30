'use client';

import React, { useState, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import userService from '@/services/user';
import PublicLayout from '@/components/layout/PublicLayout';

function AcceptInvitationForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get('token');

  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [passwordStrength, setPasswordStrength] = useState({
    minLength: false,
    hasLetterAndNumber: false,
    hasUpperCase: false,
    hasSpecialChar: false,
  });

  const handlePasswordChange = (value: string) => {
    setPassword(value);
    setPasswordStrength({
      minLength: value.length >= 8,
      hasLetterAndNumber: /(?=.*[A-Za-z])(?=.*\d)/.test(value),
      hasUpperCase: /[A-Z]/.test(value),
      hasSpecialChar: /[!@#$%^&*(),.?":{}|<>]/.test(value),
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    if (!/(?=.*[A-Za-z])(?=.*\d)/.test(password)) {
      setError('Password must contain both letters and numbers');
      return;
    }

    if (!token) {
      setError('Invalid invitation link');
      return;
    }

    setLoading(true);

    try {
      await userService.acceptInvitation(token, {
        firstName,
        lastName,
        password,
      });

      // Redirect to login with success message
      router.push('/login?message=account_created');
    } catch (err: any) {
      const errorMessage = err.response?.data?.error || 'Failed to accept invitation';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  if (!token) {
    return (
      <PublicLayout>
        <div className="min-h-[calc(100vh-128px)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
          <div className="max-w-md w-full text-center">
            <div className="bg-red-50 border border-red-200 rounded-lg p-6">
              <svg className="mx-auto h-12 w-12 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <h2 className="mt-4 text-xl font-semibold text-gray-900">Invalid Invitation Link</h2>
              <p className="mt-2 text-sm text-gray-600">
                This invitation link is invalid or has expired.
              </p>
              <button
                onClick={() => router.push('/login')}
                className="mt-6 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
              >
                Go to Login
              </button>
            </div>
          </div>
        </div>
      </PublicLayout>
    );
  }

  return (
    <PublicLayout>
      <div className="min-h-[calc(100vh-128px)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full space-y-8">
          <div>
            <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
              Accept Invitation
            </h2>
            <p className="mt-2 text-center text-sm text-gray-600">
              Complete your account setup to join the team
            </p>
          </div>

          <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
            <div className="bg-white rounded-lg shadow-md p-8 space-y-6">
              {error && (
                <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
                  {error}
                </div>
              )}

              <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
                <div>
                  <label htmlFor="firstName" className="block text-sm font-medium text-gray-700 mb-2">
                    First Name
                  </label>
                  <input
                    id="firstName"
                    name="firstName"
                    type="text"
                    required
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    className="appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-lg focus:outline-none focus:ring-primary-500 focus:border-primary-500 focus:z-10 sm:text-sm"
                    placeholder="John"
                  />
                </div>

                <div>
                  <label htmlFor="lastName" className="block text-sm font-medium text-gray-700 mb-2">
                    Last Name
                  </label>
                  <input
                    id="lastName"
                    name="lastName"
                    type="text"
                    required
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    className="appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-lg focus:outline-none focus:ring-primary-500 focus:border-primary-500 focus:z-10 sm:text-sm"
                    placeholder="Doe"
                  />
                </div>
              </div>

              <div>
                <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-2">
                  Password <span className="text-red-500">*</span>
                </label>
                <input
                  id="password"
                  name="password"
                  type="password"
                  required
                  value={password}
                  onChange={(e) => handlePasswordChange(e.target.value)}
                  className="appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-lg focus:outline-none focus:ring-primary-500 focus:border-primary-500 focus:z-10 sm:text-sm"
                  placeholder="At least 8 characters"
                />

                {/* Password Strength Indicators */}
                {password && (
                  <div className="mt-3 space-y-2">
                    <p className="text-xs font-medium text-gray-700">
                      Password Requirements:
                    </p>
                    <div className="space-y-1.5">
                      <PasswordRequirement
                        met={passwordStrength.minLength}
                        text="At least 8 characters"
                      />
                      <PasswordRequirement
                        met={passwordStrength.hasLetterAndNumber}
                        text="Contains letters and numbers"
                      />
                      <PasswordRequirement
                        met={passwordStrength.hasUpperCase}
                        text="Contains uppercase letter"
                      />
                      <PasswordRequirement
                        met={passwordStrength.hasSpecialChar}
                        text="Contains special character"
                      />
                    </div>
                  </div>
                )}
              </div>

              <div>
                <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 mb-2">
                  Confirm Password <span className="text-red-500">*</span>
                </label>
                <input
                  id="confirmPassword"
                  name="confirmPassword"
                  type="password"
                  required
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-lg focus:outline-none focus:ring-primary-500 focus:border-primary-500 focus:z-10 sm:text-sm"
                  placeholder="Re-enter password"
                />
              </div>

              <div className="pt-4">
                <button
                  type="submit"
                  disabled={loading}
                  className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-lg text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
                >
                  {loading ? 'Creating Account...' : 'Accept Invitation & Create Account'}
                </button>
              </div>
            </div>
          </form>

          <div className="text-center">
            <button
              onClick={() => router.push('/login')}
              className="text-sm text-primary-600 hover:text-primary-500"
            >
              Already have an account? Sign in
            </button>
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}

// Password Requirement Component
function PasswordRequirement({ met, text }: { met: boolean; text: string }) {
  return (
    <div className="flex items-center space-x-2">
      {met ? (
        <svg className="w-4 h-4 text-green-500 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
        </svg>
      ) : (
        <svg className="w-4 h-4 text-gray-300 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
        </svg>
      )}
      <span className={`text-xs ${met ? 'text-green-700 font-medium' : 'text-gray-500'}`}>
        {text}
      </span>
    </div>
  );
}

export default function AcceptInvitationPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
        </div>
      }
    >
      <AcceptInvitationForm />
    </Suspense>
  );
}
