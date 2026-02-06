'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Modal from '@/components/ui/Modal';

const SessionWarning: React.FC = () => {
  const router = useRouter();
  const [showWarning, setShowWarning] = useState(false);
  const [sessionTimeLeft, setSessionTimeLeft] = useState(15 * 60); // 15 minutes in seconds
  const [lastActivity, setLastActivity] = useState(Date.now());

  // Reset session timer on user activity
  useEffect(() => {
    const handleActivity = () => {
      const now = Date.now();
      // Only reset if more than 1 second has passed (debounce)
      if (now - lastActivity > 1000) {
        setLastActivity(now);
        setSessionTimeLeft(15 * 60); // Reset to 15 minutes
        if (showWarning) {
          setShowWarning(false);
        }
      }
    };

    // Listen to user activity events
    const events = ['mousedown', 'keydown', 'scroll', 'touchstart'];
    events.forEach(event => {
      document.addEventListener(event, handleActivity);
    });

    return () => {
      events.forEach(event => {
        document.removeEventListener(event, handleActivity);
      });
    };
  }, [lastActivity, showWarning]);

  useEffect(() => {
    const interval = setInterval(() => {
      setSessionTimeLeft((prev) => {
        const newTime = prev - 1;

        // Show warning when 2 minutes left
        if (newTime === 120 && !showWarning) {
          setShowWarning(true);
        }

        // Auto logout when time runs out
        if (newTime <= 0) {
          router.push('/login?session_expired=true');
        }

        return newTime;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [showWarning, router]);

  const handleExtendSession = async () => {
    try {
      // Call the session endpoint to trigger renewal on backend
      // This will renew both Redis TTL and the auth cookie
      const response = await fetch('/api/auth/session', {
        method: 'GET',
        credentials: 'include', // Include cookies
      });

      if (response.ok) {
        setShowWarning(false);
        setSessionTimeLeft(15 * 60); // Reset to 15 minutes
      } else {
        // Session couldn't be renewed, redirect to login
        router.push('/login?session_expired=true');
      }
    } catch (error) {
      console.error('Failed to extend session:', error);
      router.push('/login?session_expired=true');
    }
  };

  return (
    <Modal
      isOpen={showWarning}
      onClose={() => setShowWarning(false)}
      title="Session Expiring Soon"
      size="sm"
    >
      <div className="text-center py-4">
        <svg className="mx-auto h-12 w-12 text-yellow-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <h3 className="mt-4 text-lg font-medium text-gray-900">Your session is about to expire</h3>
        <p className="mt-2 text-sm text-gray-600">
          You will be logged out in {Math.floor(sessionTimeLeft / 60)} minutes unless you extend your session.
        </p>
        <div className="mt-6 flex flex-col space-y-3">
          <button
            onClick={handleExtendSession}
            className="w-full px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
          >
            Extend Session
          </button>
          <button
            onClick={() => router.push('/login')}
            className="w-full px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
          >
            Logout Now
          </button>
        </div>
      </div>
    </Modal>
  );
};

export default SessionWarning;
