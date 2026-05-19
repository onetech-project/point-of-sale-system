'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { PlatformShell } from '@/components/platform/PlatformShell';
import { platformService, PlatformTicket } from '@/services/platform';

function statusClass(status: string): string {
  if (status === 'resolved' || status === 'closed') return 'bg-green-100 text-green-700';
  if (status === 'in_progress') return 'bg-blue-100 text-blue-700';
  if (status === 'waiting_on_tenant') return 'bg-yellow-100 text-yellow-700';
  return 'bg-gray-100 text-gray-700';
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' });
}

export default function PlatformTicketsPage() {
  const [tickets, setTickets] = useState<PlatformTicket[]>([]);
  const [status, setStatus] = useState('');
  const [noteByTicket, setNoteByTicket] = useState<Record<string, string>>({});

  const load = () => {
    platformService.listTickets({ status }).then(response => setTickets(response.tickets));
  };

  useEffect(() => {
    load();
  }, []);

  const updateStatus = async (ticket: PlatformTicket, nextStatus: string) => {
    await platformService.updateTicket(ticket.id, { status: nextStatus });
    load();
  };

  const addNote = async (ticket: PlatformTicket) => {
    const body = noteByTicket[ticket.id]?.trim();
    if (!body) return;
    await platformService.addTicketNote(ticket.id, body);
    setNoteByTicket(current => ({ ...current, [ticket.id]: '' }));
    load();
  };

  return (
    <PlatformShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Support Tickets</h1>
          <p className="mt-1 text-sm text-gray-500">Billing and account requests from tenants.</p>
        </div>

        <div className="flex gap-3 rounded-xl border border-gray-200 bg-white p-4">
          <select value={status} onChange={event => setStatus(event.target.value)} className="rounded-lg border border-gray-300 px-3 py-2 text-sm">
            <option value="">All statuses</option>
            <option value="open">Open</option>
            <option value="in_progress">In progress</option>
            <option value="waiting_on_tenant">Waiting on tenant</option>
            <option value="resolved">Resolved</option>
            <option value="closed">Closed</option>
          </select>
          <button onClick={load} className="rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white">
            Filter
          </button>
        </div>

        <div className="space-y-4">
          {tickets.length === 0 ? (
            <div className="rounded-xl border border-gray-200 bg-white p-8 text-center text-sm text-gray-500">
              No tickets found.
            </div>
          ) : (
            tickets.map(ticket => (
              <div key={ticket.id} className="rounded-xl border border-gray-200 bg-white p-5">
                <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <h2 className="font-semibold text-gray-900">{ticket.subject}</h2>
                      <span className={`rounded-full px-2 py-1 text-xs font-semibold uppercase ${statusClass(ticket.status)}`}>
                        {ticket.status}
                      </span>
                      <span className="rounded-full bg-gray-100 px-2 py-1 text-xs font-semibold uppercase text-gray-700">
                        {ticket.priority}
                      </span>
                    </div>
                    <p className="mt-1 text-sm text-gray-500">
                      <Link href={`/platform/tenants/${ticket.tenant_id}`} className="text-primary-600 hover:underline">
                        {ticket.tenant_name ?? ticket.tenant_id}
                      </Link>{' '}
                      - last activity {formatDate(ticket.last_activity_at)}
                    </p>
                    <p className="mt-3 text-sm text-gray-700">{ticket.description}</p>
                  </div>
                  <select
                    value={ticket.status}
                    onChange={event => updateStatus(ticket, event.target.value)}
                    className="w-fit rounded-lg border border-gray-300 px-3 py-2 text-sm"
                  >
                    <option value="open">Open</option>
                    <option value="in_progress">In progress</option>
                    <option value="waiting_on_tenant">Waiting on tenant</option>
                    <option value="resolved">Resolved</option>
                    <option value="closed">Closed</option>
                  </select>
                </div>
                <div className="mt-4 flex flex-col gap-3 sm:flex-row">
                  <input
                    value={noteByTicket[ticket.id] ?? ''}
                    onChange={event =>
                      setNoteByTicket(current => ({ ...current, [ticket.id]: event.target.value }))
                    }
                    placeholder="Internal note"
                    className="min-w-0 flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm"
                  />
                  <button
                    onClick={() => addNote(ticket)}
                    className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
                  >
                    Add Note
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </PlatformShell>
  );
}
