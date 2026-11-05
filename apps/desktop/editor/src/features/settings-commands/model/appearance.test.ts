/**
 * Appearance handling without a window.
 *
 * Three behaviors are verified: that the theme's element is written after the
 * mode's, so the theme takes precedence; that leaving a list restores the value
 * from the settings without reading a file; and that a size is applied only
 * after the keyboard focus has stayed on its row.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { DESIGNED } from '@/entities/settings'
import type { Appearance, Themes } from '@/entities/settings'
import { IS_MODE, IS_SIZES, IS_THEME, MARKER } from '../lib/head'
import { createWindowAppearance } from './appearance'
import { WORDS as words } from '@/shared/words'

/** The theme the page was served with. */
const SERVED = ':root { --numen-surface: #101014 }'

/** What the mode's element holds while the tokens are read as a pair. */
const PAIR = ':root { color-scheme: light dark; }'

/** What the sizes' element holds while the window is drawn as designed. */
const SIZED = ':root { --numen-interface-scale: 1; --numen-text-scale: 1; }'

const createStyle = (sheet: Document, is: string, css: string): HTMLStyleElement => {
  const one = sheet.createElement('style')
  one.setAttribute(MARKER, is)
  one.textContent = css
  return one
}

/** A page as the window's handler serves one: the link, then the three elements. */
const page = (css = SERVED): Document => {
  const sheet = document.implementation.createHTMLDocument('numen')
  sheet.head.append(
    sheet.createElement('link'),
    createStyle(sheet, IS_MODE, PAIR),
    createStyle(sheet, IS_THEME, css),
    createStyle(sheet, IS_SIZES, SIZED),
  )
  return sheet
}

/** The elements of the head, in order. */
const getHeadStyles = (sheet: Document) =>
  [...sheet.head.querySelectorAll('style')].map((one) => one.textContent)

/** What the theme's element holds, which is the second of the three. */
const getThemeCss = (sheet: Document) => getHeadStyles(sheet)[1]

/** What the sizes' element holds, which is the last of them. */
const sizes = (sheet: Document) => getHeadStyles(sheet)[2]

const APPEARANCE: Appearance = {
  themes: [
    { name: 'preset:numen', title: 'numen', isBuiltIn: true, pluginId: '', isPinned: false },
    { name: 'preset:dracula', title: 'dracula', isBuiltIn: true, pluginId: '', isPinned: true },
    { name: 'mine:sea', title: 'sea', isBuiltIn: false, pluginId: '', isPinned: false },
  ],
  applied: 'preset:numen',
  mode: 'system',
  sizes: { interfaceScale: 1, textScale: 1 },
  bounds: { interfaceScale: { min: 0.8, max: 2 }, textScale: { min: 0.8, max: 1.75 } },
}

/** A short delay for pending application requests to complete. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/** A delay longer than the interval after which the window applies a size. */
const wait = () => new Promise((isDone) => setTimeout(isDone, 200))

/** The user's folder, which a test writes to. */
const folder = () => {
  let wake: ((names: readonly string[]) => void) | null = null
  return {
    watchThemes: async function* (): AsyncIterable<readonly string[]> {
      for (;;) yield await new Promise<readonly string[]>((now) => (wake = now))
    },
    /** These themes changed, and the window has answered for them. */
    changeThemes: async (...names: readonly string[]) => {
      wake?.(names)
      await settle()
    },
  }
}

