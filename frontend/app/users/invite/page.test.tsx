import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import InviteUserPage from './page';
import userService from '@/services/user';
import { useAuth } from '@/store/auth';

const mockPush = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
}));

jest.mock('@/store/auth', () => ({
  useAuth: jest.fn(),
}));

jest.mock('@/components/auth/ProtectedRoute', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/components/layout/DashboardLayout', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/services/user', () => ({
  __esModule: true,
  default: {
    getTeamMembers: jest.fn(),
    updateTeamMember: jest.fn(),
    getInvitations: jest.fn(),
    inviteUser: jest.fn(),
    resendInvitation: jest.fn(),
    revokeInvitation: jest.fn(),
  },
}));

const mockUserService = userService as jest.Mocked<typeof userService>;
const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  return { promise, resolve, reject };
}

const cashier = {
  id: 'cashier-1',
  email: 'cashier@example.com',
  role: 'cashier' as const,
  status: 'active' as const,
  firstName: 'Citra',
  lastName: 'Cashier',
  createdAt: '2026-05-01T00:00:00Z',
  updatedAt: '2026-05-01T00:00:00Z',
};

const manager = {
  id: 'manager-2',
  email: 'manager@example.com',
  role: 'manager' as const,
  status: 'active' as const,
  firstName: 'Mina',
  lastName: 'Manager',
  createdAt: '2026-05-01T00:00:00Z',
  updatedAt: '2026-05-01T00:00:00Z',
};

beforeEach(() => {
  jest.clearAllMocks();
  mockUseAuth.mockReturnValue({
    user: { id: 'manager-1', email: 'current@example.com', role: 'manager' },
  } as any);
  mockUserService.getTeamMembers.mockResolvedValue([cashier, manager]);
  mockUserService.getInvitations.mockResolvedValue([
    {
      id: 'pending-1',
      email: 'pending@example.com',
      role: 'cashier',
      status: 'pending',
      expiresAt: '2026-05-30T00:00:00Z',
      invitedBy: 'manager-1',
      createdAt: '2026-05-22T00:00:00Z',
    },
    {
      id: 'accepted-invite-1',
      email: 'accepted-invite@example.com',
      role: 'cashier',
      status: 'accepted',
      expiresAt: '2026-05-30T00:00:00Z',
      invitedBy: 'manager-1',
      createdAt: '2026-05-22T00:00:00Z',
    },
  ]);
  mockUserService.updateTeamMember.mockResolvedValue(cashier);
  mockUserService.revokeInvitation.mockResolvedValue({
    id: 'pending-1',
    email: 'pending@example.com',
    role: 'cashier',
    status: 'revoked',
    expiresAt: '2026-05-30T00:00:00Z',
    invitedBy: 'manager-1',
    createdAt: '2026-05-22T00:00:00Z',
  });
});

afterEach(() => {
  jest.restoreAllMocks();
});

it('separates accepted team members from pending invitations', async () => {
  render(<InviteUserPage />);

  expect(await screen.findByText('Accepted Team Members')).toBeInTheDocument();
  expect(screen.getByText('Pending Invitations')).toBeInTheDocument();
  expect(await screen.findByText('cashier@example.com')).toBeInTheDocument();
  expect(await screen.findByText('pending@example.com')).toBeInTheDocument();
  expect(screen.queryByText('accepted-invite@example.com')).not.toBeInTheDocument();
});

it('lets managers manage cashier rows but not manager rows', async () => {
  render(<InviteUserPage />);

  const cashierRole = await screen.findByLabelText('Change role for cashier@example.com');
  expect(cashierRole).toBeInTheDocument();
  expect(screen.queryByLabelText('Change role for manager@example.com')).not.toBeInTheDocument();

  fireEvent.change(cashierRole, { target: { value: 'manager' } });

  await waitFor(() => {
    expect(mockUserService.updateTeamMember).toHaveBeenCalledWith('cashier-1', {
      role: 'manager',
    });
  });
});

it('shows a loading indicator while changing a member role', async () => {
  const update = createDeferred<typeof cashier>();
  mockUserService.updateTeamMember.mockReturnValueOnce(update.promise);

  render(<InviteUserPage />);

  const cashierRole = await screen.findByLabelText('Change role for cashier@example.com');
  const statusButton = screen.getByRole('button', { name: 'Deactivate' });

  fireEvent.change(cashierRole, { target: { value: 'manager' } });

  expect(await screen.findByText('Saving role...')).toBeInTheDocument();
  expect(cashierRole).toBeDisabled();
  expect(statusButton).toBeDisabled();

  update.resolve(cashier);

  await waitFor(() => {
    expect(screen.queryByText('Saving role...')).not.toBeInTheDocument();
  });
});

it('lets managers deactivate cashier rows', async () => {
  render(<InviteUserPage />);

  fireEvent.click(await screen.findByRole('button', { name: 'Deactivate' }));

  await waitFor(() => {
    expect(mockUserService.updateTeamMember).toHaveBeenCalledWith('cashier-1', {
      status: 'suspended',
    });
  });
});

it('shows a loading indicator while changing member status', async () => {
  const update = createDeferred<typeof cashier>();
  mockUserService.updateTeamMember.mockReturnValueOnce(update.promise);

  render(<InviteUserPage />);

  const cashierRole = await screen.findByLabelText('Change role for cashier@example.com');

  fireEvent.click(screen.getByRole('button', { name: 'Deactivate' }));

  expect(await screen.findByRole('button', { name: 'Deactivating...' })).toBeDisabled();
  expect(cashierRole).toBeDisabled();

  update.resolve(cashier);

  await waitFor(() => {
    expect(screen.queryByRole('button', { name: 'Deactivating...' })).not.toBeInTheDocument();
  });
});

it('lets owners manage manager rows', async () => {
  mockUseAuth.mockReturnValue({
    user: { id: 'owner-1', email: 'owner@example.com', role: 'owner' },
  } as any);

  render(<InviteUserPage />);

  expect(await screen.findByLabelText('Change role for manager@example.com')).toBeInTheDocument();
});

it('revokes pending invitations', async () => {
  jest.spyOn(window, 'confirm').mockReturnValue(true);

  render(<InviteUserPage />);

  fireEvent.click(await screen.findByRole('button', { name: 'Revoke' }));

  await waitFor(() => {
    expect(mockUserService.revokeInvitation).toHaveBeenCalledWith('pending-1');
  });
});
