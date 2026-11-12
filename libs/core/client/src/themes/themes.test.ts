/**
 * How the application's description of the window's appearance is put in the windows' terms. The
 * rest passes a message through unchanged, and the schema defines its shape.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { THEME_MODE, THEME_ORIGIN, theme } from '../clients'
import { installCoreAnswers } from '../testing'
import { createThemeClient } from './themes'

const request = {
  listThemes: vi.fn(),
  readTheme: vi.fn(),
  writeAppearance: vi.fn(),
  watchThemes: vi.fn(),
}

const themes = createThemeClient(theme)

let restore: () => void

beforeEach(() => {
  vi.resetAllMocks()
  restore = installCoreAnswers({ theme: request })
})

afterEach(() => {
  restore()
})

const createAnswer = (overrides: Record<string, unknown> = {}) => ({
  themes: [],
  applied: 'preset:numen',
  mode: THEME_MODE.SYSTEM,
  interfaceScale: 1,
  textScale: 1,
  interfaceScaleBounds: { min: 0.8, max: 2 },
  textScaleBounds: { min: 0.8, max: 1.75 },
  ...overrides,
})

describe('every theme there is', () => {
  it('keeps apart the shipped, the own and the other themes', async () => {
    request.listThemes.mockResolvedValue(
      createAnswer({
        themes: [
          { name: 'preset:numen', title: 'numen', origin: THEME_ORIGIN.PRESET },
          { name: 'mine:sea', title: 'sea', origin: THEME_ORIGIN.MINE, isPinned: true },
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

    expect((await themes.getAppearance()).themes).toStrictEqual([
      { name: 'preset:numen', title: 'numen', origin: 'preset', pluginId: '', isPinned: false },
      { name: 'mine:sea', title: 'sea', origin: 'mine', pluginId: '', isPinned: true },
      {
        name: 'plugin:org.example.themes/night',
        title: 'Night',
        origin: 'other',
        pluginId: 'org.example.themes',
        isPinned: true,
      },
    ])
  })
})

describe('which half of a color pair is read', () => {
  it('is returned in the window’s own words', async () => {
    for (const [mode, word] of [
      [THEME_MODE.LIGHT, 'light'],
      [THEME_MODE.DARK, 'dark'],
      [THEME_MODE.SYSTEM, 'system'],
      [THEME_MODE.UNSPECIFIED, 'system'],
    ] as const) {
      request.listThemes.mockResolvedValue(createAnswer({ mode }))
      expect((await themes.getAppearance()).mode).toBe(word)
    }
  })

  it('is the system’s where the window has no word for it', async () => {
    request.listThemes.mockResolvedValue(createAnswer({ mode: 99 }))

    expect((await themes.getAppearance()).mode).toBe('system')
  })
})

describe('how far a size goes', () => {
  it('is the two ends the application named', async () => {
    request.listThemes.mockResolvedValue(createAnswer({ applied: 'mine:sea' }))
    const { applied, sizes, bounds } = await themes.getAppearance()

    expect(applied).toBe('mine:sea')
    expect(sizes).toStrictEqual({ interfaceScale: 1, textScale: 1 })
    expect(bounds).toStrictEqual({
      interfaceScale: { min: 0.8, max: 2 },
      textScale: { min: 0.8, max: 1.75 },
    })
  })

  it('is nothing at either end where the application named neither', async () => {
    request.listThemes.mockResolvedValue(
      createAnswer({ interfaceScaleBounds: undefined, textScaleBounds: undefined }),
    )

    expect((await themes.getAppearance()).bounds).toStrictEqual({
      interfaceScale: { min: 0, max: 0 },
      textScale: { min: 0, max: 0 },
    })
  })
})

describe('a choice written', () => {
  it('sends the mode as the schema carries it and answers why it was refused', async () => {
    request.writeAppearance.mockResolvedValue({ error: 'the file is read-only' })

    const reason = await themes.writeAppearance('mine:sea', 'light')

    expect(reason).toBe('the file is read-only')
    expect(request.writeAppearance.mock.calls[0]?.[0]).toMatchObject({
      name: 'mine:sea',
      mode: THEME_MODE.LIGHT,
    })
  })
})

describe('the rest of what is requested', () => {
  it('returns the text of a theme’s file', async () => {
    request.readTheme.mockResolvedValue({ css: ':root { --numen-surface: black }' })

    expect(await themes.readTheme('mine:sea')).toBe(':root { --numen-surface: black }')
    expect(request.readTheme.mock.calls[0]?.[0]).toMatchObject({ name: 'mine:sea' })
  })

  it('names the themes that changed, as they land', async () => {
    request.watchThemes.mockImplementation(async function* () {
      yield { names: ['mine:sea'] }
      yield { names: ['mine:sea', 'mine:dawn'] }
    })

    const received: (readonly string[])[] = []
    for await (const names of themes.watchThemes(new AbortController().signal))
      received.push(names)

    expect(received).toStrictEqual([['mine:sea'], ['mine:sea', 'mine:dawn']])
  })

  it('stops the request where the signal it was given is aborted', async () => {
    let requestSignal: AbortSignal | undefined
    request.watchThemes.mockImplementation(async function* (
      _request: unknown,
      context: { signal: AbortSignal },
    ) {
      requestSignal = context.signal
      await new Promise((resolve) => context.signal.addEventListener('abort', resolve))
      yield* [] as { names: string[] }[]
    })
    const controller = new AbortController()

    const readTask = (async () => {
      for await (const names of themes.watchThemes(controller.signal)) expect(names).toBeDefined()
    })()
    await vi.waitFor(() => expect(requestSignal).toBeDefined())
    controller.abort()

    await vi.waitFor(() => expect(requestSignal?.aborted).toBe(true))
    await readTask.catch(() => undefined)
  })
})