/** A window with a theme applied, over a page a test can read the head of. */
const window = (overrides: Partial<Appearance> = {}, css = SERVED) => {
  const sheet = page(css)
  const response = folder()
  const request: string[] = []
  let appearance: Appearance = { ...APPEARANCE, ...overrides }
  let items = 0
  let textError = ''
  const texts: Record<string, string> = {}
  const core: Themes = {
    getAppearance: async () => {
      items += 1
      return appearance
    },
    readTheme: async (name) => {
      request.push(name)
      if (textError) throw new Error(textError)
      return texts[name] ?? `:root { --numen-surface: ${name} }`
    },
    watchThemes: response.watchThemes,
  }
  /** The notifications sent to the window, in order. */
  const notifiedCalls: { text: string; kind: string }[] = []
  return {
    appearance: createWindowAppearance({
      core,
      words,
      write: (text, kind = 'info') => {
        notifiedCalls.push({ text, kind })
      },
      sheet,
    }),
    sheet,
    requests: request,
    messages: notifiedCalls,
    changeThemes: response.changeThemes,
    getAppearanceCount: () => items,
    writeTheme: (name: string, css: string) => (texts[name] = css),
    failsText: (error: string) => (textError = error),
    setAppearance: (next: Partial<Appearance>) => (appearance = { ...appearance, ...next }),
  }
}

/** The window listing its themes, over a page served with a theme. */
const startWindow = async (overrides: Partial<Appearance> = {}, css = SERVED) => {
  const one = window(overrides, css)
  await one.appearance.start()
  await settle()
  return one
}

describe('the page as it was served', () => {
  it('applies the theme the page arrived with, and reads no file to do it', async () => {
    const one = await startWindow()

    expect(getHeadStyles(one.sheet)).toStrictEqual([PAIR, SERVED, SIZED])
    expect(one.requests).toStrictEqual([])
  })

  it('writes over the theme’s element, and moves none of them', async () => {
    const one = await startWindow()
    const before = [...one.sheet.head.querySelectorAll('style')]

    one.appearance.previewTheme('mine:sea')
    await settle()

    expect([...one.sheet.head.querySelectorAll('style')]).toStrictEqual(before)
    expect(one.sheet.head.lastElementChild).toBe(before[2])
    expect(getHeadStyles(one.sheet)).toStrictEqual([
      PAIR,
      ':root { --numen-surface: mine:sea }',
      SIZED,
    ])
  })

  it('notifies the three apart by the mark each carries', async () => {
    // A theme's file is user-supplied CSS: it may set the color scheme of the
    // mode's element and a multiplier on the sizes' element.
    const one = await startWindow({}, ':root { color-scheme: dark; --numen-text-scale: 1.3 }')

    one.appearance.previewTheme('mine:sea')
    await settle()

    expect(getHeadStyles(one.sheet)).toStrictEqual([
      PAIR,
      ':root { --numen-surface: mine:sea }',
      SIZED,
    ])
  })

  it('makes the pair itself, mode first, for a page served in nothing', async () => {
    const sheet = document.implementation.createHTMLDocument('numen')
    const bareAppearance = createWindowAppearance({
      core: {
        getAppearance: async () => APPEARANCE,
        readTheme: async (name) => `:root { --numen-surface: ${name} }`,
        watchThemes: async function* (): AsyncIterable<readonly string[]> {
          await new Promise<never>(vi.fn())
        },
      },
      words,
      write: vi.fn(),
      sheet,
    })
    await bareAppearance.start()

    bareAppearance.previewTheme('mine:sea')
    await settle()

    expect(getHeadStyles(sheet)).toStrictEqual([PAIR, ':root { --numen-surface: mine:sea }', SIZED])
  })
})

