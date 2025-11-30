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
}

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string, tenantSlug?: string) => Promise<any>;
  logout: () => Promise<void>;
  register: (businessName: string, email: string, password: string, firstName?: string, lastName?: string) => Promise<any>;
  checkAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  const checkAuth = useCallback(async () => {
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
        setUser(null);
        setIsAuthenticated(false);
        apiClient.clearAuth();
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
  }, [checkAuth]);

  const login = async (email: string, password: string, tenantSlug?: string) => {
    try {
      const data = await apiClient.post<{ user: User; message: string }>('/api/auth/login', {
        email,
        password,
        tenant_slug: tenantSlug,
      });

      // Token is stored in HTTP-only cookie by backend, no need to store in localStorage
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

  const register = async (businessName: string, email: string, password: string, firstName?: string, lastName?: string) => {
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
