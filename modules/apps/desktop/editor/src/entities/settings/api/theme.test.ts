/**
 * The one thing this module decides on its own: how the application's account
 * of the way the window is drawn reads in the window's own words.
 *
 * Everything else hands a message across unchanged, and the schema says what
 * those look like.
 */
import { describe, expect, it, vi } from 'vitest'
import { Mode as Modes, Shelf } from '@numen/protocol'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

vi.mock('@/shared/clients', () => ({ theme: asked }))

const asked = {
  listThemes: vi.fn(),
  readTheme: vi.fn(),
  writeAppearance: vi.fn(),
  watchThemes: vi.fn(),
}

const { themes } = await import('./theme')

/** What the application answers about a window drawn as designed. */
const createAnswer = (over: Record<string, unknown> = {}) => ({
  themes: [],
  applied: 'preset/Numen.css',
  mode: Modes.SYSTEM,
  interfaceScale: 1,
  textScale: 1,
  interfaceScaleBounds: { least: 0.8, most: 2 },
  textScaleBounds: { least: 0.8, most: 1.75 },
  ...over,
})

describe('every theme there is', () => {
  it('says which ship inside the application and which are the person’s own', async () => {
    asked.listThemes.mockResolvedValue(
      createAnswer({
        themes: [
          { name: 'preset/Numen.css', title: 'Numen', shelf: Shelf.PRESET, pinned: false },
          { name: 'own/Dusk.css', title: 'Dusk', shelf: Shelf.MINE, pinned: true },
        ],
      }),
    )

    const { themes: every } = await themes.appearance()
    expect(every).toEqual([
      { name: 'preset/Numen.css', title: 'Numen', isBuiltIn: true, isPinned: false },
      { name: 'own/Dusk.css', title: 'Dusk', isBuiltIn: false, isPinned: true },
    ])
  })
})

describe('which half of a colour pair is read', () => {
  it('is said in the window’s own words', async () => {
    for (const [said, word] of [
      [Modes.LIGHT, 'light'],
      [Modes.DARK, 'dark'],
      [Modes.SYSTEM, 'system'],
    ] as const) {
      asked.listThemes.mockResolvedValue(createAnswer({ mode: said }))
      expect((await themes.appearance()).mode).toBe(word)
    }
  })

  // A mode the window has no word for is one the machine decides, which is what
  // a window that was never told anything is drawn as.
  it('is the system’s where the window has no word for what was said', async () => {
    asked.listThemes.mockResolvedValue(createAnswer({ mode: 99 }))
    expect((await themes.appearance()).mode).toBe('system')
  })

  it('is carried back to the application as the schema names it', async () => {
    asked.writeAppearance.mockResolvedValue({ error: '' })
    await themes.chooses('own/Dusk.css', 'dark', { interfaceScale: 1.25, textScale: 1.5 })

    expect(asked.writeAppearance.mock.calls[0]?.[0]).toEqual({
      name: 'own/Dusk.css',
      mode: Modes.DARK,
      interfaceScale: 1.25,
      textScale: 1.5,
    })
  })
})

describe('how far a size goes', () => {
  it('is the two ends the application named', async () => {
    asked.listThemes.mockResolvedValue(createAnswer())
    const { sizes, bounds } = await themes.appearance()

    expect(sizes).toEqual({ interfaceScale: 1, textScale: 1 })
    expect(bounds).toEqual({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    })
  })

  // A bound the application left out is no bound at all, and not a bound of
  // some number the window made up.
  it('is nothing at either end where the application named neither', async () => {
    asked.listThemes.mockResolvedValue(
      createAnswer({ interfaceScaleBounds: undefined, textScaleBounds: undefined }),
    )

    expect((await themes.appearance()).bounds).toEqual({
      interfaceScale: { least: 0, most: 0 },
      textScale: { least: 0, most: 0 },
    })
  })
})

describe('the rest of what is asked', () => {
  it('hands back the text of a theme’s file as it stands', async () => {
    asked.readTheme.mockResolvedValue({ css: ':root { --numen-surface: black }' })
    expect(await themes.text('own/Dusk.css')).toBe(':root { --numen-surface: black }')
    expect(asked.readTheme.mock.calls[0]?.[0]).toEqual({ name: 'own/Dusk.css' })
  })

  it('says why a choice was refused, and says nothing where it was not', async () => {
    asked.writeAppearance.mockResolvedValue({ error: 'that size is outside its bounds' })
    expect(await themes.chooses('preset/Numen.css', 'light', { interfaceScale: 9, textScale: 1 })).toBe(
      'that size is outside its bounds',
    )

    asked.writeAppearance.mockResolvedValue({ error: '' })
    expect(await themes.chooses('preset/Numen.css', 'light', { interfaceScale: 1, textScale: 1 })).toBe(
      '',
    )
  })

  it('names the themes the person’s folder changed, as they land', async () => {
    asked.watchThemes.mockImplementation(async function* () {
      yield { names: ['own/Dusk.css'] }
      yield { names: ['own/Dusk.css', 'own/Dawn.css'] }
    })

    const heard: (readonly string[])[] = []
    for await (const names of themes.changed(new AbortController().signal)) heard.push(names)

    expect(heard).toEqual([['own/Dusk.css'], ['own/Dusk.css', 'own/Dawn.css']])
  })
})
