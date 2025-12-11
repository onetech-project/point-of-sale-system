"use client"

import { useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { useToasts } from '../components/ui/Toast'
import { useSSE } from '../services/sse'

type NotificationEventData = {
  id?: string
  tenant_id?: string
  title?: string
  body?: string
  message?: string
  data?: any
  timestamp?: string
  type?: string
}

export default function useSSENotifications() {
  const router = useRouter()
  const { push } = useToasts()

  const handleEvent = useCallback((ev: { id?: string; event?: string; data?: any }) => {
    if (!ev || ev.data == null) return
    // ev.data may be a string (raw text) or an object
    const raw = ev.data
    let payload: NotificationEventData | null = null
    if (typeof raw === 'string') {
      try {
        payload = JSON.parse(raw)
      } catch (_e) {
        // non-JSON payload — ignore
        return
      }
    } else if (typeof raw === 'object') {
      payload = raw as NotificationEventData
    }
    if (!payload) return
    // determine event name: prefer SSE event field if present
    const eventName = ev.event || payload.type || ''

    // Handle aggregated order.paid events
    if (eventName === 'order.paid.aggregate') {
      // payload may be the wrapper or contain count directly
      const count = (payload as any).count ?? payload.data?.count ?? 0
      const summary = payload.body || payload.message || payload.data?.summary || `You have ${count} paid order${count !== 1 ? 's' : ''}`
      push({
        title: 'Order updates',
        message: summary,
        actionLabel: 'VIEW',
        onAction: () => router.push('/orders'),
      })
      return
    }

    const title = payload.title || (payload.type ? payload.type.replace(/\./g, ' ') : 'Notification')
    const message = payload.message || 'You have a new notification.'
    const route = payload.data?.route

    push({
      title,
      message,
      actionLabel: route ? 'View' : undefined,
      onAction: route ? () => router.push(route) : undefined,
    })
  }, [push, router])

  // Start SSE subscription. `useSSE` will manage connection lifecycle.
  useSSE(undefined, handleEvent, { url: undefined })
}
