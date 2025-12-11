"use client"

import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

type Toast = {
  id: string
  title?: string
  message: string
  actionLabel?: string
  onAction?: () => void
  duration?: number
}

type ToastContextValue = {
  push: (t: Omit<Toast, 'id'>) => string
  dismiss: (id: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function useToasts() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToasts must be used within ToastProvider')
  return ctx
}

export const ToastProvider: React.FC<React.PropsWithChildren<Record<string, unknown>>> = ({ children }) => {
  const [toasts, setToasts] = useState<Toast[]>([])

  const push = useCallback((t: Omit<Toast, 'id'>) => {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
    const toast: Toast = { id, ...t, duration: t.duration ?? 5000 }
    setToasts((s) => [toast, ...s])
    return id
  }, [])

  const dismiss = useCallback((id: string) => {
    setToasts((s) => s.filter((t) => t.id !== id))
  }, [])

  // Auto-dismiss
  useEffect(() => {
    const timers: NodeJS.Timeout[] = []
    toasts.forEach((t) => {
      const timer = setTimeout(() => {
        setToasts((s) => s.filter((x) => x.id !== t.id))
      }, t.duration)
      timers.push(timer)
    })
    return () => timers.forEach((tt) => clearTimeout(tt))
  }, [toasts])

  const value = useMemo(() => ({ push, dismiss }), [push, dismiss])

  return (
    <ToastContext.Provider value={value}>
      {children}

      {/* Toast container */}
      <div className="fixed top-4 right-4 z-50 flex flex-col gap-3 items-end">
        {toasts.map((t) => (
          <div key={t.id} className="max-w-sm w-full bg-slate-800 text-white rounded-lg shadow-lg p-3 flex items-start gap-3">
            <div className="flex-1">
              {t.title && <div className="font-semibold text-sm">{t.title}</div>}
              <div className="text-sm text-slate-200">{t.message}</div>
            </div>
            <div className="flex items-center gap-2">
              {t.actionLabel && (
                <button
                  onClick={() => {
                    try {
                      t.onAction && t.onAction()
                    } finally {
                      dismiss(t.id)
                    }
                  }}
                  className="text-sm text-blue-400 hover:underline"
                >
                  {t.actionLabel}
                </button>
              )}
              <button onClick={() => dismiss(t.id)} className="text-slate-400 hover:text-white text-sm">✕</button>
            </div>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export default ToastProvider
