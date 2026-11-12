import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { THEME_MODE, THEME_ORIGIN } from '@numen/core-client'
import { installCoreAnswers } from '@numen/core-client/testing'
import { themeClient } from './theme'

const core = {
  listThemes: vi.fn(),
  readTheme: vi.fn(),
  writeAppearance: vi.fn(),
  watchThemes: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  vi.resetAllMocks()
  restore = installCoreAnswers({ theme: core })
})

afterEach(() => {
  restore()
})

const answer = (overrides: Record<string, unknown> = {}) => ({
  themes: [],
  applied: 'preset:numen',
  mode: THEME_MODE.SYSTEM,
  interfaceScale: 1,
  textScale: 1,
  ...overrides,
})

describe('the themes the application lists', () => {
  it('keep apart the shipped, the own and the other themes', async () => {
    core.listThemes.mockResolvedValue(
      answer({
        themes: [
          { name: 'preset:numen', title: 'numen', origin: THEME_ORIGIN.PRESET },
          { name: 'mine:sea', title: 'sea', origin: THEME_ORIGIN.MINE },
          {
            name: 'plugin:org.example.themes/night',
            title: 'Night',
            origin: THEME_ORIGIN.PLUGIN,
            pluginId: 'org.example.themes',
            isPinned: true,
          },
        ],
      }),
    )

    const { themes } = await themeClient.listThemes()

    expect(themes.map((one) => [one.name, one.origin, one.pluginId, one.isPinned])).toStrictEqual([
      ['preset:numen', 'preset', '', false],
      ['mine:sea', 'mine', '', false],
      ['plugin:org.example.themes/night', 'other', 'org.example.themes', true],
    ])
  })

  it('carry the applied theme and the mode in the window’s words', async () => {
    core.listThemes.mockResolvedValue(answer({ applied: 'mine:sea', mode: THEME_MODE.DARK }))

    expect(await themeClient.listThemes()).toMatchObject({ applied: 'mine:sea', mode: 'dark' })
  })

  it('are read in the system mode where the core names none', async () => {
    core.listThemes.mockResolvedValue(answer({ mode: THEME_MODE.UNSPECIFIED }))

    expect((await themeClient.listThemes()).mode).toBe('system')
  })
})

describe('a choice written', () => {
  it('sends the mode as the schema carries it and answers why it was refused', async () => {
    core.writeAppearance.mockResolvedValue({ error: 'the file is read-only' })

    const reason = await themeClient.writeAppearance('mine:sea', 'light')

    expect(reason).toBe('the file is read-only')
    expect(core.writeAppearance.mock.calls[0]?.[0]).toMatchObject({
      name: 'mine:sea',
      mode: THEME_MODE.LIGHT,
    })
  })
})

describe('a theme read', () => {
  it('is its stylesheet', async () => {
    core.readTheme.mockResolvedValue({ css: ':root { color-scheme: dark; }' })

    expect(await themeClient.readTheme('mine:sea')).toBe(':root { color-scheme: dark; }')
  })
})
