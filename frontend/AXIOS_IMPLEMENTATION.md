# Axios Implementation with Token Management

## Overview

The frontend now uses Axios instead of fetch API, with advanced features including:
- Automatic Bearer token injection for authenticated requests
- Automatic token refresh when expired
- Request queue management during token refresh
- Centralized error handling

## Architecture

### API Client Structure

```
src/services/api.ts
├── APIClient Class
│   ├── Axios Instance (with interceptors)
│   ├── Request Interceptor (adds Bearer token)
│   ├── Response Interceptor (handles token refresh)
│   └── HTTP Methods (get, post, put, patch, delete)
└── Exported apiClient instance
```

## Key Features

### 1. Automatic Bearer Token Injection (Private Routes Only)

The request interceptor intelligently adds the Bearer token **only to private/authenticated endpoints**:

```typescript
// Public endpoints that don't need authentication
const publicEndpoints = [
  '/auth/login',
  '/auth/register', 
  '/auth/refresh',
  '/auth/forgot-password',
  '/auth/reset-password',
  '/auth/verify-email',
  '/tenants/register',
  '/health',
];

// Token only added if NOT a public endpoint
if (!isPublicEndpoint) {
  config.headers.Authorization = `Bearer ${token}`;
}
```

**How it works:**
- Checks if endpoint is in public list
- If public → No token added
- If private → Token automatically added
- No need to manually manage tokens

### 2. Automatic Token Refresh (Private Routes Only)

When a **private endpoint** request receives a 401 (Unauthorized) response:

```typescript
// Response Interceptor detects 401
if (error.response?.status === 401 && !originalRequest._retry) {
  // Attempt to refresh token
  const newToken = await this.refreshAccessToken();
  // Retry original request with new token
  return this.axiosInstance(originalRequest);
}
```

**Workflow:**
1. Request receives 401 response
2. Interceptor catches the error
3. Calls `/auth/refresh` endpoint
4. Receives new token
5. Retries original request with new token
6. Returns response to caller

### 3. Request Queue Management

When multiple requests fail simultaneously (token expired):

```typescript
// First request triggers refresh
this.isRefreshing = true;

// Subsequent requests are queued
this.failedQueue.push({ resolve, reject });

// After refresh, all queued requests are processed
this.processQueue(null, newToken);
```

**Benefits:**
- Prevents multiple simultaneous refresh calls
- Ensures all pending requests use the new token
- Maintains request order and integrity

### 4. Graceful Failure Handling

If token refresh fails:
- Clears all auth data
- Redirects to login page with session expired message
- Rejects all queued requests

## Usage Examples

### Basic GET Request

```typescript
import apiClient from '@/services/api';

// Automatically includes Bearer token
const data = await apiClient.get('/api/users');
```

### POST Request with Data

```typescript
const newUser = await apiClient.post('/api/users', {
  name: 'John Doe',
  email: 'john@example.com'
});
```

### Custom Configuration

```typescript
const data = await apiClient.get('/api/users', {
  params: { page: 1, limit: 10 },
  headers: { 'X-Custom-Header': 'value' }
});
```

## Authentication Flow

### Login Flow

```typescript
// 1. User logs in
const response = await apiClient.post('/auth/login', {
  email: 'user@example.com',
  password: 'password'
});

// 2. Token is stored
apiClient.setToken(response.token);
localStorage.setItem('user', JSON.stringify(response.user));

// 3. All subsequent requests include token
```

### Token Refresh Flow

```
User Request
    ↓
[Request Interceptor]
    ↓ (Add Bearer Token)
API Server
    ↓
401 Unauthorized
    ↓
[Response Interceptor]
    ↓
Check if refreshing?
    ├─ Yes → Queue request
    └─ No → Start refresh
         ↓
    Call /auth/refresh
         ↓
    New Token Received
         ↓
    Update all queued requests
         ↓
    Retry original requests
         ↓
    Return responses
```

### Logout Flow

```typescript
// 1. Call logout endpoint
await apiClient.post('/auth/logout');

// 2. Clear auth data
apiClient.clearAuth();
localStorage.removeItem('user');

// 3. Redirect to login
router.push('/login');
```

## Configuration

### Environment Variables

