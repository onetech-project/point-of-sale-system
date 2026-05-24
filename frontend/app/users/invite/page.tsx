'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import userService, { Invitation, TeamMember, TeamMemberUpdateRequest } from '@/services/user';
import { useAuth } from '@/store/auth';
import { ROLES, type Role } from '@/constants/roles';

type MemberAction = 'role' | 'status';

export default function InviteUserPage() {
  const { user } = useAuth();
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [role, setRole] = useState<Role>(ROLES.CASHIER);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [loadingInvitations, setLoadingInvitations] = useState(true);
  const [loadingTeamMembers, setLoadingTeamMembers] = useState(true);
  const [savingMember, setSavingMember] = useState<string | null>(null);
  const [savingInvitation, setSavingInvitation] = useState<string | null>(null);

  useEffect(() => {
    loadTeamMembers();
    loadInvitations();
  }, []);

  const pendingInvitations = useMemo(
    () => invitations.filter((invitation) => invitation.status === 'pending'),
    [invitations]
  );

  const loadTeamMembers = async () => {
    try {
      setLoadingTeamMembers(true);
      const data = await userService.getTeamMembers();
      setTeamMembers(data);
    } catch (err: any) {
      console.error('Failed to load team members:', err);
      setError(err.response?.data?.error || 'Failed to load team members');
    } finally {
      setLoadingTeamMembers(false);
    }
  };

  const loadInvitations = async () => {
    try {
      setLoadingInvitations(true);
      const data = await userService.getInvitations();
      setInvitations(data);
    } catch (err: any) {
      console.error('Failed to load invitations:', err);
      setError(err.response?.data?.error || 'Failed to load invitations');
    } finally {
      setLoadingInvitations(false);
    }
  };

  const handleResend = async (invitationId: string) => {
    setError('');
    setSuccess('');
    setSavingInvitation(`resend:${invitationId}`);

    try {
      await userService.resendInvitation(invitationId);
      setSuccess('Invitation resent successfully');
      await loadInvitations();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to resend invitation');
    } finally {
      setSavingInvitation(null);
    }
  };

  const handleRevoke = async (invitationId: string) => {
    if (!window.confirm('Revoke this invitation?')) {
      return;
    }

    setError('');
    setSuccess('');
    setSavingInvitation(`revoke:${invitationId}`);

    try {
      await userService.revokeInvitation(invitationId);
      setSuccess('Invitation revoked successfully');
      await loadInvitations();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to revoke invitation');
    } finally {
      setSavingInvitation(null);
    }
  };

  const handleUpdateMember = async (
    member: TeamMember,
    data: TeamMemberUpdateRequest,
    action: MemberAction
  ) => {
    setError('');
    setSuccess('');
    setSavingMember(`${action}:${member.id}`);

    try {
      await userService.updateTeamMember(member.id, data);
      setSuccess('Team member updated successfully');
      await loadTeamMembers();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to update team member');
    } finally {
      setSavingMember(null);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setLoading(true);

    try {
      await userService.inviteUser({ email, role });
      setSuccess(`Invitation sent successfully to ${email}`);
      setEmail('');
      setRole(ROLES.CASHIER);
      await loadInvitations();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to send invitation');
    } finally {
      setLoading(false);
    }
  };

  const canManageMember = (member: TeamMember) => {
    if (!user || member.id === user.id) {
      return false;
    }

    if (user.role === ROLES.OWNER) {
      return member.role === ROLES.MANAGER || member.role === ROLES.CASHIER;
    }

    return user.role === ROLES.MANAGER && member.role === ROLES.CASHIER;
  };

  const getRoleColor = (role: string) => {
    switch (role) {
      case ROLES.OWNER:
        return 'bg-purple-100 text-purple-800';
      case ROLES.ADMIN:
        return 'bg-blue-100 text-blue-800';
      case ROLES.MANAGER:
        return 'bg-green-100 text-green-800';
      case ROLES.CASHIER:
        return 'bg-gray-100 text-gray-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const getMemberStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-100 text-green-800';
      case 'suspended':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900">Team Management</h1>
          <p className="mt-2 text-sm text-gray-600">
            Invite teammates and manage accepted team accounts
          </p>
        </div>

        <div className="bg-white rounded-lg shadow mb-6">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-xl font-semibold text-gray-900">Send Invitation</h2>
          </div>
          <form onSubmit={handleSubmit} className="p-6">
            {error && (
              <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
                {error}
              </div>
            )}
            {success && (
              <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg text-green-700">
                {success}
              </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-2">
                  Email Address
                </label>
                <input
                  type="email"
                  id="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                  placeholder="colleague@example.com"
                  required
                />
              </div>

              <div>
                <label htmlFor="role" className="block text-sm font-medium text-gray-700 mb-2">
                  Role
                </label>
                <select
                  id="role"
                  value={role}
                  onChange={(e) => setRole(e.target.value as Role)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                  required
                >
                  <option value={ROLES.CASHIER}>Cashier</option>
                  <option value={ROLES.MANAGER}>Manager</option>
                </select>
              </div>
            </div>

            <div className="mt-6 flex items-center justify-between">
              <button
                type="button"
                onClick={() => router.push('/dashboard')}
                className="px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading}
                className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? 'Sending...' : 'Send Invitation'}
              </button>
            </div>
          </form>
        </div>

        <div className="bg-white rounded-lg shadow mb-6">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-xl font-semibold text-gray-900">Accepted Team Members</h2>
          </div>
          <div className="p-6">
            {loadingTeamMembers ? (
              <LoadingState label="Loading team members..." />
            ) : teamMembers.length === 0 ? (
              <EmptyState label="No accepted team members yet" />
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <TableHeader>Name</TableHeader>
                      <TableHeader>Email</TableHeader>
                      <TableHeader>Role</TableHeader>
                      <TableHeader>Status</TableHeader>
                      <TableHeader>Joined</TableHeader>
                      <TableHeader>Actions</TableHeader>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {teamMembers.map((member) => {
                      const canManage = canManageMember(member);
                      const nextStatus = member.status === 'active' ? 'suspended' : 'active';
                      const name = [member.firstName, member.lastName].filter(Boolean).join(' ');
                      const roleSaving = savingMember === `role:${member.id}`;
                      const statusSaving = savingMember === `status:${member.id}`;
                      const memberSaving = roleSaving || statusSaving;
                      const statusActionLabel = member.status === 'active' ? 'Deactivate' : 'Activate';
                      const statusSavingLabel =
                        member.status === 'active' ? 'Deactivating...' : 'Activating...';

                      return (
                        <tr key={member.id}>
                          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                            {name || '-'}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">
                            {member.email}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            {canManage ? (
                              <div className="flex min-w-[9rem] flex-col gap-1">
                                <select
                                  aria-label={`Change role for ${member.email}`}
                                  value={member.role}
                                  disabled={memberSaving}
                                  onChange={(e) =>
                                    handleUpdateMember(
                                      member,
                                      { role: e.target.value as 'manager' | 'cashier' },
                                      'role'
                                    )
                                  }
                                  className="rounded-md border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 focus:ring-2 focus:ring-primary-500 focus:border-transparent disabled:bg-gray-100 disabled:cursor-not-allowed"
                                >
                                  <option value={ROLES.CASHIER}>Cashier</option>
                                  <option value={ROLES.MANAGER}>Manager</option>
                                </select>
                                {roleSaving && <InlineLoading label="Saving role..." />}
                              </div>
                            ) : (
                              <RoleBadge role={member.role} className={getRoleColor(member.role)} />
                            )}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span
                              className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getMemberStatusColor(member.status)}`}
                            >
                              {member.status.toUpperCase()}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {new Date(member.createdAt).toLocaleDateString()}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {canManage ? (
                              <button
                                type="button"
                                disabled={memberSaving}
                                onClick={() =>
                                  handleUpdateMember(member, { status: nextStatus }, 'status')
                                }
                                className="inline-flex min-w-[7.5rem] items-center gap-2 text-primary-600 hover:text-primary-900 font-medium disabled:text-gray-400 disabled:cursor-not-allowed"
                              >
                                {statusSaving && <Spinner className="h-3.5 w-3.5 border-gray-400" />}
                                {statusSaving ? statusSavingLabel : statusActionLabel}
                              </button>
                            ) : (
                              <span className="text-gray-400">Restricted</span>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>

        <div className="bg-white rounded-lg shadow">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-xl font-semibold text-gray-900">Pending Invitations</h2>
          </div>
          <div className="p-6">
            {loadingInvitations ? (
              <LoadingState label="Loading invitations..." />
            ) : pendingInvitations.length === 0 ? (
              <EmptyState label="No pending invitations" />
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <TableHeader>Email</TableHeader>
                      <TableHeader>Role</TableHeader>
                      <TableHeader>Sent</TableHeader>
                      <TableHeader>Expires</TableHeader>
                      <TableHeader>Actions</TableHeader>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {pendingInvitations.map((invitation) => (
                      <tr key={invitation.id}>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                          {invitation.email}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <RoleBadge
                            role={invitation.role}
                            className={getRoleColor(invitation.role)}
                          />
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {new Date(invitation.createdAt).toLocaleDateString()}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {new Date(invitation.expiresAt).toLocaleDateString()}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                          <div className="flex items-center gap-4">
                            <button
                              type="button"
                              disabled={savingInvitation === `resend:${invitation.id}`}
                              onClick={() => handleResend(invitation.id)}
                              className="text-primary-600 hover:text-primary-900 font-medium disabled:text-gray-400 disabled:cursor-not-allowed"
                            >
                              {savingInvitation === `resend:${invitation.id}` ? 'Sending...' : 'Resend'}
                            </button>
                            <button
                              type="button"
                              disabled={savingInvitation === `revoke:${invitation.id}`}
                              onClick={() => handleRevoke(invitation.id)}
                              className="text-red-600 hover:text-red-900 font-medium disabled:text-gray-400 disabled:cursor-not-allowed"
                            >
                              {savingInvitation === `revoke:${invitation.id}` ? 'Revoking...' : 'Revoke'}
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}

function TableHeader({ children }: { children: React.ReactNode }) {
  return (
    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
      {children}
    </th>
  );
}

function RoleBadge({ role, className }: { role: string; className: string }) {
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${className}`}>
      {role.toUpperCase()}
    </span>
  );
}

function LoadingState({ label }: { label: string }) {
  return (
    <div className="text-center py-8">
      <Spinner className="h-8 w-8 border-primary-600 mx-auto" />
      <p className="mt-2 text-gray-600">{label}</p>
    </div>
  );
}

function InlineLoading({ label }: { label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-xs font-medium text-gray-500">
      <Spinner className="h-3.5 w-3.5 border-gray-400" />
      {label}
    </span>
  );
}

function Spinner({ className = '' }: { className?: string }) {
  return (
    <span
      aria-hidden="true"
      className={`inline-block animate-spin rounded-full border-2 border-t-transparent ${className}`}
    />
  );
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="text-center py-8 text-gray-500">
      <svg
        className="mx-auto h-12 w-12 text-gray-400"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
        />
      </svg>
      <p className="mt-2">{label}</p>
    </div>
  );
}
