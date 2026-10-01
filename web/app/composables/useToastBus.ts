/**
 * Thin wrapper over Nuxt UI's `useToast` that gives the rest of the app
 * a stable surface (info / success / warn / error / toast for WS events).
 */

/**
 * The input accepted by `useToast().add`, which is `Partial<Toast>`.
 *
 * Derived from the *method* rather than the API object: `Parameters<T>` needs a
 * function type, so `Parameters<ReturnType<typeof useToast>>` never worked —
 * it collapsed to `unknown` and silently poisoned every `opts` field below.
 */
type ToastApi = ReturnType<typeof useToast>
type ToastInput = Parameters<ToastApi['add']>[0]
type ToastColor = ToastInput['color']

export interface UseToastBus {
  success: (message: string, opts?: Partial<ToastInput>) => void
  info:    (message: string, opts?: Partial<ToastInput>) => void
  warn:    (message: string, opts?: Partial<ToastInput>) => void
  error:   (message: string, opts?: Partial<ToastInput>) => void
  add:     (message: string, opts?: Partial<ToastInput>) => void
}

export function useToastBus(): UseToastBus {
  const toast = useToast()

  function emit(color: ToastColor, message: string, opts?: Partial<ToastInput>) {
    toast.add({
      title: opts?.title ?? message,
      description: opts?.description,
      color,
      icon: opts?.icon,
      // Nuxt UI v3 renamed `timeout` to `duration` (ToastProps picks it from
      // reka-ui's ToastRootProps). Passing `timeout` here was silently ignored
      // and every toast fell back to the default duration.
      duration: opts?.duration ?? 4000
    })
  }

  return {
    success: (m, o) => emit('success', m, o),
    info:    (m, o) => emit('info',    m, o),
    warn:    (m, o) => emit('warning', m, o),
    error:   (m, o) => emit('error',   m, o),
    add:     (m, o) => emit(undefined, m, o)
  }
}