```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Axios Configuration

```typescript
{
  baseURL: 'http://localhost:8080',
  timeout: 30000, // 30 seconds
  withCredentials: true, // Send cookies
  headers: {
    'Content-Type': 'application/json'
  }
}
```

## Token Storage

### Access Token
- **Location**: `localStorage.getItem('access_token')`
- **Usage**: Sent with every request in Authorization header
- **Lifetime**: Short-lived (e.g., 15 minutes)

### User Data
- **Location**: `localStorage.getItem('user')`
- **Contains**: User profile information
- **Usage**: Display user info without additional requests

## Error Handling

### Network Errors

```typescript
try {
  const data = await apiClient.get('/api/users');
} catch (error) {
  if (axios.isAxiosError(error)) {
    if (error.response) {
      // Server responded with error status
      console.error('Server error:', error.response.data);
    } else if (error.request) {
      // Request made but no response received
      console.error('Network error:', error.message);
    }
  }
}
```

### Token Refresh Failure

When token refresh fails:
1. All auth data is cleared
2. User is redirected to `/login?session=expired`
3. User sees "Session expired, please log in again" message

## Security Considerations

### Token Storage
- Access tokens stored in localStorage (XSS vulnerable)
- Consider using HttpOnly cookies for production
- Tokens are automatically cleared on logout

### CORS Configuration
- `withCredentials: true` enables cookie sending
- Backend must set appropriate CORS headers

### Token Lifetime
- Short-lived access tokens (15 minutes recommended)
- Refresh tokens for seamless UX
- Automatic cleanup on session end

## API Methods

### Available Methods

```typescript
// GET request
apiClient.get<T>(endpoint: string, config?: AxiosRequestConfig): Promise<T>

// POST request
apiClient.post<T>(endpoint: string, data?: any, config?: AxiosRequestConfig): Promise<T>

// PUT request
apiClient.put<T>(endpoint: string, data?: any, config?: AxiosRequestConfig): Promise<T>

// PATCH request
apiClient.patch<T>(endpoint: string, data?: any, config?: AxiosRequestConfig): Promise<T>

// DELETE request
apiClient.delete<T>(endpoint: string, config?: AxiosRequestConfig): Promise<T>

// Set token manually
apiClient.setToken(token: string): void

// Clear auth data
apiClient.clearAuth(): void

// Get axios instance (advanced usage)
apiClient.getAxiosInstance(): AxiosInstance
```

## Backend Requirements

The frontend expects the following backend endpoints:

### 1. Token Refresh Endpoint

```
POST /auth/refresh
```

**Headers:**
- Cookie: Contains refresh token

**Response:**
```json
{
  "token": "new_access_token_here",
  "user": {
    "id": "user_id",
    "email": "user@example.com"
  }
}
```

### 2. Login Endpoint

```
POST /auth/login
```

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password"
}
```

**Response:**
```json
{
  "token": "access_token_here",
  "user": {
    "id": "user_id",
    "email": "user@example.com",
    "role": "admin"
  },
  "tenantId": "tenant_id"
}
```

## Migration from Fetch

### Before (Fetch)

```typescript
const response = await fetch(`${API_URL}/users`, {
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  }
});
const data = await response.json();
```

### After (Axios)

```typescript
// Token automatically added
const data = await apiClient.get('/users');
```

## Testing

### Mock API Client

```typescript
jest.mock('@/services/api', () => ({
  default: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  }
}));
```

### Test Token Refresh

```typescript
test('should refresh token on 401', async () => {
  // Mock 401 response then success
  apiClient.get.mockRejectedValueOnce({ response: { status: 401 } })
             .mockResolvedValueOnce({ data: 'success' });
  
  const result = await apiClient.get('/protected');
  expect(result).toBe('success');
});
```

## Troubleshooting

### Issue: Token not being sent

**Solution:** Check that token is stored in localStorage:
```typescript
console.log(localStorage.getItem('access_token'));
```

### Issue: Infinite refresh loop

**Solution:** Ensure refresh endpoint doesn't trigger refresh interceptor:
- Use axios.post directly instead of apiClient.post in refreshAccessToken()

### Issue: CORS errors

**Solution:** Backend must include:
```
Access-Control-Allow-Credentials: true
Access-Control-Allow-Origin: http://localhost:3000
```

## Performance Considerations

- Axios instance is created once (singleton pattern)
- Failed requests are queued, not dropped
- Automatic retry reduces user-facing errors
- LocalStorage access is synchronous and fast

## Future Enhancements

1. **Secure Token Storage**: Move to HttpOnly cookies
2. **Token Expiry Prediction**: Refresh before expiry
3. **Offline Support**: Queue requests when offline
4. **Request Caching**: Cache GET requests
5. **Rate Limiting**: Client-side rate limit protection

---

**Version**: 1.0.0  
**Last Updated**: 2025-11-23  
**Maintained By**: Development Team
