import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  adoptPluginStyle,
  adoptSheet,
  clearPluginStyles,
  createSheet,
  getPluginSheet,
  getPluginStyle,
  loadPluginStyle,
  registerPluginStyle,
  unregisterPluginStyle,
} from './styles'

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

describe('loadPluginStyle and adoptPluginStyle', () => {
  beforeEach(() => {
    vi.stubGlobal('CSSStyleSheet', TextSheet)
    Object.defineProperty(document, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })
  })
  afterEach(() => {
    clearPluginStyles()
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

    await loadPluginStyle('md.numen.sketch', '/plugin/md.numen.sketch/dist/styles.css', fetchFn)

    expect(fetchFn).toHaveBeenCalledWith('/plugin/md.numen.sketch/dist/styles.css')
    expect(getPluginStyle('md.numen.sketch')).toBe('/plugin/md.numen.sketch/dist/styles.css')
    const sheet = getPluginSheet('md.numen.sketch') as unknown as TextSheet
    expect(sheet.text).toBe('.sketch { color: red }')
    expect(sheet.baseURL).toContain('/plugin/md.numen.sketch/dist/styles.css')

    registerPluginStyle('md.numen.sketch', '/plugin/md.numen.sketch/dist/next.css')
    expect(getPluginSheet('md.numen.sketch')).toBeUndefined()

    await loadPluginStyle('md.numen.sketch', '/plugin/md.numen.sketch/dist/next.css', fetchFn)
    expect(getPluginSheet('md.numen.sketch')).toBeDefined()
    clearPluginStyles()
    expect(getPluginStyle('md.numen.sketch')).toBeUndefined()
    expect(getPluginSheet('md.numen.sketch')).toBeUndefined()
  })

  it('skips fetching when sheets cannot be constructed or when sheet creation fails or races', async () => {
    deleteAdoptedStyleSheets()
    const skippedFetch = answer('.a {}')
    await loadPluginStyle('md.numen.sketch', '/s.css', skippedFetch)
    expect(skippedFetch).not.toHaveBeenCalled()
    expect(getPluginStyle('md.numen.sketch')).toBe('/s.css')

    Object.defineProperty(document, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })
    await loadPluginStyle('md.numen.sketch', '/bad.css', answer('THROW'))
    expect(getPluginSheet('md.numen.sketch')).toBeUndefined()

    await loadPluginStyle('md.numen.sketch', '/stale.css', async () => {
      registerPluginStyle('md.numen.sketch', '/newer.css')
      return { ok: true, text: async () => '.stale {}' } as unknown as Response
    })
    expect(getPluginSheet('md.numen.sketch')).toBeUndefined()
  })

  it('adopts a sheet into a root once and removes it on the last release', async () => {
    const existing = new TextSheet() as unknown as CSSStyleSheet
    document.adoptedStyleSheets = [existing]
    await loadPluginStyle('md.numen.sketch', '/s.css', answer('.a {}'))
    const sheet = getPluginSheet('md.numen.sketch')!

    const releaseFirst = adoptPluginStyle('md.numen.sketch', document)
    const releaseSecond = adoptPluginStyle('md.numen.sketch', document)
    expect(document.adoptedStyleSheets).toStrictEqual([existing, sheet])

    releaseFirst?.()
    releaseFirst?.()
    expect(document.adoptedStyleSheets).toStrictEqual([existing, sheet])
    releaseSecond?.()
    expect(document.adoptedStyleSheets).toStrictEqual([existing])

    const releaseAgain = adoptSheet(document, sheet)
    expect(document.adoptedStyleSheets).toStrictEqual([existing, sheet])
    releaseAgain()
    expect(document.adoptedStyleSheets).toStrictEqual([existing])

    // Adopting when sheet is already present in adoptedStyleSheets does not duplicate it
    document.adoptedStyleSheets = [existing, sheet]
    const releasePreExisting = adoptSheet(document, sheet)
    expect(document.adoptedStyleSheets).toStrictEqual([existing, sheet])
    releasePreExisting()
    expect(document.adoptedStyleSheets).toStrictEqual([existing])
  })

  it('keeps the link for a stylesheet that cannot be read', async () => {
    await loadPluginStyle('md.numen.sketch', '/s.css', answer('', false))

    expect(getPluginStyle('md.numen.sketch')).toBe('/s.css')
    expect(adoptPluginStyle('md.numen.sketch', document)).toBeUndefined()
  })

  it('drops the sheet when the plugin is unregistered', async () => {
    await loadPluginStyle('md.numen.sketch', '/s.css', answer('.a {}'))

    unregisterPluginStyle('md.numen.sketch')

    expect(getPluginSheet('md.numen.sketch')).toBeUndefined()
    expect(adoptPluginStyle('md.numen.sketch', document)).toBeUndefined()
  })
})
