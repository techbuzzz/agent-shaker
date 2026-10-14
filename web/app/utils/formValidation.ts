/**
 * Form validation rules.
 *
 * The app has no schema library, so this stays deliberately small: rules are
 * plain functions over the form object, and a validator is just a function
 * that returns one message per bad field. Nothing here knows about Vue, which
 * keeps it unit-testable and keeps the components boring.
 *
 * Messages are written for the person filling the form — "Give the project a
 * name", not "name is required" — because a field-level message is the only
 * place the user gets to fix the mistake.
 */

export type FormErrors<T> = Partial<Record<keyof T, string>>

/** Validates a whole form. Returns one message per offending field. */
export type FormValidator<T> = (values: Readonly<T>) => FormErrors<T>

/** Reads a value as a trimmed string; null/undefined become ''. */
export function text(value: unknown): string {
  return value == null ? '' : String(value).trim()
}

/** A non-empty value is required. `label` is the field's human name. */
export function required<T extends object, K extends keyof T>(
  field: K,
  label: string
): FormValidator<T> {
  return (values) => (text(values[field]) === '' ? { [field]: `${label} is required` } as FormErrors<T> : {})
}

/** Caps the length of a free-text field. */
export function maxLength<T extends object, K extends keyof T>(
  field: K,
  limit: number,
  label: string = 'This value'
): FormValidator<T> {
  return (values) => {
    const value = text(values[field])
    return value.length > limit ? { [field]: `${label} must be ${limit} characters or fewer` } as FormErrors<T> : {}
  }
}

/** Combines validators, keeping the first message per field. */
export function compose<T extends object>(
  ...validators: Array<FormValidator<T>>
): FormValidator<T> {
  return (values) => validators.reduce<FormErrors<T>>((acc, validator) => {
    const next = validator(values)
    for (const [field, message] of Object.entries(next) as Array<[keyof T, string]>) {
      if (!acc[field]) acc[field] = message
    }
    return acc
  }, {})
}

/**
 * A rule for "pick one of these options" fields, where the empty string means
 * "nothing chosen" rather than "empty input".
 */
export function requiredChoice<T extends object, K extends keyof T>(
  field: K,
  label: string
): FormValidator<T> {
  return (values) => (text(values[field]) === '' ? { [field]: `Choose ${label.toLowerCase()}` } as FormErrors<T> : {})
}