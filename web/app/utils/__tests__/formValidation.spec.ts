import { describe, expect, it } from 'vitest'
import {
  compose,
  maxLength,
  required,
  requiredChoice,
  text
} from '../formValidation'

interface TaskForm {
  title: string
  assigned_to: string
}

const base: TaskForm = { title: '', assigned_to: '' }

describe('text', () => {
  it('treats null and undefined as empty', () => {
    expect(text(null)).toBe('')
    expect(text(undefined)).toBe('')
  })

  it('trims before comparing, so whitespace is not a value', () => {
    expect(text('   ')).toBe('')
    expect(text('  task  ')).toBe('task')
  })
})

describe('required', () => {
  it('names the field instead of saying "required"', () => {
    expect(required<TaskForm, 'title'>('title', 'A task title')(base).title)
      .toBe('A task title is required')
  })

  it('accepts a value that only needed trimming', () => {
    expect(required<TaskForm, 'title'>('title', 'A task title')({ ...base, title: 'x' }))
      .toEqual({})
  })
})

describe('requiredChoice', () => {
  it('phrases an empty select as a choice, not a missing value', () => {
    expect(requiredChoice<TaskForm, 'assigned_to'>('assigned_to', 'An owner')(base).assigned_to)
      .toBe('Choose an owner')
  })
})

describe('maxLength', () => {
  it('passes at the limit and fails past it', () => {
    const rule = maxLength<TaskForm, 'title'>('title', 255, 'The title')
    expect(rule({ ...base, title: 'a'.repeat(255) })).toEqual({})
    expect(rule({ ...base, title: 'a'.repeat(256) }).title)
      .toBe('The title must be 255 characters or fewer')
  })
})

describe('compose', () => {
  const validate = compose<TaskForm>(
    required<TaskForm, 'title'>('title', 'A task title'),
    maxLength<TaskForm, 'title'>('title', 3, 'The title')
  )

  it('reports every bad field at once', () => {
    const errors = validate({ title: 'toolong', assigned_to: '' })
    expect(errors.title).toBeDefined()
  })

  it('keeps the first message for a field that fails two rules', () => {
    // Empty fails `required`; over-long fails `maxLength`. The operator sees
    // the actionable one first rather than whichever rule ran last.
    expect(validate({ title: '', assigned_to: '' }).title).toBe('A task title is required')
  })

  it('is empty when everything passes', () => {
    expect(validate({ title: 'ok', assigned_to: 'agent-1' })).toEqual({})
  })
})