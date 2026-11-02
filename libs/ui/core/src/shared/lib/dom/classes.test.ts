import { describe, expect, it } from 'vitest'
import { cn } from './classes'

describe('cn', () => {
  it('has the text colour of a variant and the small text size in one class list', () => {
    expect(cn('bg-accent text-accent-ink', 'text-small h-7')).toBe(
      'bg-accent text-accent-ink text-small h-7',
    )
  })

  it('lets the last text colour win', () => {
    expect(cn('text-ink', 'text-hushed')).toBe('text-hushed')
  })

  it('lets the last text size win', () => {
    expect(cn('text-base', 'text-small')).toBe('text-small')
  })
})
