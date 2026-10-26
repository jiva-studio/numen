// @vitest-environment node
/**
 * A stylesheet loaded after the one of `@numen/ui`, built less what that one carries.
 */
import { describe, expect, it } from 'vitest'
import { getNormalCss, removePresent } from './vite'

const UI = `
@layer utilities {
  .prose { --tw-prose-body: oklch(37.3% .034 259.733); color: var(--tw-prose-body) }
  .prose-numen { --tw-prose-body: var(--numen-answer-fg) }
}
`

describe('removePresent', () => {
  it('removes the declarations the ui stylesheet carries', () => {
    const later = `
@layer utilities {
  .prose { --tw-prose-body: oklch(37.3% .034 259.733); color: var(--tw-prose-body) }
  .p-2 { padding: .5rem }
}
`
    expect(removePresent(later, UI)).toBe(
      getNormalCss('@layer utilities { .p-2 { padding: .5rem } }'),
    )
  })

  it('keeps declarations with another value or another selector', () => {
    const later = `
@layer utilities { .prose { color: red } }
@layer components { .prose-numen { --tw-prose-body: var(--numen-answer-fg) } }
`
    expect(removePresent(later, UI)).toBe(getNormalCss(later))
  })

  it('returns an empty string when every declaration is present', () => {
    expect(removePresent(UI, UI)).toBe('')
  })
})