describe('the theme the keyboard is standing on', () => {
  it('is applied while active there', async () => {
    const one = await startWindow()

    one.appearance.previewTheme('mine:sea')
    await settle()

    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: mine:sea }')
    expect(one.requests).toStrictEqual(['mine:sea'])
  })

  it('gives way to the one the settings name once the keyboard remains nowhere', async () => {
    const one = await startWindow()

    one.appearance.previewTheme('mine:sea')
    await settle()
    one.appearance.restore()
    await settle()

    expect(getHeadStyles(one.sheet)).toStrictEqual([PAIR, SERVED, SIZED])
    // The theme the page arrived in is not read for again.
    expect(one.requests).toStrictEqual(['mine:sea'])
  })

  it('is read once, however often the keyboard walks back over it', async () => {
    const one = await startWindow()

    one.appearance.previewTheme('mine:sea')
    await settle()
    one.appearance.previewTheme('preset:dracula')
    await settle()
    one.appearance.previewTheme('mine:sea')
    await settle()

    expect(one.requests).toStrictEqual(['mine:sea', 'preset:dracula'])
  })

  it('is returned to be unreadable where its file is, and the window keeps the applied theme', async () => {
    vi.spyOn(console, 'error').mockImplementation(vi.fn())
    const one = await startWindow()
    one.failsText('the file is gone')

    one.appearance.previewTheme('mine:sea')
    await settle()

    expect(one.messages.at(-1)).toStrictEqual({ text: words.themeUnreadable, kind: 'error' })
    expect(getHeadStyles(one.sheet)).toStrictEqual([PAIR, SERVED, SIZED])
  })

  it('is the last row the keyboard landed on, whatever order the files come back in', async () => {
    const one = await startWindow()

    one.appearance.previewTheme('preset:dracula')
    one.appearance.previewTheme('mine:sea')
    await settle()

    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: mine:sea }')
  })
})

describe('which half of a pair the tokens are read as', () => {
  it('is written into the mode’s element, and leaves the theme where it was', async () => {
    const one = await startWindow()

    one.appearance.previewMode('dark')
    await settle()

    expect(getHeadStyles(one.sheet)).toStrictEqual([':root { color-scheme: dark; }', SERVED, SIZED])
  })

  it('goes back to what the settings say once the keyboard remains nowhere', async () => {
    const one = await startWindow({ mode: 'light' })

    one.appearance.previewMode('dark')
    await settle()
    one.appearance.restore()
    await settle()

    expect(getHeadStyles(one.sheet).at(0)).toBe(':root { color-scheme: light; }')
  })
})

describe('the size the keyboard is standing on', () => {
  it('is not drawn while the keyboard is still walking over rows', async () => {
    const one = await startWindow()

    one.appearance.previewSize('interfaceScale', 1.5)
    await settle()
    one.appearance.previewSize('interfaceScale', 1.75)
    await settle()

    expect(sizes(one.sheet)).toBe(SIZED)
  })

  it('cancels a pending size hold when the keyboard moves to a theme row', async () => {
    const one = await startWindow()

    one.appearance.previewSize('interfaceScale', 1.5)
    one.appearance.previewTheme('mine:sea')
    await wait()

    expect(sizes(one.sheet)).toBe(SIZED)
  })

  it('is drawn once the keyboard has stood on it', async () => {
    const one = await startWindow()

    one.appearance.previewSize('interfaceScale', 1.5)
    await wait()

    expect(sizes(one.sheet)).toBe(':root { --numen-interface-scale: 1.5; --numen-text-scale: 1; }')
  })

  it('is drawn once, at the row the keyboard came to rest on', async () => {
    const one = await startWindow()

    one.appearance.previewSize('interfaceScale', 1.25)
    one.appearance.previewSize('interfaceScale', 1.5)
    one.appearance.previewSize('textScale', 1.25)
    await wait()

    expect(sizes(one.sheet)).toBe(':root { --numen-interface-scale: 1; --numen-text-scale: 1.25; }')
  })

  it('gives way to the size the settings name once the keyboard remains nowhere', async () => {
    const one = await startWindow()
    one.appearance.previewSize('textScale', 1.5)
    await wait()

    one.appearance.restore()
    await settle()

    expect(sizes(one.sheet)).toBe(SIZED)
  })

  it('leaves the theme and the mode where they stand', async () => {
    const one = await startWindow()

    one.appearance.previewSize('interfaceScale', 2)
    await wait()

    expect(getHeadStyles(one.sheet).slice(0, 2)).toStrictEqual([PAIR, SERVED])
  })
})

