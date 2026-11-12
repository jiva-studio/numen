import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ThemeClient } from '../api/theme'
import type { ThemeEntry, ThemeList } from '../lib/appearance'
import { WORDS } from '../words'
import { createAppearance } from './appearance'

const NUMEN: ThemeEntry = {
  name: 'preset:numen',
  title: 'numen',
  origin: 'preset',
  pluginId: '',
  isPinned: false,
}
const NIGHT: ThemeEntry = {
  name: 'plugin:org.example.themes/night',
  title: 'Night',
  origin: 'other',
  pluginId: 'org.example.themes',
  isPinned: true,
}
const DAY: ThemeEntry = {
  name: 'plugin:org.example.themes/day',
  title: 'Day',
  origin: 'other',
  pluginId: 'org.example.themes',
  isPinned: false,
}
const SEA: ThemeEntry = {
  name: 'mine:sea',
  title: 'sea',
  origin: 'mine',
  pluginId: '',
  isPinned: false,
}

const LIST: ThemeList = {
  themes: [NUMEN, NIGHT, DAY, SEA],
  applied: NUMEN.name,
  mode: 'system',
}

async function* waits(): AsyncGenerator<readonly string[]> {
  await new Promise<never>(() => undefined)
}

const createClient = (list: ThemeList = LIST) => {
  const state = { list }
  const client: ThemeClient = {
    listThemes: vi.fn(async () => state.list),
    readTheme: vi.fn(async (name: string) => `:root { --theme: "${name}" }`),
    writeAppearance: vi.fn(async () => ''),
    watchThemes: () => waits(),
  }
  return { client, state }
}

const open = (client: ThemeClient, errors: string[] = []) => {
  const page = document.implementation.createHTMLDocument('')
  const appearance = createAppearance({
    client,
    getPluginName: (id) => (id === 'org.example.themes' ? 'Sample Themes' : id),
    onError: (text) => errors.push(text),
    document: page,
  })
  return { appearance, page, errors }
}

const style = (page: Document, is: string) =>
  page.head.querySelector(`style[data-appearance="${is}"]`)?.textContent

describe('the themes the palette offers', () => {
  let setup: ReturnType<typeof createClient>

  beforeEach(() => {
    setup = createClient()
  })

  it('stand in the shipped group, one group for each plugin under its name, and the own group', async () => {
    const { appearance } = open(setup.client)
    await appearance.refresh()

    const groups = appearance.getGroups('')

    expect(groups.map((one) => one.id)).toStrictEqual([
      'shipping',
      'plugin:org.example.themes',
      'owned',
      'half',
    ])
    expect(groups.map((one) => one.title)).toStrictEqual([
      WORDS.shipping,
      'Sample Themes',
      WORDS.owned,
      WORDS.half,
    ])
    expect(groups[0]?.items.map((one) => one.title)).toStrictEqual(['numen'])
    expect(groups[1]?.items.map((one) => one.title)).toStrictEqual(['Night', 'Day'])
    expect(groups[2]?.items.map((one) => one.title)).toStrictEqual(['sea'])
  })

  it('mark the applied one and leave the others unmarked', async () => {
    const { appearance } = open(setup.client)
    await appearance.refresh()

    const groups = appearance.getGroups('')
    const first = groups[0]?.items[0]
    const other = groups[1]?.items[0]
    const modeItem = groups[3]?.items[0]

    expect(first?.detail).toBe(WORDS.current)
    expect(first?.actions).toStrictEqual([{ id: 'apply', text: WORDS.apply }])
    expect(other?.detail).toBeUndefined()
    expect(modeItem?.actions).toStrictEqual([{ id: 'apply', text: WORDS.apply }])
  })

  it('are narrowed by the text typed, whatever its case or surrounding spaces', async () => {
    const { appearance } = open(setup.client)
    await appearance.refresh()

    const titles = appearance
      .getGroups('  NIG  ')
      .flatMap((group) => group.items)
      .map((one) => one.title)

    expect(titles).toStrictEqual(['Night'])
  })
})

