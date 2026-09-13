/**
 * Thin wrapper over Nuxt UI's `useToast` that gives the rest of the app
 * a stable surface (info / success / warn / error / toast for WS events).
 */

type ToastInput = Parameters<ReturnType<typeof useToast>>['add'] extends (input: infer T) => unknown ? T : never

export interface UseToastBus {
  success: (message: string, opts?: Partial<ToastInput>) => void
  info:    (message: string, opts?: Partial<ToastInput>) => void
  warn:    (message: string, opts?: Partial<ToastInput>) => void
  error:   (message: string, opts?: Partial<ToastInput>) => void
  add:     (message: string, opts?: Partial<ToastInput>) => void
}

export function useToastBus(): UseToastBus {
  const toast = useToast()

  function emit(color: ToastInput['color'] | undefined, message: string, opts?: Partial<ToastInput>) {
    toast.add({
      title: opts?.title ?? message,
      description: opts?.description,
      color,
      icon: opts?.icon,
      timeout: opts?.timeout ?? 4000
    } as ToastInput)
  }

  return {
    success: (m, o) => emit('success', m, o),
    info:    (m, o) => emit('info',    m, o),
    warn:    (m, o) => emit('warning', m, o),
    error:   (m, o) => emit('error',   m, o),
    add:     (m, o) => emit(undefined, m, o)
  }
}
