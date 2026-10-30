/**
 * The one thing this module decides on its own: how the application's account
 * of the way the window is drawn reads in the window's own words.
 *
 * Everything else hands a message across unchanged, and the schema says what
 * those look like.
 */
import { describe, expect, it, vi } from 'vitest'
import { Mode as Modes, Origin } from '@numen/protocol'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

vi.mock('@/shared/clients', () => ({ theme: request }))

const request = {
  listThemes: vi.fn(),
  readTheme: vi.fn(),
  writeAppearance: vi.fn(),
  watchThemes: vi.fn(),
}

const { themes } = await import('./theme')

/** What the application answers about a window drawn as designed. */
const createAnswer = (overrides: Record<string, unknown> = {}) => ({
  themes: [],
  applied: 'preset/Numen.css',
  mode: Modes.SYSTEM,
  interfaceScale: 1,
  textScale: 1,
  interfaceScaleBounds: { min: 0.8, max: 2 },
  textScaleBounds: { min: 0.8, max: 1.75 },
  ...overrides,
})

describe('every theme there is', () => {
  it('says which ship inside the application and which are the person’s own', async () => {
    request.listThemes.mockResolvedValue(
      createAnswer({
        themes: [
          { name: 'preset/Numen.css', title: 'Numen', origin: Origin.PRESET, isPinned: false },
          { name: 'own/Dusk.css', title: 'Dusk', origin: Origin.MINE, isPinned: true },
        ],
      }),
    )

    const { themes: every } = await themes.getAppearance()
    expect(every).toEqual([
      { name: 'preset/Numen.css', title: 'Numen', isBuiltIn: true, isPinned: false },
      { name: 'own/Dusk.css', title: 'Dusk', isBuiltIn: false, isPinned: true },
    ])
  })
})

describe('which half of a colour pair is read', () => {
  it('is returned in the window’s own words', async () => {
    for (const [mode, word] of [
      [Modes.LIGHT, 'light'],
      [Modes.DARK, 'dark'],
      [Modes.SYSTEM, 'system'],
    ] as const) {
      request.listThemes.mockResolvedValue(createAnswer({ mode }))
      expect((await themes.getAppearance()).mode).toBe(word)
    }
  })

  // A mode the window has no word for is one the machine decides, which is what
  // a window that was never told anything is drawn as.
  it('is the system’s where the window has no word for the transcript', async () => {
    request.listThemes.mockResolvedValue(createAnswer({ mode: 99 }))
    expect((await themes.getAppearance()).mode).toBe('system')
  })

  it('is carried back to the application as the schema names it', async () => {
    request.writeAppearance.mockResolvedValue({ error: '' })
    await themes.writeAppearance('own/Dusk.css', 'dark', { interfaceScale: 1.25, textScale: 1.5 })

    expect(request.writeAppearance.mock.calls[0]?.[0]).toEqual({
      name: 'own/Dusk.css',
      mode: Modes.DARK,
      interfaceScale: 1.25,
      textScale: 1.5,
    })
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
    expect(request.readTheme.mock.calls[0]?.[0]).toEqual({ name: 'own/Dusk.css' })
  })

  it('says why a choice failed, and says nothing where it was not', async () => {
    request.writeAppearance.mockResolvedValue({ error: 'that size is outside its bounds' })
    expect(
      await themes.writeAppearance('preset/Numen.css', 'light', {
        interfaceScale: 9,
        textScale: 1,
      }),
    ).toBe('that size is outside its bounds')

    request.writeAppearance.mockResolvedValue({ error: '' })
    expect(
      await themes.writeAppearance('preset/Numen.css', 'light', {
        interfaceScale: 1,
        textScale: 1,
      }),
    ).toBe('')
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

  it('is requested for under the signal it was given', async () => {
    request.watchThemes.mockImplementation(async function* () {
      yield* [] as string[][]
    })
    const controller = new AbortController()

    const receivedThemes: (readonly string[])[] = []
    for await (const names of themes.watchThemes(controller.signal)) receivedThemes.push(names)

    expect(request.watchThemes).toHaveBeenLastCalledWith({}, { signal: controller.signal })
  })
})