describe('choosing a theme', () => {
  it('writes the name with the mode in force and updates the single style elements on the page', async () => {
    const { client } = createClient()
    const { appearance, page } = open(client)
    await appearance.refresh()

    await appearance.chooseItem(`theme:${DAY.name}`)
    await appearance.chooseItem(`theme:${SEA.name}`)

    expect(client.writeAppearance).toHaveBeenCalledWith(DAY.name, 'system')
    expect(page.head.querySelectorAll('style[data-appearance="theme"]')).toHaveLength(1)
    expect(page.head.querySelectorAll('style[data-appearance="mode"]')).toHaveLength(1)
    expect(style(page, 'theme')).toBe(`:root { --theme: "${SEA.name}" }`)
    expect(style(page, 'mode')).toBe(':root { color-scheme: light dark; }')
    expect(appearance.applied.value).toBe(SEA.name)
  })

  it('reports why the choice was not written and leaves the page as it is', async () => {
    const { client } = createClient()
    vi.mocked(client.writeAppearance).mockResolvedValue('the settings are read-only')
    const { appearance, page, errors } = open(client)
    await appearance.refresh()

    await appearance.chooseItem(`theme:${DAY.name}`)

    expect(errors).toStrictEqual(['the settings are read-only'])
    expect(style(page, 'theme')).toBeUndefined()
    expect(appearance.applied.value).toBe(NUMEN.name)
  })

  it('reports a failing service', async () => {
    const { client } = createClient()
    vi.mocked(client.writeAppearance).mockRejectedValue(new Error('offline'))
    const { appearance, errors } = open(client)
    await appearance.refresh()

    await appearance.chooseItem(`theme:${DAY.name}`)

    expect(errors).toHaveLength(1)
  })
})

describe('choosing a mode', () => {
  it('writes the applied theme with the mode and writes the color scheme of the mode', async () => {
    const { client } = createClient()
    const { appearance, page } = open(client)
    await appearance.refresh()

    await appearance.chooseItem('mode:dark')

    expect(client.writeAppearance).toHaveBeenCalledWith(NUMEN.name, 'dark')
    expect(style(page, 'mode')).toBe(':root { color-scheme: dark; }')
    expect(appearance.mode.value).toBe('dark')
  })

  it('cannot be done while the applied theme sets light or dark itself', async () => {
    const { client } = createClient({ ...LIST, applied: NIGHT.name })
    const { appearance } = open(client)
    await appearance.refresh()

    const modes = appearance
      .getGroups('')
      .flatMap((group) => group.items.filter((one) => one.id.startsWith('mode:')))

    expect(modes).toHaveLength(3)
    expect(modes.every((one) => one.isDisabled === true && one.detail === WORDS.pinned)).toBe(true)
  })

  it('marks the one in force and leaves modes enabled when the applied theme is unknown', async () => {
    const { client } = createClient({ themes: [NIGHT], applied: 'missing', mode: 'light' })
    const { appearance } = open(client)
    await appearance.refresh()

    const modes = appearance
      .getGroups('')
      .flatMap((group) => group.items)
      .filter((one) => one.id.startsWith('mode:'))
    const current = modes.filter((one) => one.detail === WORDS.current)

    expect(modes.every((one) => one.isDisabled === undefined)).toBe(true)
    expect(current.map((one) => one.id)).toStrictEqual(['mode:light'])
  })

  it('ignores an item that is neither a theme nor a mode', async () => {
    const { client } = createClient()
    const { appearance } = open(client)
    await appearance.refresh()

    await appearance.chooseItem('other:thing')

    expect(client.writeAppearance).not.toHaveBeenCalled()
  })
})

describe('reading the themes again', () => {
  it('leaves the page as the application served it on the first read and when nothing changed', async () => {
    const { client } = createClient()
    const { appearance, page } = open(client)

    await appearance.refresh()
    await appearance.refresh()

    expect(client.readTheme).not.toHaveBeenCalled()
    expect(style(page, 'theme')).toBeUndefined()
  })

  it('applies the theme when only the mode changed', async () => {
    const { client, state } = createClient()
    const { appearance, page } = open(client)
    await appearance.refresh()

    state.list = { ...LIST, mode: 'dark' }
    await appearance.refresh()

    expect(appearance.mode.value).toBe('dark')
    expect(style(page, 'mode')).toBe(':root { color-scheme: dark; }')
  })

  it('applies the default theme to the page when the applied theme is removed', async () => {
    const { client, state } = createClient({ ...LIST, applied: DAY.name })
    const { appearance, page } = open(client)
    await appearance.refresh()

    state.list = { themes: [NUMEN, SEA], applied: NUMEN.name, mode: 'system' }
    await appearance.refresh()

    expect(appearance.applied.value).toBe(NUMEN.name)
    expect(style(page, 'theme')).toBe(`:root { --theme: "${NUMEN.name}" }`)
  })

  it('reports themes that cannot be listed', async () => {
    const { client } = createClient()
    vi.mocked(client.listThemes).mockRejectedValue(new Error('offline'))
    const { appearance, errors } = open(client)

    await appearance.refresh()

    expect(errors).toHaveLength(1)
    expect(errors[0]).toContain(WORDS.themesUnlisted)
  })
})
