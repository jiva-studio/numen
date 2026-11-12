// @vitest-environment node
/**
 * A stylesheet loaded after the one of `@numen/ui`, built less what that one carries.
 */
import { existsSync, mkdirSync, writeFileSync } from 'node:fs'
import { dirname } from 'node:path'
import { describe, expect, it } from 'vitest'
import { UI_STYLES, createLayeredStylesPlugin, getNormalCss, removePresentRules } from './vite'

const UI = `
@layer utilities {
  .prose { --tw-prose-body: oklch(37.3% .034 259.733); color: var(--tw-prose-body) }
  .prose-numen { --tw-prose-body: var(--numen-answer-fg) }
}
`

describe('removePresentRules', () => {
  it('removes the declarations the ui stylesheet carries', () => {
    const later = `
@layer utilities {
  .prose { --tw-prose-body: oklch(37.3% .034 259.733); color: var(--tw-prose-body) }
  .p-2 { padding: .5rem }
}
`
    expect(removePresentRules(later, UI)).toBe(
      getNormalCss('@layer utilities { .p-2 { padding: .5rem } }'),
    )
  })

  it('keeps declarations with another value or another selector', () => {
    const later = `
@layer utilities { .prose { color: red } }
@layer components { .prose-numen { --tw-prose-body: var(--numen-answer-fg) } }
`
    expect(removePresentRules(later, UI)).toBe(getNormalCss(later))
  })

  it('returns an empty string when every declaration is present', () => {
    expect(removePresentRules(UI, UI)).toBe('')
  })

  it('minifies css and compares selector, at-rule context, and blockless at-rules', () => {
    expect(getNormalCss('.a {\n  color: red;\n}')).toBe('.a{color:red}')
    // Same declaration under a different selector in the same layer is kept
    const differentSelector = '@layer utilities { .other { color: var(--tw-prose-body) } }'
    expect(removePresentRules(differentSelector, UI)).toBe(getNormalCss(differentSelector))

    // Blockless @layer statements are stripped while non-empty @layer blocks stay
    const withLayerOrder =
      '@layer theme, base, utilities;\n@layer utilities { .p-2 { padding: .5rem } }'
    expect(removePresentRules(withLayerOrder, UI)).toBe(
      getNormalCss('@layer utilities { .p-2 { padding: .5rem } }'),
    )

    // Blockless non-layer at-rules are deduplicated when present and kept when new
    const nsPresent = '@namespace svg "http://www.w3.org/2000/svg"; .a { color: red }'
    const nsLaterSame = '@namespace svg "http://www.w3.org/2000/svg"; .b { color: blue }'
    const nsLaterDiff = '@namespace math "http://www.w3.org/1998/Math/MathML"; .b { color: blue }'
    expect(removePresentRules(nsLaterSame, nsPresent)).toBe(getNormalCss('.b { color: blue }'))
    expect(removePresentRules(nsLaterDiff, nsPresent)).toBe(getNormalCss(nsLaterDiff))
  })

  it('builds a Vite plugin that strips UI declarations only for the target stylesheet', () => {
    if (!existsSync(UI_STYLES)) {
      mkdirSync(dirname(UI_STYLES), { recursive: true })
      writeFileSync(UI_STYLES, UI, 'utf8')
    }
    const plugin = createLayeredStylesPlugin('/app/styles.css')
    expect(plugin.name).toBe('numen-layered-styles')
    const transform =
      typeof plugin.transform === 'function' ? plugin.transform : plugin.transform?.handler
    expect(
      transform?.call({} as never, '.custom-only { z-index: 999 }', '/other.css'),
    ).toBeUndefined()
    const transformed = transform?.call(
      {} as never,
      '.custom-only { z-index: 999 }',
      '/app/styles.css?used',
    ) as { code: string; map: null } | undefined
    expect(transformed).toEqual({
      code: getNormalCss('.custom-only { z-index: 999 }'),
      map: null,
    })
    expect(UI_STYLES).toContain('numen-ui.css')
  })
})
