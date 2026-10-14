/**
 * The "nothing selected" value for `<USelect>`.
 *
 * Reka UI (and therefore Nuxt UI v4) refuses a `SelectItem` whose value is the
 * empty string, and logs
 *   "A <SelectItem /> must have a value prop that is not an empty string"
 * while rendering. Passing `''` for an optional field is the natural thing to
 * write, so this got written in six places — and each one threw on open.
 *
 * A sentinel that can never collide with a real UUID is the fix that keeps the
 * bound model a plain string. Convert it back to `undefined` at the point where
 * the value leaves the form; `optionalId` does exactly that.
 */
export const NO_SELECTION = '__none__' as const

/** True when a bound select value means "no choice made". */
export function isNone(value: unknown): boolean {
  return value === '' || value === null || value === undefined || value === NO_SELECTION
}

/** An optional id from a select: the sentinel (or an empty value) becomes undefined. */
export function optionalId(value: unknown): string | undefined {
  return isNone(value) ? undefined : String(value)
}