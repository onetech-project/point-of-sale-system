'use client';

import React, { useEffect, useState } from 'react';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import { tenantService } from '@/services/tenant';
import { TenantData } from '@/types/tenant';

export default function TenantDataPage() {
  const [data, setData] = useState<TenantData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchTenantData();
  }, []);

  const fetchTenantData = async () => {
    try {
      const result = await tenantService.getAllTenantData();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const handleExport = async () => {
    try {
      const blob = await tenantService.exportTenantData();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `tenant-data-${Date.now()}.json`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      alert('Export failed: ' + (err instanceof Error ? err.message : 'Unknown error'));
    }
  };

  const handleDeleteUser = async (userId: string, force: boolean) => {
    if (!confirm(`Are you sure you want to ${force ? 'permanently delete' : 'soft delete'} this user?`)) {
      return;
    }

    try {
      const result = await tenantService.deleteUser(userId, force);
      alert(result.message);
      fetchTenantData();
    } catch (err) {
      alert('Delete failed: ' + (err instanceof Error ? err.message : 'Unknown error'));
    }
  };

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER]}>
      <DashboardLayout>
        <div className="max-w-6xl mx-auto p-6">
          <div className="flex justify-between items-center mb-6">
            <h1 className="text-3xl font-bold">Tenant Data Management</h1>
            <button
              onClick={handleExport}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              Export All Data
            </button>
          </div>

          {loading && <div className="text-center py-8">Loading...</div>}
          {error && <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">{error}</div>}

          {data && (
            <div className="space-y-6">
              {/* Business Profile */}
              <section className="bg-white rounded-lg shadow p-6">
                <h2 className="text-xl font-semibold mb-4">Business Profile</h2>
                <dl className="grid grid-cols-2 gap-4">
                  <div>
                    <dt className="font-medium text-gray-600">Business Name</dt>
                    <dd className="mt-1">{data.tenant.business_name}</dd>
                  </div>
                  <div>
                    <dt className="font-medium text-gray-600">Slug</dt>
                    <dd className="mt-1">{data.tenant.slug}</dd>
                  </div>
                  <div>
                    <dt className="font-medium text-gray-600">Status</dt>
                    <dd className="mt-1">
                      <span className={`px-2 py-1 rounded text-sm ${data.tenant.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                        }`}>
                        {data.tenant.status}
                      </span>
                    </dd>
                  </div>
                  <div>
                    <dt className="font-medium text-gray-600">Created At</dt>
                    <dd className="mt-1">{new Date(data.tenant.created_at).toLocaleDateString()}</dd>
                  </div>
                </dl>
              </section>

              {/* Team Members */}
              <section className="bg-white rounded-lg shadow p-6">
                <h2 className="text-xl font-semibold mb-4">Team Members ({data.team_members.length})</h2>
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead>
                      <tr>
                        <th className="px-4 py-2 text-left text-sm font-medium text-gray-700">Name</th>
                        <th className="px-4 py-2 text-left text-sm font-medium text-gray-700">Email</th>
                        <th className="px-4 py-2 text-left text-sm font-medium text-gray-700">Role</th>
                        <th className="px-4 py-2 text-left text-sm font-medium text-gray-700">Status</th>
                        <th className="px-4 py-2 text-left text-sm font-medium text-gray-700">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200">
                      {data.team_members.map((member) => (
                        <tr key={member.id}>
                          <td className="px-4 py-3">
                            {member.first_name || member.last_name
                              ? `${member.first_name || ''} ${member.last_name || ''}`.trim()
                              : '-'}
                          </td>
                          <td className="px-4 py-3">{member.email}</td>
                          <td className="px-4 py-3">
                            <span className="px-2 py-1 bg-blue-100 text-blue-800 rounded text-sm">
                              {member.role}
                            </span>
                          </td>
                          <td className="px-4 py-3">{member.status}</td>
                          <td className="px-4 py-3 space-x-2">
                            {member.role !== 'owner' && (
                              <>
                                <button
                                  onClick={() => handleDeleteUser(member.id, false)}
                                  className="text-sm px-2 py-1 bg-yellow-100 text-yellow-800 rounded hover:bg-yellow-200"
                                >
                                  Soft Delete
                                </button>
                                <button
                                  onClick={() => handleDeleteUser(member.id, true)}
                                  className="text-sm px-2 py-1 bg-red-100 text-red-800 rounded hover:bg-red-200"
                                >
                                  Hard Delete
                                </button>
                              </>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </section>

              {/* Configuration */}
              <section className="bg-white rounded-lg shadow p-6">
                <h2 className="text-xl font-semibold mb-4">Configuration</h2>
                <dl className="space-y-3">
                  <div>
                    <dt className="font-medium text-gray-600">Delivery Types</dt>
                    <dd className="mt-1">{data.configuration.enabled_delivery_types.join(', ')}</dd>
                  </div>
                  <div>
                    <dt className="font-medium text-gray-600">Payment Integration</dt>
                    <dd className="mt-1">
                      {data.configuration.midtrans_configured ? (
                        <span className="text-green-600">Midtrans Configured ({data.configuration.midtrans_environment})</span>
                      ) : (
                        <span className="text-gray-600">Not Configured</span>
                      )}
                    </dd>
                  </div>
                </dl>
              </section>
            </div>
          )}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
