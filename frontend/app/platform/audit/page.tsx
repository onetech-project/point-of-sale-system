'use client';

import React, { useEffect, useState } from 'react';
import { ClipboardList, Filter } from 'lucide-react';
import { PlatformShell } from '@/components/platform/PlatformShell';
import { platformService, TenantActivity } from '@/services/platform';

function formatDate(value?: string): string {
  if (!value) return '-';
  return new Date(value).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' });
}

export default function PlatformAuditPage() {
  const [events, setEvents] = useState<TenantActivity[]>([]);
  const [tenantId, setTenantId] = useState('');
  const [adminId, setAdminId] = useState('');
  const [action, setAction] = useState('');
  const [loading, setLoading] = useState(true);

  const load = () => {
    setLoading(true);
    platformService
      .listAuditEvents({ tenant_id: tenantId, admin_id: adminId, action, limit: 100 })
      .then(response => setEvents(response.events))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <PlatformShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Platform Audit</h1>
          <p className="mt-1 text-sm text-gray-500">Platform admin lifecycle, billing metadata, support-owner, and note actions.</p>
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <div className="grid gap-3 md:grid-cols-4">
            <input
              value={tenantId}
              onChange={event => setTenantId(event.target.value)}
              placeholder="Tenant ID"
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
            />
            <input
              value={adminId}
              onChange={event => setAdminId(event.target.value)}
              placeholder="Admin ID"
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
            />
            <select value={action} onChange={event => setAction(event.target.value)} className="rounded-lg border border-gray-300 px-3 py-2 text-sm">
              <option value="">All actions</option>
              <option value="LOGIN">Login</option>
              <option value="CREATE">Create</option>
              <option value="UPDATE">Update</option>
              <option value="DELETE">Delete</option>
              <option value="suspend">Suspend</option>
              <option value="deactivate">Deactivate</option>
              <option value="schedule_delete">Schedule delete</option>
              <option value="reactivate">Reactivate</option>
              <option value="change_plan">Change plan</option>
              <option value="change_billing_cycle">Change billing cycle</option>
              <option value="extend_trial">Extend trial</option>
              <option value="extend_grace">Extend grace</option>
              <option value="regenerate_payment_link">Refresh payment link</option>
              <option value="add_note">Add note</option>
              <option value="assign_support_owner">Assign support owner</option>
            </select>
            <button onClick={load} className="inline-flex items-center justify-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white">
              <Filter className="h-4 w-4" aria-hidden="true" />
              Filter
            </button>
          </div>
        </div>

        <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
          <div className="flex items-center gap-2 border-b border-gray-200 px-5 py-4">
            <ClipboardList className="h-4 w-4 text-gray-500" aria-hidden="true" />
            <h2 className="font-semibold text-gray-900">Audit Events</h2>
          </div>
          <div className="divide-y divide-gray-100">
            {loading ? (
              <p className="p-8 text-center text-sm text-gray-500">Loading audit events...</p>
            ) : events.length === 0 ? (
              <p className="p-8 text-center text-sm text-gray-500">No audit events found.</p>
            ) : (
              events.map((event, index) => (
                <div key={`${event.action}-${event.occurred_at}-${index}`} className="grid gap-3 p-4 md:grid-cols-[1fr_180px]">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="font-semibold text-gray-900">{event.action.replaceAll('_', ' ')}</p>
                      {event.resource && (
                        <span className="rounded-full bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700">
                          {event.resource}
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-sm text-gray-500">
                      {event.before_status && event.after_status ? `${event.before_status} -> ${event.after_status}` : event.actor_type}
                    </p>
                    {event.reason && <p className="mt-2 text-sm text-gray-700">{event.reason}</p>}
                  </div>
                  <p className="text-sm text-gray-500 md:text-right">{formatDate(event.occurred_at)}</p>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </PlatformShell>
  );
}
