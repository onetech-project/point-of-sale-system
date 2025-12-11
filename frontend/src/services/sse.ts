import { useEffect, useRef } from 'react'

// Generic SSE event typedefs
export type EventHandler<T = any> = (event: SseEvent<T>) => void

export interface SseEvent<T = any> {
  id?: string
  event?: string
  data?: T
}

export interface NotificationEventData {
  id?: string
  tenant_id?: string
  title?: string
  body?: string
  data?: any
  timestamp?: string
}

type EventHandlerInternal = EventHandler<any>

export interface SSEOptions {
  url: string
  snapshotFn?: () => Promise<any>
}

export class SseClient {
  private url: string
  private snapshotFn?: () => Promise<any>
  private es?: EventSource
  private lastEventId?: string
  private onEvent?: EventHandler
  private onOpen?: () => void
  private onError?: (err?: any) => void
  private reconnectMs = 500
  private maxReconnectMs = 10000
  private stopped = false

  constructor(opts: SSEOptions) {
    this.url = opts.url
    this.snapshotFn = opts.snapshotFn
  }

  connect(opts?: { onEvent?: EventHandler; onOpen?: () => void; onError?: (err?: any) => void }) {
    this.onEvent = opts?.onEvent
    this.onOpen = opts?.onOpen
    this.onError = opts?.onError
    this.stopped = false
    this.open()
  }

  private async open() {
    try {
      const url = this.buildUrl(this.url, this.lastEventId)
      this.es = new EventSource(url, { withCredentials: true })
      this.es.onopen = () => {
        this.onOpen && this.onOpen()
      }
      this.es.onmessage = (ev: MessageEvent) => {
        // SSE messages may be JSON (the canonical case) or simple text heartbeats
        // (e.g. "heartbeat"). Attempt to parse JSON, but fall back to raw text
        // so the client doesn't throw on a plain-text heartbeat.
        let eventName: string | undefined
        let parsedData: any
        try {
          const data = JSON.parse(ev.data)
          eventName = (data.event as string) || undefined
          parsedData = data.data !== undefined ? data.data : data
        } catch (_err) {
          // Non-JSON payload — common when server sends simple heartbeats.
          // If it's a recognized heartbeat token, treat it as a no-op event.
          if (typeof ev.data === 'string') {
            const trimmed = ev.data.trim()
            if (trimmed === 'heartbeat' || trimmed === 'ping') {
              // ignore quiet heartbeats: don't call onEvent for them
              return
            }
            // otherwise expose raw text as the event data
            parsedData = ev.data
          } else {
            parsedData = ev.data
          }
        }

        const parsed: SseEvent = { id: ev.lastEventId || undefined, event: eventName, data: parsedData }
        if (parsed.id) this.lastEventId = parsed.id
        this.onEvent && this.onEvent(parsed)
      }
      this.es.onerror = async (err) => {
        // Try to detect closed connection and snapshot if provided
        if ((this.es && (this.es as any).readyState === 2) || !this.es) {
          // closed by server
          if (this.snapshotFn) {
            try {
              await this.snapshotFn()
            } catch (e) {
              // ignore snapshot errors
            }
          }
        }

        this.onError && this.onError(err)
        this.scheduleReconnect()
      }
    } catch (err) {
      this.onError && this.onError(err)
      this.scheduleReconnect()
    }
  }

  private scheduleReconnect() {
    if (this.stopped) return
    setTimeout(() => {
      this.reconnectMs = Math.min(this.reconnectMs * 2, this.maxReconnectMs)
      this.open()
    }, this.reconnectMs)
  }

  close() {
    this.stopped = true
    if (this.es) {
      this.es.close()
    }
  }

  private buildUrl(base: string, lastEventId?: string) {
    if (!lastEventId) return base
    const sep = base.includes('?') ? '&' : '?'
    return `${base}${sep}lastEventId=${encodeURIComponent(lastEventId)}`
  }
}



// --- subscribeToStream wrapper merged from sseService.ts ---
export interface SubscribeOpts {
  url?: string
  snapshotFn?: () => Promise<any>
}

export function subscribeToStream(lastEventId: string | undefined, onEvent: (ev: { id?: string; event?: string; data?: any }) => void, opts?: SubscribeOpts) {
  const baseUrl = opts?.url || `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/sse`

  const sse = new SseClient({
    url: baseUrl,
    snapshotFn: opts?.snapshotFn,
  })

  if (lastEventId) {
    ; (sse as any).lastEventId = lastEventId
  }

  sse.connect({
    onEvent: onEvent,
    onError: (err) => {
      console.error('SSE stream error', err)
    },
  })

  return {
    close: () => sse.close(),
  }
}

/**
 * React hook wrapper for subscribeToStream.
 * - `lastEventId`: optional last seen event id to resume from
 * - `onEvent`: callback invoked for each parsed event
 * - `opts`: optional subscribe options (url, snapshotFn, getAuthToken)
 */
export function useSSE(lastEventId: string | undefined, onEvent?: EventHandler<any>, opts?: SubscribeOpts) {
  const subRef = useRef<{ close: () => void } | null>(null)

  useEffect(() => {
    if (!onEvent) return
    const sub = subscribeToStream(lastEventId, onEvent as any, opts)
    subRef.current = sub
    return () => {
      sub.close()
      subRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lastEventId, opts?.url, opts?.snapshotFn, onEvent])

  return {
    close: () => subRef.current?.close(),
  }
}

export default SseClient
