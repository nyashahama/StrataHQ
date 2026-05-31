'use client'
import { createContext, useContext, useState, useCallback, useEffect, useRef, type ReactNode } from 'react'

interface Toast {
  id: string
  message: string
  type: 'success' | 'info' | 'error'
}

interface ToastContextValue {
  addToast: (message: string, type?: Toast['type']) => void
}

const ToastContext = createContext<ToastContextValue>({ addToast: () => {} })

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const dismissalTimers = useRef<Set<ReturnType<typeof setTimeout>>>(new Set())
  const nextToastID = useRef(0)

  const addToast = useCallback((message: string, type: Toast['type'] = 'success') => {
    const id = `${nextToastID.current++}-${Date.now()}`
    setToasts(prev => [...prev, { id, message, type }])
    const timer = setTimeout(() => {
      dismissalTimers.current.delete(timer)
      setToasts(prev => prev.filter(t => t.id !== id))
    }, 3000)
    dismissalTimers.current.add(timer)
  }, [])

  useEffect(() => () => {
    for (const timer of dismissalTimers.current) {
      clearTimeout(timer)
    }
    dismissalTimers.current.clear()
  }, [])

  return (
    <ToastContext.Provider value={{ addToast }}>
      {children}
      <div className="fixed bottom-5 right-5 z-[60] flex flex-col gap-2 pointer-events-none">
        {toasts.map(toast => (
          <div
            key={toast.id}
            className={[
              'px-4 py-3 rounded-lg shadow-lg text-[13px] font-medium pointer-events-auto',
              toast.type === 'success' ? 'bg-green-bg border border-green/20 text-green' : '',
              toast.type === 'info'    ? 'bg-accent-bg border border-accent/20 text-accent' : '',
              toast.type === 'error'   ? 'bg-red-bg border border-red/20 text-red' : '',
            ].join(' ')}
          >
            {toast.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  return useContext(ToastContext)
}
