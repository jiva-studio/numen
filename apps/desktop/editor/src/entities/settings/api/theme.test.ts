/**
 * The one thing this module decides on its own: how the application's description of
 * the window's appearance is expressed in the window's own terms.
 *
 * Everything else passes a message through unchanged, and the schema defines their
 * shape.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { THEME_MODE, THEME_ORIGIN } from '@numen/core-client'
import { installCoreAnswers } from '@numen/core-client/testing'
import { themes } from './theme'

const request = {
  listThemes: vi.fn(),
  readTheme: vi.fn(),
  watchThemes: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  restore = installCoreAnswers({ theme: request })
})

afterEach(() => {
  restore()
})

const createAnswer = (overrides: Record<string, unknown> = {}) => ({
  themes: [],
  applied: 'preset/Numen.css',
  mode: THEME_MODE.SYSTEM,
  interfaceScale: 1,
  textScale: 1,
  interfaceScaleBounds: { min: 0.8, max: 2 },
  textScaleBounds: { min: 0.8, max: 1.75 },
  ...overrides,
})

describe('every theme there is', () => {
  it('says which ship inside the application, which a plugin contributes and which are the person’s own', async () => {
    request.listThemes.mockResolvedValue(
      createAnswer({
        themes: [
          {
            name: 'preset/Numen.css',
            title: 'Numen',
            origin: THEME_ORIGIN.PRESET,
            isPinned: false,
            pluginId: '',
          },
          {
            name: 'own/Dusk.css',
            title: 'Dusk',
            origin: THEME_ORIGIN.MINE,
            isPinned: true,
            pluginId: '',
          },
          {
            name: 'plugin:org.example.themes/night',
            title: 'Night',
            origin: THEME_ORIGIN.PLUGIN,
            isPinned: true,
            pluginId: 'org.example.themes',
          },
        ],
      }),
    )

    const { themes: every } = await themes.getAppearance()
    expect(every).toEqual([
      { name: 'preset/Numen.css', title: 'Numen', isBuiltIn: true, pluginId: '', isPinned: false },
      { name: 'own/Dusk.css', title: 'Dusk', isBuiltIn: false, pluginId: '', isPinned: true },
      {
        name: 'plugin:org.example.themes/night',
        title: 'Night',
        isBuiltIn: false,
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
    ] as const) {
      request.listThemes.mockResolvedValue(createAnswer({ mode }))
      expect((await themes.getAppearance()).mode).toBe(word)
    }
  })

  // A mode the window has no term for is decided by the machine, which is how a
  // window that was never configured is drawn.
  it('is the system’s where the window has no word for the transcript', async () => {
    request.listThemes.mockResolvedValue(createAnswer({ mode: 99 }))
    expect((await themes.getAppearance()).mode).toBe('system')
  })
})

describe('how far a size goes', () => {
  it('is the two ends the application named', async () => {
    request.listThemes.mockResolvedValue(createAnswer())
    const { sizes, bounds } = await themes.getAppearance()

    expect(sizes).toEqual({ interfaceScale: 1, textScale: 1 })
    expect(bounds).toEqual({
      interfaceScale: { min: 0.8, max: 2 },
      textScale: { min: 0.8, max: 1.75 },
    })
  })

  // A bound the application left out is no bound at all, and not a bound of
  // some number the window made up.
  it('is nothing at either end where the application named neither', async () => {
    request.listThemes.mockResolvedValue(
      createAnswer({ interfaceScaleBounds: undefined, textScaleBounds: undefined }),
    )

    expect((await themes.getAppearance()).bounds).toEqual({
      interfaceScale: { min: 0, max: 0 },
      textScale: { min: 0, max: 0 },
    })
  })
})

describe('the rest of what is requested', () => {
  it('returns the text of a theme’s file currently', async () => {
    request.readTheme.mockResolvedValue({ css: ':root { --numen-surface: black }' })
    expect(await themes.readTheme('own/Dusk.css')).toBe(':root { --numen-surface: black }')
    expect(request.readTheme.mock.calls[0]?.[0]).toMatchObject({ name: 'own/Dusk.css' })
  })

  it('names the themes the person’s folder changed, as they land', async () => {
    request.watchThemes.mockImplementation(async function* () {
      yield { names: ['own/Dusk.css'] }
      yield { names: ['own/Dusk.css', 'own/Dawn.css'] }
    })

    const receivedThemes: (readonly string[])[] = []
    for await (const names of themes.watchThemes(new AbortController().signal))
      receivedThemes.push(names)

    expect(receivedThemes).toEqual([['own/Dusk.css'], ['own/Dusk.css', 'own/Dawn.css']])
  })

  it('stops the request where the signal it was given is aborted', async () => {
    let requestSignal: AbortSignal | undefined
    request.watchThemes.mockImplementation(async function* (_request, context) {
      requestSignal = context.signal
      await new Promise((resolve) => context.signal.addEventListener('abort', resolve))
      yield* [] as { names: string[] }[]
    })
    const controller = new AbortController()

    const received: (readonly string[])[] = []
    const readTask = (async () => {
      for await (const names of themes.watchThemes(controller.signal)) received.push(names)
    })()
    await vi.waitFor(() => expect(requestSignal).toBeDefined())
    controller.abort()

    await vi.waitFor(() => expect(requestSignal?.aborted).toBe(true))
    await readTask.catch(() => undefined)
  })
})
