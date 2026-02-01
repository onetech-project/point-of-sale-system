'use client';

import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import apiClient from '@/services/api';
import { AxiosError } from 'axios';
import { type Role } from '@/constants/roles';

interface User {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  role: Role;
  tenantId?: string;
}

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string, tenantSlug?: string) => Promise<any>;
  logout: () => Promise<void>;
  register: (
    businessName: string,
    email: string,
    password: string,
    firstName?: string,
    lastName?: string
  ) => Promise<any>;
  checkAuth: () => Promise<void>;
}

// Public pages that don't require authentication
const PUBLIC_PAGES = {
  AUTH: ['/login', '/register'],
  GUEST_ORDER_PATTERN: /^\/orders\/[A-Z0-9-]+$/, // Matches /orders/{orderReference}
  GUEST_PATHS: ['/menu/', '/checkout/'],
};

const isPublicPage = (pathname: string): boolean => {
  // Check exact auth pages
  if (PUBLIC_PAGES.AUTH.includes(pathname)) return true;

  // Check guest order status page pattern
  if (PUBLIC_PAGES.GUEST_ORDER_PATTERN.test(pathname)) return true;

  // Check guest ordering paths
  if (PUBLIC_PAGES.GUEST_PATHS.some(path => pathname.includes(path))) return true;

  return false;
};

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  // Handle authentication errors globally
  const handleAuthError = useCallback(() => {
    // Clear auth state
    setUser(null);
    setIsAuthenticated(false);
    apiClient.clearAuth();
  }, []);

  // Register auth error handler with API client
  useEffect(() => {
    apiClient.setAuthErrorHandler(handleAuthError);
  }, [handleAuthError]);

  const checkAuth = useCallback(async () => {
    // Skip auth check for public routes (but don't clear auth state if already authenticated)
    if (typeof window !== 'undefined') {
      const pathname = window.location.pathname;

      if (isPublicPage(pathname)) {
        // Don't clear existing auth state on public pages
        // This allows proper redirect after login
        setIsLoading(false);
        return;
      }
    }

    try {
      const data = await apiClient.get<{ user: User }>('/api/auth/session');

      if (data && data.user) {
        setUser(data.user);
        setIsAuthenticated(true);
      } else {
        setUser(null);
        setIsAuthenticated(false);
      }
    } catch (error) {
      const axiosError = error as AxiosError;

      // If 401, no valid session exists
      if (axiosError.response?.status === 401) {
        handleAuthError();
        
        // Redirect to login if not on a public page
        if (typeof window !== 'undefined') {
          const pathname = window.location.pathname;
          if (!isPublicPage(pathname)) {
            window.location.href = '/login?session_expired=true';
          }
        }
      } else {
        // For network errors or other issues, default to not authenticated
        console.error('Auth check failed:', error);
        setUser(null);
        setIsAuthenticated(false);
      }
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    checkAuth();

    // Set up periodic session validation (every 5 minutes) to trigger renewal
    // This ensures the session stays alive as long as the user has the app open
    const interval = setInterval(() => {
      // Only check auth if not on a public page
      if (typeof window !== 'undefined' && !isPublicPage(window.location.pathname)) {
        checkAuth();
      }
    }, 5 * 60 * 1000); // 5 minutes

    return () => clearInterval(interval);
  }, [checkAuth]);

  const login = async (email: string, password: string) => {
    try {
      const data = await apiClient.post<{ user: User; message: string }>('/api/auth/login', {
        email,
        password,
      });

      // Token is stored in HTTP-only cookie by backend
      setUser(data.user);
      setIsAuthenticated(true);
      return data;
    } catch (error) {
      const axiosError = error as AxiosError<{ error?: string }>;
      throw new Error(axiosError.response?.data?.error || 'Login failed');
    }
  };

  const logout = async () => {
    try {
      await apiClient.post('/api/auth/logout');
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      apiClient.clearAuth();
      setUser(null);
      setIsAuthenticated(false);
    }
  };

  const register = async (
    businessName: string,
    email: string,
    password: string,
    firstName?: string,
    lastName?: string
  ) => {
    try {
      const data = await apiClient.post('/api/tenants/register', {
        business_name: businessName,
        email,
        password,
        first_name: firstName,
        last_name: lastName,
      });

      return data;
    } catch (error) {
      const axiosError = error as AxiosError<{ error?: string }>;
      throw new Error(axiosError.response?.data?.error || 'Registration failed');
    }
  };

  const value: AuthContextType = {
    user,
    isLoading,
    isAuthenticated,
    login,
    logout,
    register,
    checkAuth,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
