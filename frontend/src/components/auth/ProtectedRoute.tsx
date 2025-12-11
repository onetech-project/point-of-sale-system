'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '../../store/auth';
import { ROLES, type Role } from '../../constants/roles';
import SessionWarning from './SessionWarning';
import ToastProvider from '../ui/Toast';
import useSSENotifications from '../../hooks/useSSENotifications';

interface ProtectedRouteProps {
  children: React.ReactNode;
  allowedRoles?: Role[];
  requireAuth?: boolean;
}

export default function ProtectedRoute({
  children,
  allowedRoles = [],
  requireAuth = true,
}: ProtectedRouteProps) {
  const router = useRouter();
  const { user, isLoading, isAuthenticated } = useAuth();
  const [authorized, setAuthorized] = useState(false);

  useEffect(() => {
    if (isLoading) return;

    if (requireAuth && !isAuthenticated) {
      router.push('/login');
      return;
    }

    if (allowedRoles.length > 0 && user) {
      const hasPermission = allowedRoles.includes(user.role);
      if (!hasPermission) {
        router.push('/unauthorized');
        return;
      }
    }

    setAuthorized(true);
  }, [user, isLoading, isAuthenticated, allowedRoles, requireAuth, router]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (!authorized) {
    return null;
  }

  return (
    <>
      <SessionWarning />
      {/* ToastProvider renders the toast container and provides `useToasts()` */}
      <ToastProvider>
        {/* SSE listener runs inside the ToastProvider so it can call `useToasts()` */}
        <SSEListener />
        {children}
      </ToastProvider>
    </>
  );
}

function SSEListener() {
  useSSENotifications()
  return null
}
