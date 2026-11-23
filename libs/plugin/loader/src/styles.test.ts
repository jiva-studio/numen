import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPluginStyles, createSheet, type PluginStyles } from './styles'

/** A sheet as an engine that constructs them makes one: it keeps the text it was given. */
class TextSheet {
  text = ''
  baseURL?: string | undefined
  constructor(options?: { baseURL?: string }) {
    this.baseURL = options?.baseURL
  }
  replaceSync(text: string): void {
    if (text === 'THROW') throw new Error('bad css')
    this.text = text
  }
}

const answer = (text: string, ok = true) =>
  vi.fn(async () => ({ ok, text: async () => text }) as unknown as Response)

const nativeAdoptedDescriptor = Object.getOwnPropertyDescriptor(
  Document.prototype,
  'adoptedStyleSheets',
)

const deleteAdoptedStyleSheets = (): void => {
  Reflect.deleteProperty(document, 'adoptedStyleSheets')
  Reflect.deleteProperty(Document.prototype, 'adoptedStyleSheets')
}

describe('the plugin styles', () => {
  let styles: PluginStyles
  beforeEach(() => {
    styles = createPluginStyles()
    vi.stubGlobal('CSSStyleSheet', TextSheet)
    Object.defineProperty(document, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })
  })
  afterEach(() => {
    styles.clear()
    vi.unstubAllGlobals()
    Reflect.deleteProperty(document, 'adoptedStyleSheets')
    if (nativeAdoptedDescriptor) {
      Object.defineProperty(Document.prototype, 'adoptedStyleSheets', nativeAdoptedDescriptor)
    }
  })

  it('checks whether the engine can construct and adopt stylesheets', () => {
    expect(createSheet('.a {}')).toBeDefined()

    deleteAdoptedStyleSheets()
    expect(createSheet('.a {}')).toBeUndefined()

    Object.defineProperty(document, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })
    vi.stubGlobal('CSSStyleSheet', class {})
    expect(createSheet('.a {}')).toBeUndefined()

    vi.stubGlobal('CSSStyleSheet', undefined)
    expect(createSheet('.a {}')).toBeUndefined()
  })

  it('constructs a sheet with optional baseURL and returns undefined on parse error', () => {
    const withoutBase = createSheet('.a {}') as unknown as TextSheet
    expect(withoutBase.text).toBe('.a {}')
    expect(withoutBase.baseURL).toBeUndefined()

    const withBase = createSheet('.b {}', 'https://example.org/s.css') as unknown as TextSheet
    expect(withBase.text).toBe('.b {}')
    expect(withBase.baseURL).toBe('https://example.org/s.css')

    expect(createSheet('THROW')).toBeUndefined()
  })

  it('reads the stylesheet into a sheet the panes adopt', async () => {
    const fetchFn = answer('.sketch { color: red }')

    await styles.load('md.numen.sketch', '/plugin/md.numen.sketch/dist/styles.css', fetchFn)

    expect(fetchFn).toHaveBeenCalledWith('/plugin/md.numen.sketch/dist/styles.css')
    expect(styles.getUrl('md.numen.sketch')).toBe('/plugin/md.numen.sketch/dist/styles.css')
    const sheet = styles.getSheet('md.numen.sketch') as unknown as TextSheet
    expect(sheet.text).toBe('.sketch { color: red }')
    expect(sheet.baseURL).toContain('/plugin/md.numen.sketch/dist/styles.css')

    styles.register('md.numen.sketch', '/plugin/md.numen.sketch/dist/next.css')
    expect(styles.getSheet('md.numen.sketch')).toBeUndefined()

    await styles.load('md.numen.sketch', '/plugin/md.numen.sketch/dist/next.css', fetchFn)
    expect(styles.getSheet('md.numen.sketch')).toBeDefined()
    styles.clear()
    expect(styles.getUrl('md.numen.sketch')).toBeUndefined()
    expect(styles.getSheet('md.numen.sketch')).toBeUndefined()
  })

  it('skips fetching when sheets cannot be constructed or when sheet creation fails or races', async () => {
    deleteAdoptedStyleSheets()
    const skippedFetch = answer('.a {}')
    await styles.load('md.numen.sketch', '/s.css', skippedFetch)
    expect(skippedFetch).not.toHaveBeenCalled()
    expect(styles.getUrl('md.numen.sketch')).toBe('/s.css')

    Object.defineProperty(document, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })
    await styles.load('md.numen.sketch', '/bad.css', answer('THROW'))
    expect(styles.getSheet('md.numen.sketch')).toBeUndefined()

    await styles.load('md.numen.sketch', '/stale.css', async () => {
      styles.register('md.numen.sketch', '/newer.css')
      return { ok: true, text: async () => '.stale {}' } as unknown as Response
    })
    expect(styles.getSheet('md.numen.sketch')).toBeUndefined()
  })

  it('adopts a sheet into a root once and removes it on the last release', async () => {
    const existing = new TextSheet() as unknown as CSSStyleSheet
    document.adoptedStyleSheets = [existing]
    await styles.load('md.numen.sketch', '/s.css', answer('.a {}'))
    const sheet = styles.getSheet('md.numen.sketch')!

    const releaseFirst = styles.adopt('md.numen.sketch', document)
    const releaseSecond = styles.adopt('md.numen.sketch', document)
    expect(document.adoptedStyleSheets).toStrictEqual([existing, sheet])

    releaseFirst?.()
    releaseFirst?.()
    expect(document.adoptedStyleSheets).toStrictEqual([existing, sheet])
    releaseSecond?.()
    expect(document.adoptedStyleSheets).toStrictEqual([existing])

    const releaseAgain = styles.adoptSheet(document, sheet)
    expect(document.adoptedStyleSheets).toStrictEqual([existing, sheet])
    releaseAgain()
    expect(document.adoptedStyleSheets).toStrictEqual([existing])

    // Adopting when sheet is already present in adoptedStyleSheets does not duplicate it
    document.adoptedStyleSheets = [existing, sheet]
    const releasePreExisting = styles.adoptSheet(document, sheet)
    expect(document.adoptedStyleSheets).toStrictEqual([existing, sheet])
    releasePreExisting()
    expect(document.adoptedStyleSheets).toStrictEqual([existing])
  })

  it('keeps the link for a stylesheet that cannot be read', async () => {
    await styles.load('md.numen.sketch', '/s.css', answer('', false))

    expect(styles.getUrl('md.numen.sketch')).toBe('/s.css')
    expect(styles.adopt('md.numen.sketch', document)).toBeUndefined()
  })

  it('keeps the styles of each store apart', async () => {
    const other = createPluginStyles()
    await styles.load('md.numen.sketch', '/s.css', answer('.a {}'))

    expect(other.getUrl('md.numen.sketch')).toBeUndefined()
    expect(other.getSheet('md.numen.sketch')).toBeUndefined()
  })

  it('drops the sheet when the plugin is unregistered', async () => {
    await styles.load('md.numen.sketch', '/s.css', answer('.a {}'))

    styles.unregister('md.numen.sketch')

    expect(styles.getSheet('md.numen.sketch')).toBeUndefined()
    expect(styles.adopt('md.numen.sketch', document)).toBeUndefined()
  })
})
