'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { useAuth } from '../../hooks/useAuth';

interface ProtectedRouteProps {
  children: React.ReactNode;
  allowedRoles?: string[];
  requireAuth?: boolean;
}

export default function ProtectedRoute({
  children,
  allowedRoles = [],
  requireAuth = true,
}: ProtectedRouteProps) {
  const router = useRouter();
  const { user, loading } = useAuth();
  const [authorized, setAuthorized] = useState(false);

  useEffect(() => {
    if (loading) return;

    if (requireAuth && !user) {
      router.push('/auth/login');
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
  }, [user, loading, allowedRoles, requireAuth, router]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (!authorized) {
    return null;
  }

  return <>{children}</>;
}
