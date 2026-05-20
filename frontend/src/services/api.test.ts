import type { AxiosError } from 'axios';

let responseErrorHandler: ((error: AxiosError) => Promise<never>) | undefined;

const mockAxiosInstance = {
  defaults: { baseURL: 'http://localhost:8080' },
  interceptors: {
    request: {
      use: jest.fn(),
    },
    response: {
      use: jest.fn((_success, error) => {
        responseErrorHandler = error;
      }),
    },
  },
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
  patch: jest.fn(),
  delete: jest.fn(),
};

jest.mock('axios', () => ({
  __esModule: true,
  default: {
    create: jest.fn(() => mockAxiosInstance),
    post: jest.fn(),
  },
}));

describe('APIClient tenant availability interceptor', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.resetModules();
    responseErrorHandler = undefined;
  });

  it('calls tenant unavailable handler for protected suspended tenant responses', async () => {
    const apiClient = (await import('./api')).default;
    const handler = jest.fn();
    apiClient.setTenantUnavailableHandler(handler);

    await expect(getResponseErrorHandler()({
      response: {
        status: 403,
        data: { status: 'suspended', error: 'Tenant account suspended' },
      },
      config: { url: '/api/auth/session' },
    } as AxiosError)).rejects.toBeDefined();

    expect(handler).toHaveBeenCalledWith('suspended');
  });

  it('does not route public guest tenant unavailable responses to account unavailable', async () => {
    const apiClient = (await import('./api')).default;
    const handler = jest.fn();
    apiClient.setTenantUnavailableHandler(handler);

    await expect(getResponseErrorHandler()({
      response: {
        status: 403,
        data: { status: 'inactive', error: 'Tenant currently unavailable' },
      },
      config: { url: '/api/public/menu/tenant-1/products' },
    } as AxiosError)).rejects.toBeDefined();

    expect(handler).not.toHaveBeenCalled();
  });
});

function getResponseErrorHandler(): (error: AxiosError) => Promise<never> {
  if (!responseErrorHandler) {
    throw new Error('response error interceptor was not registered');
  }
  return responseErrorHandler;
}
