import { describe, expect, it } from 'vitest'
import { addVersion } from './write-report.js'

describe('addVersion', () => {
  it('stamps the scripts and styles of the page with the run', () => {
    const html = '<link rel="stylesheet" href="tokens/colors.css" /><script src="data.js"></script>'
    expect(addVersion(html, 'abc')).toBe(
      '<link rel="stylesheet" href="tokens/colors.css?v=abc" /><script src="data.js?v=abc"></script>',
    )
  })

  it('leaves the font and outside links as they are', () => {
    const html =
      '<link rel="preload" href="fonts/inter.woff2" /><a href="https://example.com/a.js">'
    expect(addVersion(html, 'abc')).toBe(html)
  })
})
