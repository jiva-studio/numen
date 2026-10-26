import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  adoptPluginStyle,
  clearPluginStyles,
  getPluginSheet,
  getPluginStyle,
  loadPluginStyle,
  unregisterPluginStyle,
} from './styles'

/** A sheet as an engine that constructs them makes one: it keeps the text it was given. */
class TextSheet {
  text = ''
  replaceSync(text: string): void {
    this.text = text
  }
}

const answer = (text: string, ok = true) =>
  vi.fn(async () => ({ ok, text: async () => text }) as unknown as Response)

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
  })

  it('reads the stylesheet into a sheet the panes adopt', async () => {
    const fetchFn = answer('.sketch { color: red }')

    await loadPluginStyle(
      'md.numen.sketch',
      '/api/plugins/md.numen.sketch/dist/styles.css',
      fetchFn,
    )

    expect(fetchFn).toHaveBeenCalledWith('/api/plugins/md.numen.sketch/dist/styles.css')
    expect(getPluginStyle('md.numen.sketch')).toBe('/api/plugins/md.numen.sketch/dist/styles.css')
    expect((getPluginSheet('md.numen.sketch') as unknown as TextSheet).text).toBe(
      '.sketch { color: red }',
    )
  })

  it('adopts a sheet into a root once and removes it on the last release', async () => {
    await loadPluginStyle('md.numen.sketch', '/s.css', answer('.a {}'))
    const sheet = getPluginSheet('md.numen.sketch')

    const releaseFirst = adoptPluginStyle('md.numen.sketch', document)
    const releaseSecond = adoptPluginStyle('md.numen.sketch', document)
    expect(document.adoptedStyleSheets).toStrictEqual([sheet])

    releaseFirst?.()
    expect(document.adoptedStyleSheets).toStrictEqual([sheet])
    releaseSecond?.()
    expect(document.adoptedStyleSheets).toStrictEqual([])
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
