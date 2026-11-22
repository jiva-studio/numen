import { describe, expect, it } from 'vitest'
import { renderCompletionIcon } from './completionTheme'

describe('the completion icons of task fields', () => {
  it.each(['project', 'context', 'tag', 'date', 'priority'])(
    'draws an svg for the %s type',
    (type) => {
      const icon = renderCompletionIcon({ label: 'x', type })

      expect(icon).toBeInstanceOf(SVGElement)
    },
  )

  it('draws nothing for a completion of another type', () => {
    expect(renderCompletionIcon({ label: 'x', type: 'text' })).toBeNull()
    expect(renderCompletionIcon({ label: 'x' })).toBeNull()
  })
})