describe('the person editing their own theme file', () => {
  it('is followed: the file is read again, and the window applies what the file now says', async () => {
    const one = await startWindow()
    one.appearance.previewTheme('mine:sea')
    await settle()

    one.writeTheme('mine:sea', ':root { --numen-surface: #001 }')
    await one.changeThemes('mine:sea')

    expect(one.requests).toStrictEqual(['mine:sea', 'mine:sea'])
    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: #001 }')
  })

  it('is followed while the theme is the one the settings name', async () => {
    const one = await startWindow({ applied: 'mine:sea' })

    one.writeTheme('mine:sea', ':root { --numen-surface: #002 }')
    await one.changeThemes('mine:sea')

    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: #002 }')
  })

  it('leaves a theme the folder returned nothing about where it was', async () => {
    const one = await startWindow()
    one.appearance.previewTheme('mine:sea')
    await settle()

    await one.changeThemes('mine:tide')

    expect(one.requests).toStrictEqual(['mine:sea'])
    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: mine:sea }')
  })

  it('reads the settings again, so a file made or taken out is counted', async () => {
    const one = await startWindow()

    await one.changeThemes('mine:sea')

    expect(one.getAppearanceCount()).toBe(2)
  })
})

describe('the settings read again while a row is previewed', () => {
  it('shows what the settings now name, and forgets what the keyboard stood on', async () => {
    const one = await startWindow()
    one.appearance.previewTheme('mine:sea')
    one.appearance.previewSize('textScale', 1.5)
    await wait()
    one.setAppearance({ applied: 'preset:dracula' })

    await one.appearance.reload()

    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: preset:dracula }')
    expect(one.appearance.sized.value).toStrictEqual({ interfaceScale: 1, textScale: 1 })
  })
})

/** The first moments of a window and the stream it follows, with every answer in a test's hands. */
const createManual = (waits = Infinity) => {
  const sheet = page()
  const reads: { name: string; give: (css: string) => void; fail: () => void }[] = []
  const signals: AbortSignal[] = []
  let fault = false
  let wake: ((names: readonly string[] | null) => void) | null = null
  let waited = 0
  const core: Themes = {
    getAppearance: async () => APPEARANCE,
    readTheme: (name) =>
      new Promise<string>((give, fail) =>
        reads.push({ name, give, fail: () => fail(new Error('gone')) }),
      ),
    watchThemes: async function* (signal) {
      signals.push(signal)
      if (fault) throw new Error('down')
      while (!signal.aborted) {
        const names = await new Promise<readonly string[] | null>((next) => {
          wake = next
          signal.addEventListener('abort', () => next(null))
        })
        if (names === null) return
        yield names
      }
    },
  }
  const notifiedCalls: string[] = []
  const windowAppearance = createWindowAppearance({
    core,
    words,
    write: (text) => {
      notifiedCalls.push(text)
    },
    sheet,
    wait: () =>
      waited++ < waits ? new Promise((isDone) => setTimeout(isDone, 0)) : new Promise(vi.fn()),
  })
  return {
    appearance: windowAppearance,
    sheet,
    reads,
    signals,
    messages: notifiedCalls,
    fails: () => (fault = true),
    changeThemes: async (...names: readonly string[]) => {
      wake?.(names)
      await settle()
    },
  }
}

describe('the window before it has been told anything', () => {
  it('has no themes, applies no theme, reads as a system pair, and draws at the designed size', () => {
    const { appearance } = createManual()

    expect(appearance.applied.value).toBe('')
    expect(appearance.mode.value).toBe('system')
    expect(appearance.sized.value).toStrictEqual({ interfaceScale: DESIGNED, textScale: DESIGNED })
  })

  it('writes nothing over the page it was served, whatever the keyboard relies on', async () => {
    const { appearance, sheet, reads } = createManual()

    appearance.previewTheme('mine:sea')
    await settle()

    expect(reads).toStrictEqual([])
    expect(getHeadStyles(sheet)).toStrictEqual([PAIR, SERVED, SIZED])
  })
})

