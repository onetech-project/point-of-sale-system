'use client';

import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import DashboardLayout from '@/src/components/layout/DashboardLayout';
import { 
  getRetentionPolicies, 
  updateRetentionPolicy, 
  RetentionPolicy, 
  UpdateRetentionPolicyRequest 
} from '@/services/retention';

export default function RetentionPoliciesPage() {
  const { t } = useTranslation();
  const [policies, setPolicies] = useState<RetentionPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<string | null>(null);
  const [formData, setFormData] = useState<UpdateRetentionPolicyRequest>({});
  const [error, setError] = useState<string | null>(null);

  // Load retention policies
  useEffect(() => {
    loadPolicies();
  }, []);

  const loadPolicies = async () => {
    try {
      setLoading(true);
      const data = await getRetentionPolicies();
      setPolicies(data);
    } catch (err) {
      console.error('Failed to load retention policies:', err);
      setError('Failed to load retention policies');
    } finally {
      setLoading(false);
    }
  };

  // Start editing a policy
  const startEdit = (policy: RetentionPolicy) => {
    setEditing(policy.id);
    setFormData({
      retention_period_days: policy.retention_period_days,
      grace_period_days: policy.grace_period_days,
      cleanup_method: policy.cleanup_method,
      notification_days_before: policy.notification_days_before,
      is_active: policy.is_active,
    });
  };

  // Cancel editing
  const cancelEdit = () => {
    setEditing(null);
    setFormData({});
  };

  // Save policy changes
  const savePolicy = async (policyId: string) => {
    try {
      // Validate retention period against legal minimum
      const policy = policies.find(p => p.id === policyId);
      if (policy && formData.retention_period_days !== undefined) {
        if (formData.retention_period_days < policy.legal_minimum_days) {
          alert(
            `⚠️ Legal Compliance Violation\n\n` +
            `Retention period (${formData.retention_period_days} days) cannot be less than legal minimum (${policy.legal_minimum_days} days).\n\n` +
            `This would violate:\n` +
            `• Indonesian Tax Law: 5 years (1825 days) for financial records\n` +
            `• UU PDP Article 56: 7 years (2555 days) for audit trails\n\n` +
            `Please increase the retention period to meet legal requirements.`
          );
          return;
        }
      }

      await updateRetentionPolicy(policyId, formData);
      await loadPolicies();
      setEditing(null);
      setFormData({});
      setError(null);
    } catch (err) {
      console.error('Failed to update policy:', err);
      setError('Failed to update policy');
    }
  };

  // Format table/record type display
  const formatPolicyName = (policy: RetentionPolicy): string => {
    if (policy.record_type) {
      return `${policy.table_name} (${policy.record_type})`;
    }
    return policy.table_name;
  };

  // Format cleanup method display
  const formatCleanupMethod = (method: string): string => {
    const methods: Record<string, string> = {
      soft_delete: 'Soft Delete',
      hard_delete: 'Hard Delete',
      anonymize: 'Anonymize',
    };
    return methods[method] || method;
  };

  if (loading) {
    return (
      <DashboardLayout>
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-gray-800">Data Retention Policies</h1>
          <p className="text-gray-600 mt-2">
            Manage automated data cleanup and retention periods for UU PDP compliance
          </p>
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg">
            {error}
          </div>
        )}

        <div className="bg-white rounded-lg shadow overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Table / Type
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Retention Period (Days)
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Legal Minimum
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Cleanup Method
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Grace Period
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Notification
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {policies.map((policy) => (
                  <tr key={policy.id} className={editing === policy.id ? 'bg-blue-50' : ''}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">
                        {formatPolicyName(policy)}
                      </div>
                      <div className="text-xs text-gray-500">{policy.retention_field}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {editing === policy.id ? (
                        <input
                          type="number"
                          min={policy.legal_minimum_days}
                          value={formData.retention_period_days ?? policy.retention_period_days}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              retention_period_days: parseInt(e.target.value),
                            })
                          }
                          className="w-24 px-2 py-1 border border-gray-300 rounded text-sm"
                        />
                      ) : (
                        <span className="text-sm text-gray-900">{policy.retention_period_days}</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="text-sm text-gray-500">{policy.legal_minimum_days}</span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {editing === policy.id ? (
                        <select
                          value={formData.cleanup_method ?? policy.cleanup_method}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              cleanup_method: e.target.value as any,
                            })
                          }
                          className="px-2 py-1 border border-gray-300 rounded text-sm"
                        >
                          <option value="soft_delete">Soft Delete</option>
                          <option value="hard_delete">Hard Delete</option>
                          <option value="anonymize">Anonymize</option>
                        </select>
                      ) : (
                        <span className="text-sm text-gray-900">
                          {formatCleanupMethod(policy.cleanup_method)}
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {editing === policy.id ? (
                        <input
                          type="number"
                          min={0}
                          value={formData.grace_period_days ?? policy.grace_period_days ?? ''}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              grace_period_days: e.target.value ? parseInt(e.target.value) : null,
                            })
                          }
                          placeholder="None"
                          className="w-20 px-2 py-1 border border-gray-300 rounded text-sm"
                        />
                      ) : (
                        <span className="text-sm text-gray-500">
                          {policy.grace_period_days ?? '-'}
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {editing === policy.id ? (
                        <input
                          type="number"
                          min={0}
                          value={
                            formData.notification_days_before ?? policy.notification_days_before ?? ''
                          }
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              notification_days_before: e.target.value
                                ? parseInt(e.target.value)
                                : null,
                            })
                          }
                          placeholder="None"
                          className="w-20 px-2 py-1 border border-gray-300 rounded text-sm"
                        />
                      ) : (
                        <span className="text-sm text-gray-500">
                          {policy.notification_days_before ?? '-'}
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {editing === policy.id ? (
                        <select
                          value={formData.is_active !== undefined ? String(formData.is_active) : String(policy.is_active)}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              is_active: e.target.value === 'true',
                            })
                          }
                          className="px-2 py-1 border border-gray-300 rounded text-sm"
                        >
                          <option value="true">Active</option>
                          <option value="false">Inactive</option>
                        </select>
                      ) : (
                        <span
                          className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                            policy.is_active
                              ? 'bg-green-100 text-green-800'
                              : 'bg-gray-100 text-gray-800'
                          }`}
                        >
                          {policy.is_active ? 'Active' : 'Inactive'}
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      {editing === policy.id ? (
                        <div className="space-x-2">
                          <button
                            onClick={() => savePolicy(policy.id)}
                            className="text-blue-600 hover:text-blue-900"
                          >
                            Save
                          </button>
                          <button
                            onClick={cancelEdit}
                            className="text-gray-600 hover:text-gray-900"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => startEdit(policy)}
                          className="text-indigo-600 hover:text-indigo-900"
                        >
                          Edit
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <h3 className="text-sm font-medium text-blue-800 mb-2">UU PDP Compliance Notice</h3>
          <p className="text-sm text-blue-700">
            Retention policies enforce data minimization (UU PDP Article 5) and ensure personal
            data is not kept longer than necessary. Legal minimums are enforced to comply with
            Indonesian tax law (5 years) and audit trail requirements (7 years).
          </p>
        </div>
      </div>
    </DashboardLayout>
  );
}
