'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/store/auth';
import { useSubscription } from '@/store/subscription';

interface GracePeriodWallProps {
  onDismiss?: () => void;
}

const GracePeriodWall: React.FC<GracePeriodWallProps> = ({ onDismiss }) => {
  const router = useRouter();
  const { logout } = useAuth();
  const { setExpired } = useSubscription();

  const handleSubscribe = () => {
    setExpired(false);
    router.push('/subscription');
    onDismiss?.();
  };

  const handleLogout = async () => {
    setExpired(false);
    await logout();
  };

  return (
    <div className="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-xl max-w-md w-full p-8 flex flex-col items-center text-center">
        <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mb-4">
          <svg
            className="w-8 h-8 text-red-600"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
            />
          </svg>
        </div>

        <h2 className="text-xl font-bold text-gray-900 mb-2">Subscription Required</h2>
        <p className="text-gray-600 mb-6">
          Your subscription has expired. Subscribe to continue using the POS system.
        </p>

        <button
          onClick={handleSubscribe}
          className="w-full bg-primary-600 hover:bg-primary-700 text-white font-semibold py-3 px-6 rounded-lg transition-colors mb-3"
        >
          Subscribe Now
        </button>

        <button
          onClick={handleLogout}
          className="text-sm text-gray-500 hover:text-gray-700 transition-colors"
        >
          Log Out
        </button>
      </div>
    </div>
  );
};

export default GracePeriodWall;
