import type { FormErrors, FormValidator } from '~/utils/formValidation'

/**
 * Submission state shared by every form modal: validation, the in-flight flag,
 * and one error surface that survives a failed request.
 *
 * Why the handler is a prop rather than an event: the modals used to do
 * `await emit('submit', payload)` and close. `emit` returns `undefined`, so the
 * `await` resolved on the next tick — the dialog closed while the request was
 * still in flight and whatever came back (401, 409, 500) went nowhere. Calling
 * a prop as a function is the only version where the dialog can actually wait
 * for the answer and report it.
 */
export interface UseFormModalOptions<T extends object> {
  /** The live form object. Mutated by the inputs; read by the validator. */
  form: T
  validate: FormValidator<T>
  /** Called on submit with a copy of the form. Must reject on failure. */
  onSubmit: (payload: T) => Promise<unknown>
  /** Closes the dialog after a successful submit. */
  close: () => void
  /** Copy for the banner when the request itself fails. */
  failureMessage?: string
}

export interface UseFormModal<T extends object> {
  errors: ComputedRef<FormErrors<T>>
  submitting: Ref<boolean>
  submitError: Ref<string | null>
  /** Runs validation, then the handler. Returns true when the write landed. */
  submit: () => Promise<boolean>
}

/**
 * Turns the backend's error envelope into one sentence a person can act on.
 * The Go service answers `{"error":{"code","message","request_id"}}`
 * (internal/httpx/errors.go); $fetch wraps it on `err.data`.
 */
export function describeSubmitError(err: unknown, fallback: string): string {
  const candidate = err as { data?: { error?: { message?: string } }; message?: string; statusCode?: number }
  const message = candidate?.data?.error?.message ?? candidate?.message
  if (message && !/^\[?(POST|GET|PUT|DELETE)/i.test(message)) return message
  if (candidate?.statusCode) return `${fallback} (HTTP ${candidate.statusCode})`
  return fallback
}

export function useFormModal<T extends object>(
  options: UseFormModalOptions<T>
): UseFormModal<T> {
  const { form, validate, onSubmit, close, failureMessage = 'The server rejected this change' } = options

  const touched = ref(false)
  const submitting = ref(false)
  const submitError = ref<string | null>(null)

  const errors = computed(() => (touched.value ? validate(form) : {}))

  async function submit(): Promise<boolean> {
    submitError.value = null
    touched.value = true

    const current = validate(form)
    if (Object.keys(current).length > 0) {
      focusFirstInvalid(current)
      return false
    }

    submitting.value = true
    try {
      await onSubmit({ ...form } as T)
      close()
      return true
    } catch (err: unknown) {
      submitError.value = describeSubmitError(err, failureMessage)
      return false
    } finally {
      submitting.value = false
    }
  }

  return { errors, submitting, submitError, submit }
}

/**
 * Moves focus to the first control that failed validation, so keyboard and
 * screen-reader users are not left at the top of a form they cannot see the
 * problem in.
 */
function focusFirstInvalid(errors: FormErrors<object>) {
  if (typeof document === 'undefined') return
  nextTick(() => {
    const firstField = Object.keys(errors)[0]
    if (!firstField) return
    const root = document.querySelector('[role="dialog"]') ?? document
    const control = root.querySelector<HTMLElement>(`[name="${firstField}"], #${CSS.escape(firstField)}`)
    control?.focus()
  })
}