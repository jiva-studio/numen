/**
 * The window follows the editor settings: it writes the font size, the line height and the font
 * family as tokens of the page once the settings are loaded, and again whenever one changes. The
 * settings come through the real store over the transport of a test core. Every value here is
 * invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSettingsStore, SETTINGS_WORDS, type PluginSettingInfo } from '@/entities/settings'
import { createSettingsCoreApi, type SettingsCore } from '@/testing/settingsWire'
import { readTokens } from '@/testing/readTokens'
import { createEditorTokens } from './createEditorTokens'

const SIZE = '--numen-editor-font-size'
const HEIGHT = '--numen-editor-line-height'
const FAMILY = '--numen-editor-font-family'

const FAMILIES = [
  { value: '', title: 'As in the theme', detail: '' },
  { value: 'sans', title: 'Sans', detail: '' },
  { value: 'serif', title: 'Serif', detail: '' },
  { value: 'mono', title: 'Mono', detail: '' },
]

const DECLARATION: readonly PluginSettingInfo[] = [
  {
    key: 'editor.fontSize',
    type: 'range',
    title: 'Font size',
    description: '',
    default: 14,
    min: 10,
    max: 32,
    step: 1,
    isShown: true,
  },
  {
    key: 'editor.fontFamily',
    type: 'string',
    title: 'Font family',
    description: '',
    default: '',
    options: FAMILIES,
    activeOptions: FAMILIES,
    isShown: true,
  },
  {
    key: 'editor.lineHeight',
    type: 'range',
    title: 'Line height',
    description: '',
    default: 1.6,
    min: 1.2,
    max: 2.4,
    step: 0.1,
    isShown: true,
  },
  {
    key: 'editor.lineNumbers',
    type: 'boolean',
    title: 'Line numbers',
    description: '',
    default: false,
    isShown: true,
  },
  {
    key: 'editor.highlightActiveLine',
    type: 'boolean',
    title: 'Highlight active line',
    description: '',
    default: false,
    isShown: true,
  },
  {
    key: 'editor.softWrap',
    type: 'boolean',
    title: 'Soft wrap',
    description: '',
    default: true,
    isShown: true,
  },
]

const createWindow = (file: object, core: Partial<SettingsCore> = {}) => {
  let written = JSON.stringify(file)
  const api = createSettingsCoreApi({
    getSettings: () => Promise.resolve({ written, path: '/numen.json', app: DECLARATION }),
    updateSettings: () => Promise.resolve(),
    ...core,
  })
  const store = createSettingsStore(api.configuration.settings, SETTINGS_WORDS, vi.fn())
  return {
    store,
    setFile: (next: object) => {
      written = JSON.stringify(next)
    },
  }
}

const getEditorStyle = (): HTMLStyleElement | null =>
  document.head.querySelector<HTMLStyleElement>('style[data-appearance="editor"]')

const readPageTokens = (): ReadonlyMap<string, string> =>
  readTokens(getEditorStyle()?.textContent ?? '')

const running: (() => void)[] = []

const startTokens = (store: ReturnType<typeof createWindow>['store']) => {
  const stop = createEditorTokens(store).start()
  running.push(stop)
  return stop
}

afterEach(() => {
  while (running.length) running.pop()?.()
  document.head.innerHTML = ''
})

describe('the editor tokens of the window', () => {
  it('are written from the settings the file holds once the settings are loaded', async () => {
    const { store } = createWindow({
      app: { editor: { fontSize: 20, lineHeight: 1.8, fontFamily: 'mono' } },
    })
    await store.start()

    startTokens(store)

    const tokens = readPageTokens()
    expect(tokens.get(SIZE)).toBe('20px')
    expect(tokens.get(HEIGHT)).toBe('1.8')
    expect(tokens.get(FAMILY)).toContain('mono')
  })

  it('are the defaults of the declaration where the file holds no value, with no family', async () => {
    const { store } = createWindow({})
    await store.start()

    startTokens(store)

    const tokens = readPageTokens()
    expect(tokens.get(SIZE)).toBe('14px')
    expect(tokens.get(HEIGHT)).toBe('1.6')
    expect(tokens.has(FAMILY)).toBe(false)
  })

  it('are the defaults of the page where the settings could not be read', async () => {
    const { store } = createWindow(
      {},
      { getSettings: () => Promise.reject(new Error('the core is busy')) },
    )
    await store.start()

    startTokens(store)

    const tokens = readPageTokens()
    expect(tokens.get(SIZE)).toBe('14px')
    expect(tokens.get(HEIGHT)).toBe('1.6')
    expect(tokens.has(FAMILY)).toBe(false)
  })

  it('follow a font size written in the settings at once', async () => {
    const { store } = createWindow({})
    await store.start()
    startTokens(store)

    await store.write.writeSetting(['app', 'editor', 'fontSize'], 24)

    await vi.waitFor(() => expect(readPageTokens().get(SIZE)).toBe('24px'))
  })

  it('follow a line height and a family written in the settings', async () => {
    const { store } = createWindow({})
    await store.start()
    startTokens(store)

    await store.write.writeSetting(['app', 'editor', 'lineHeight'], 2)
    await store.write.writeSetting(['app', 'editor', 'fontFamily'], 'serif')

    await vi.waitFor(() => {
      const tokens = readPageTokens()
      expect(tokens.get(HEIGHT)).toBe('2')
      expect(tokens.get(FAMILY)).toContain('serif')
    })
  })

  it('toggles the active line colour in the open window', async () => {
    const { store } = createWindow({})
    await store.start()
    startTokens(store)
    const key = '--numen-editor-active-line'
    expect(readPageTokens().has(key)).toBe(false)
    await store.write.writeSetting(['app', 'editor', 'highlightActiveLine'], true)
    await vi.waitFor(() => expect(readPageTokens().get(key)).toBe('var(--numen-selection-match)'))
    await store.write.writeSetting(['app', 'editor', 'highlightActiveLine'], false)
    await vi.waitFor(() => expect(readPageTokens().has(key)).toBe(false))
  })

  it('drop the family when the choice goes back to the face of the theme', async () => {
    const { store } = createWindow({ app: { editor: { fontFamily: 'mono' } } })
    await store.start()
    startTokens(store)
    expect(readPageTokens().has(FAMILY)).toBe(true)

    await store.write.writeSetting(['app', 'editor', 'fontFamily'], '')

    await vi.waitFor(() => expect(readPageTokens().has(FAMILY)).toBe(false))
    expect(readPageTokens().get(SIZE)).toBe('14px')
  })

  it('follow the file when another writer changes it', async () => {
    const { store, setFile } = createWindow({ app: { editor: { fontSize: 16 } } })
    await store.start()
    startTokens(store)
    expect(readPageTokens().get(SIZE)).toBe('16px')

    setFile({ app: { editor: { fontSize: 30, lineHeight: 2.2 } } })
    await store.start()

    await vi.waitFor(() => {
      const tokens = readPageTokens()
      expect(tokens.get(SIZE)).toBe('30px')
      expect(tokens.get(HEIGHT)).toBe('2.2')
    })
  })

  it('write no family for a value that is none of the choices', async () => {
    const { store } = createWindow({
      app: { editor: { fontSize: 18, fontFamily: 'x; } body { display: none' } },
    })
    await store.start()

    startTokens(store)

    const tokens = readPageTokens()
    expect(tokens.get(SIZE)).toBe('18px')
    expect(tokens.has(FAMILY)).toBe(false)
  })

  it('ignore a change of a setting that is not an editor setting', async () => {
    const { store } = createWindow({})
    await store.start()
    startTokens(store)
    const before = getEditorStyle()?.textContent

    await store.write.writeSetting(['app', 'naming', 'syncTitleAndFilename'], false)

    expect(getEditorStyle()?.textContent).toBe(before)
    expect(readPageTokens().get(SIZE)).toBe('14px')
  })

  it('are written into one element at the end of the head, after the ones of the theme', async () => {
    const mode = document.head.appendChild(document.createElement('style'))
    mode.setAttribute('data-appearance', 'mode')
    const theme = document.head.appendChild(document.createElement('style'))
    theme.setAttribute('data-appearance', 'theme')
    const sizes = document.head.appendChild(document.createElement('style'))
    sizes.setAttribute('data-appearance', 'sizes')
    const { store } = createWindow({})
    await store.start()

    startTokens(store)
    await store.write.writeSetting(['app', 'editor', 'fontSize'], 22)
    await vi.waitFor(() => expect(readPageTokens().get(SIZE)).toBe('22px'))

    expect(document.head.querySelectorAll('style[data-appearance="editor"]')).toHaveLength(1)
    expect(document.head.lastElementChild).toBe(getEditorStyle())
  })

  it('are written into the element a second start finds, not into a new one', async () => {
    const { store } = createWindow({ app: { editor: { fontSize: 12 } } })
    await store.start()

    startTokens(store)
    startTokens(store)

    expect(document.head.querySelectorAll('style[data-appearance="editor"]')).toHaveLength(1)
    expect(readPageTokens().get(SIZE)).toBe('12px')
  })

  it('stop following the settings once the start is stopped', async () => {
    const { store } = createWindow({})
    await store.start()
    const stop = startTokens(store)
    expect(readPageTokens().get(SIZE)).toBe('14px')

    stop()
    await store.write.writeSetting(['app', 'editor', 'fontSize'], 28)
    await new Promise((resolve) => setTimeout(resolve, 20))

    expect(readPageTokens().get(SIZE)).not.toBe('28px')
  })
})
