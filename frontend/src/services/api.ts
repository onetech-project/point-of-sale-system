import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig, AxiosResponse } from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

class APIClient {
  private axiosInstance: AxiosInstance;
  private onAuthError?: () => void;
  private isRefreshing = false;
  private refreshPromise: Promise<boolean> | null = null;

  constructor() {
    this.axiosInstance = axios.create({
      baseURL: API_BASE_URL,
      timeout: 30000,
      withCredentials: true,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.setupInterceptors();
  }

  // Set callback for authentication errors
  public setAuthErrorHandler(handler: () => void): void {
    this.onAuthError = handler;
  }

  // Attempt to refresh the session
  private async refreshSession(): Promise<boolean> {
    // If already refreshing, return the existing promise
    if (this.isRefreshing && this.refreshPromise) {
      return this.refreshPromise;
    }

    this.isRefreshing = true;
    this.refreshPromise = (async () => {
      try {
        const response = await axios.post(
          `${API_BASE_URL}/api/auth/refresh`,
          {},
          {
            withCredentials: true, // Send existing cookie
          }
        );

        return response.status === 200;
      } catch (error) {
        console.error('Token refresh failed:', error);
        return false;
      } finally {
        this.isRefreshing = false;
        this.refreshPromise = null;
      }
    })();

    return this.refreshPromise;
  }

  private setupInterceptors() {
    // Request interceptor - Cookies are sent automatically with withCredentials: true
    // No need to manually add Authorization headers for cookie-based auth
    this.axiosInstance.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        // Cookies are automatically included with withCredentials: true
        return config;
      },
      (error: AxiosError) => {
        return Promise.reject(error);
      }
    );

    // Response interceptor - Handle authentication errors with refresh attempt
    this.axiosInstance.interceptors.response.use(
      (response: AxiosResponse) => {
        return response;
      },
      async (error: AxiosError) => {
        const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

        // Handle 401 authentication errors
        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true;

          // Attempt to refresh the session
          const refreshed = await this.refreshSession();

          if (refreshed) {
            // Retry the original request with new token
            return this.axiosInstance(originalRequest);
          } else {
            // Refresh failed, call auth error handler
            if (this.onAuthError) {
              this.onAuthError();
            }
          }
        }

        return Promise.reject(error);
      }
    );
  }

  private clearAuthData(): void {
    if (typeof window !== 'undefined') {
      // Only clear user data, not tokens (tokens are in HTTP-only cookies managed by backend)
      localStorage.removeItem('user');
    }
  }

  // Public method to clear auth data (used on logout)
  public clearAuth(): void {
    this.clearAuthData();
  }

  // HTTP Methods
  async get<T = any>(endpoint: string, config = {}): Promise<T> {
    const response = await this.axiosInstance.get<T>(endpoint, config);
    return response.data;
  }

  async post<T = any>(endpoint: string, data?: any, config = {}): Promise<T> {
    const response = await this.axiosInstance.post<T>(endpoint, data, config);
    return response.data;
  }

  async put<T = any>(endpoint: string, data?: any, config = {}): Promise<T> {
    const response = await this.axiosInstance.put<T>(endpoint, data, config);
    return response.data;
  }

  async patch<T = any>(endpoint: string, data?: any, config = {}): Promise<T> {
    const response = await this.axiosInstance.patch<T>(endpoint, data, config);
    return response.data;
  }

  async delete<T = any>(endpoint: string, config = {}): Promise<T> {
    const response = await this.axiosInstance.delete<T>(endpoint, config);
    return response.data;
  }

  // Get axios instance for advanced usage
  public getAxiosInstance(): AxiosInstance {
    return this.axiosInstance;
  }
}

const apiClient = new APIClient();

export default apiClient;