describe('files that come back out of order', () => {
  it('leave the theme of the row the keyboard came to rest on', async () => {
    const { appearance, sheet, reads } = createManual()
    await appearance.start()
    appearance.previewTheme('preset:dracula')
    appearance.previewTheme('mine:sea')

    reads[1]?.give('SEA')
    await settle()
    reads[0]?.give('DRACULA')
    await settle()

    expect(getThemeCss(sheet)).toBe('SEA')
  })

  it('say nothing of a file the keyboard has already left', async () => {
    const { appearance, messages, reads } = createManual()
    await appearance.start()
    appearance.previewTheme('preset:dracula')
    appearance.previewTheme('mine:sea')

    reads[1]?.give('SEA')
    await settle()
    reads[0]?.fail()
    await settle()

    expect(messages).toStrictEqual([])
  })
})

describe('the folder when it names a theme that is not applied', () => {
  it('does not read the applied theme again', async () => {
    const { appearance, reads, changeThemes } = createManual()
    await appearance.start()
    await settle()
    appearance.previewTheme('mine:sea')

    await changeThemes('mine:tide')

    expect(reads.map((one) => one.name)).toStrictEqual(['mine:sea'])
  })

  it('is empty at all, and the window is not requested for its list again', async () => {
    const one = window()
    await one.appearance.start()
    await settle()

    await one.changeThemes()

    expect(one.getAppearanceCount()).toBe(1)
  })
})

describe('the folder when it cannot be followed', () => {
  it('says the window lost touch, and keeps saying it until the stream is back', async () => {
    const { appearance, fails } = createManual(0)
    fails()

    await appearance.start()
    await settle()

    expect(appearance.lost.value).not.toBe('')
  })
})

describe('a window that is closed', () => {
  it('lets go of the folder it listens to', async () => {
    const { appearance, signals } = createManual()
    await appearance.start()

    appearance.close()

    expect(signals.map((one) => one.aborted)).toStrictEqual([true])
  })

  it('does not listen to the folder again once the stream has ended', async () => {
    const { appearance, signals } = createManual(3)
    await appearance.start()
    await settle()

    appearance.close()
    await settle()
    await settle()

    expect(signals).toHaveLength(1)
  })

  it('draws no size the keyboard was still standing on', async () => {
    const one = await startWindow()

    one.appearance.previewSize('interfaceScale', 1.5)
    one.appearance.close()
    await wait()

    expect(one.appearance.sized.value).toStrictEqual({ interfaceScale: 1, textScale: 1 })
  })
})

describe('the application that cannot list the themes', () => {
  it('is returned, as an error, and the window keeps the applied theme', async () => {
    const sheet = page()
    const notifiedCalls: { text: string; kind: string }[] = []
    const windowAppearance = createWindowAppearance({
      core: {
        getAppearance: async () => {
          throw new Error('down')
        },
        readTheme: async () => '',
        watchThemes: async function* (signal) {
          await new Promise<void>((isDone) => signal.addEventListener('abort', () => isDone()))
          yield* []
        },
      },
      words,
      write: (text, kind = 'info') => {
        notifiedCalls.push({ text, kind })
      },
      sheet,
    })

    await windowAppearance.start()

    expect(notifiedCalls).toStrictEqual([{ text: words.unlisted, kind: 'error' }])
    expect(getHeadStyles(sheet)).toStrictEqual([PAIR, SERVED, SIZED])
    windowAppearance.close()
  })
})

describe('a folder stream that ended', () => {
  afterEach(() => vi.useRealTimers())

  it('is followed again after the wait the window keeps by default', async () => {
    vi.useFakeTimers()
    let watches = 0
    const windowAppearance = createWindowAppearance({
      core: {
        getAppearance: async () => APPEARANCE,
        readTheme: async () => '',
        watchThemes: async function* () {
          watches += 1
          yield* []
        },
      },
      words,
      write: vi.fn(),
      sheet: page(),
    })
    await windowAppearance.start()
    await vi.advanceTimersByTimeAsync(0)
    expect(watches).toBe(1)

    await vi.advanceTimersByTimeAsync(1000)

    expect(watches).toBe(2)
    windowAppearance.close()
  })
})
