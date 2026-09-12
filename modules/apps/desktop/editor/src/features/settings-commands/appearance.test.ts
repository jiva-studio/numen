/**
 * How the window is drawn, without a window.
 *
 * Three things are asked here: that the theme's element is the one written
 * over, since the mode's has to stand before it; that leaving a list puts back
 * what the settings name without reading a file for it; and that a size is
 * drawn only once the keyboard has stood on its row.
 */
import { describe, expect, it, vi } from 'vitest'
import type { StepGroup } from '@/features/command-palette/@x/settings-commands'
import type { Appearance, Themes } from '@/entities/settings'
import { IS_MODE, IS_SIZES, IS_THEME, MARKER } from './lib/head'
import { INTERFACE_SCALE, TEXT_SCALE, windowAppearance } from './appearance'
import { WORDS as words } from '@/shared/words'

/** What the page was served wearing. */
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
const page = (served = SERVED): Document => {
  const sheet = document.implementation.createHTMLDocument('numen')
  sheet.head.append(
    sheet.createElement('link'),
    createStyle(sheet, IS_MODE, PAIR),
    createStyle(sheet, IS_THEME, served),
    createStyle(sheet, IS_SIZES, SIZED),
  )
  return sheet
}

/** What the head is wearing, in the order the elements stand in it. */
const getHeadStyles = (sheet: Document) =>
  [...sheet.head.querySelectorAll('style')].map((one) => one.textContent)

/** What the theme's element holds, which is the second of the three. */
const getThemeCss = (sheet: Document) => getHeadStyles(sheet)[1]

/** What the sizes' element holds, which is the last of them. */
const sizes = (sheet: Document) => getHeadStyles(sheet)[2]

const APPEARANCE: Appearance = {
  themes: [
    { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
    { name: 'preset:dracula', title: 'dracula', isBuiltIn: true, isPinned: true },
    { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
  ],
  applied: 'preset:numen',
  mode: 'system',
  sizes: { interfaceScale: 1, textScale: 1 },
  bounds: { interfaceScale: { least: 0.8, most: 2 }, textScale: { least: 0.8, most: 1.75 } },
}

/** A moment for whatever was asked of the application to come back. */
const settle = () => new Promise((done) => setTimeout(done, 0))

/** The keyboard stood on a row for longer than the window holds a size. */
const wait = () => new Promise((done) => setTimeout(done, 200))

/** A range narrower than the one the application answers with. */
const NARROW = { least: 1, most: 1.5 }

/** Every tenth the interface goes between, as a person reads them. */
const TENTHS = [
  '80%',
  '90%',
  '100%',
  '110%',
  '120%',
  '130%',
  '140%',
  '150%',
  '160%',
  '170%',
  '180%',
  '190%',
  '200%',
]

/** The person's folder, which a test writes to. */
const folder = () => {
  let wake: ((names: readonly string[]) => void) | null = null
  return {
    changed: async function* (): AsyncIterable<readonly string[]> {
      for (;;) yield await new Promise<readonly string[]>((now) => (wake = now))
    },
    /** These themes changed, and the window has answered for them. */
    says: async (...names: readonly string[]) => {
      wake?.(names)
      await settle()
    },
  }
}

/** The window wearing a theme, over a page a test can read the head of. */
const window = (over: Partial<Appearance> = {}, served = SERVED) => {
  const sheet = page(served)
  const said = folder()
  const asked: string[] = []
  const chosen: string[] = []
  let appearance: Appearance = { ...APPEARANCE, ...over }
  let listed = 0
  let failed = ''
  let refused = ''
  const texts: Record<string, string> = {}
  const core: Themes = {
    appearance: async () => {
      listed += 1
      return appearance
    },
    text: async (name) => {
      asked.push(name)
      if (refused) throw new Error(refused)
      return texts[name] ?? `:root { --numen-surface: ${name} }`
    },
    chooses: async (name, mode, sizes) => {
      chosen.push(`${name} ${mode} ${sizes.interfaceScale}/${sizes.textScale}`)
      return failed
    },
    changed: said.changed,
  }
  /** What the window was told, in the order it was told. */
  const told: { text: string; kind: string }[] = []
  return {
    worn: windowAppearance(
      core,
      words,
      (text, kind = 'report') => void told.push({ text, kind }),
      sheet,
    ),
    sheet,
    asked,
    chosen,
    told,
    says: said.says,
    listed: () => listed,
    writes: (name: string, css: string) => (texts[name] = css),
    fails: (why: string) => (failed = why),
    refuses: (why: string) => (refused = why),
    holds: (next: Partial<Appearance>) => (appearance = { ...appearance, ...next }),
  }
}

/** The window listing what it can wear, over a page it was served dressed. */
const startWindow = async (over: Partial<Appearance> = {}, served = SERVED) => {
  const one = window(over, served)
  await one.worn.start()
  await settle()
  return one
}

describe('the page as it was served', () => {
  it('wears what it arrived in, and reads no file to do it', async () => {
    const one = await startWindow()

    expect(getHeadStyles(one.sheet)).toStrictEqual([PAIR, SERVED, SIZED])
    expect(one.asked).toStrictEqual([])
  })

  it('writes over the theme’s element, and moves none of them', async () => {
    const one = await startWindow()
    const before = [...one.sheet.head.querySelectorAll('style')]

    one.worn.previewItem('mine:sea')
    await settle()

    expect([...one.sheet.head.querySelectorAll('style')]).toStrictEqual(before)
    expect(one.sheet.head.lastElementChild).toBe(before[2])
    expect(getHeadStyles(one.sheet)).toStrictEqual([
      PAIR,
      ':root { --numen-surface: mine:sea }',
      SIZED,
    ])
  })

  it('tells the three apart by the mark each carries', async () => {
    // A theme's file is a person's own CSS: it may pin the scheme the mode's
    // element holds and name a multiplier the sizes' element holds.
    const one = await startWindow({}, ':root { color-scheme: dark; --numen-text-scale: 1.3 }')

    one.worn.previewItem('mine:sea')
    await settle()

    expect(getHeadStyles(one.sheet)).toStrictEqual([
      PAIR,
      ':root { --numen-surface: mine:sea }',
      SIZED,
    ])
  })

  it('makes the pair itself, mode first, for a page served in nothing', async () => {
    const sheet = document.implementation.createHTMLDocument('numen')
    const bare = windowAppearance(
      {
        appearance: async () => APPEARANCE,
        text: async (name) => `:root { --numen-surface: ${name} }`,
        chooses: async () => '',
        changed: async function* (): AsyncIterable<readonly string[]> {
          await new Promise<never>(() => {})
        },
      },
      words,
      () => {},
      sheet,
    )
    await bare.start()

    bare.previewItem('mine:sea')
    await settle()

    expect(getHeadStyles(sheet)).toStrictEqual([PAIR, ':root { --numen-surface: mine:sea }'])
  })
})

describe('the theme the keyboard is standing on', () => {
  it('is worn while it stands there', async () => {
    const one = await startWindow()

    one.worn.previewItem('mine:sea')
    await settle()

    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: mine:sea }')
    expect(one.asked).toStrictEqual(['mine:sea'])
  })

  it('gives way to the one the settings name once the keyboard stands nowhere', async () => {
    const one = await startWindow()

    one.worn.previewItem('mine:sea')
    await settle()
    one.worn.previewItem('')
    await settle()

    expect(getHeadStyles(one.sheet)).toStrictEqual([PAIR, SERVED, SIZED])
    // The theme the page arrived in is not read for again.
    expect(one.asked).toStrictEqual(['mine:sea'])
  })

  it('is read once, however often the keyboard walks back over it', async () => {
    const one = await startWindow()

    one.worn.previewItem('mine:sea')
    await settle()
    one.worn.previewItem('preset:dracula')
    await settle()
    one.worn.previewItem('mine:sea')
    await settle()

    expect(one.asked).toStrictEqual(['mine:sea', 'preset:dracula'])
  })

  it('is said to be unreadable where its file is, and the window keeps what it wears', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    const one = await startWindow()
    one.refuses('the file is gone')

    one.worn.previewItem('mine:sea')
    await settle()

    expect(one.told.at(-1)).toStrictEqual({ text: words.unworn, kind: 'error' })
    expect(getHeadStyles(one.sheet)).toStrictEqual([PAIR, SERVED, SIZED])
  })

  it('is the last row the keyboard landed on, whatever order the files come back in', async () => {
    const one = await startWindow()

    one.worn.previewItem('preset:dracula')
    one.worn.previewItem('mine:sea')
    await settle()

    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: mine:sea }')
  })
})

describe('which half of a pair the tokens are read as', () => {
  it('is written into the mode’s element, and leaves the theme where it was', async () => {
    const one = await startWindow()

    one.worn.previewItem('mode:dark')
    await settle()

    expect(getHeadStyles(one.sheet)).toStrictEqual([
      ':root { color-scheme: dark; }',
      SERVED,
      SIZED,
    ])
  })

  it('goes back to what the settings say once the keyboard stands nowhere', async () => {
    const one = await startWindow({ mode: 'light' })

    one.worn.previewItem('mode:dark')
    await settle()
    one.worn.previewItem('')
    await settle()

    expect(getHeadStyles(one.sheet).at(0)).toBe(':root { color-scheme: light; }')
  })
})

describe('the themes the step offers', () => {
  /** Every group, by its identity, and the rows standing in each. */
  const getGroupIds = (rows: readonly StepGroup[]) =>
    Object.fromEntries(rows.map((group) => [group.id, group.items.map((row) => row.id)]))

  it('draws the themes in the two groups they come off, and nothing else', async () => {
    const one = await startWindow()

    expect(getGroupIds(one.worn.getThemeGroups())).toStrictEqual({
      shipping: ['preset:numen', 'preset:dracula'],
      owned: ['mine:sea'],
    })
  })

  it('stands the group the theme worn came off first, and it first inside it', async () => {
    const one = await startWindow({ applied: 'preset:dracula' })
    const other = await startWindow({ applied: 'mine:sea' })

    expect(getGroupIds(one.worn.getThemeGroups())).toStrictEqual({
      shipping: ['preset:dracula', 'preset:numen'],
      owned: ['mine:sea'],
    })
    expect(other.worn.getThemeGroups().map((group) => group.id)).toStrictEqual(['owned', 'shipping'])
  })

  it('says on a row only what is true of that row: that it is the one worn', async () => {
    const one = await startWindow()

    const rows = one.worn.getThemeGroups().flatMap((group) => group.items)
    expect(rows.map((row) => row.detail)).toStrictEqual([words.current, undefined, undefined])
  })

  it('says where a person’s own themes go while they have none', async () => {
    const one = await startWindow({ themes: APPEARANCE.themes.slice(0, 2) })

    const own = one.worn.getThemeGroups().find((group) => group.id === 'owned')
    expect(own?.items).toStrictEqual([])
    expect(own?.silence).toBe(words.noneOwned)
  })
})

describe('the three halves the step offers', () => {
  const rows = (one: Awaited<ReturnType<typeof startWindow>>) =>
    one.worn.modes().flatMap((group) => group.items)

  it('draws the three in one group of their own, and says which is read', async () => {
    const one = await startWindow({ mode: 'light' })

    expect(one.worn.modes().map((group) => group.id)).toStrictEqual(['half'])
    expect(rows(one).map((row) => row.id)).toStrictEqual(['mode:system', 'mode:light', 'mode:dark'])
    expect(rows(one).map((row) => row.detail)).toStrictEqual([undefined, words.current, undefined])
  })

  it('draws each as not to be chosen while the theme worn pins light and dark', async () => {
    const one = await startWindow()

    one.worn.previewItem('preset:dracula')
    await settle()

    expect(rows(one).map((row) => row.disabled)).toStrictEqual([true, true, true])
    expect(rows(one).map((row) => row.detail)).toStrictEqual([
      words.pinned,
      words.pinned,
      words.pinned,
    ])
  })

  it('draws each as one to choose again once such a theme is left', async () => {
    const one = await startWindow()

    one.worn.previewItem('preset:dracula')
    await settle()
    one.worn.previewItem('')
    await settle()

    expect(rows(one).map((row) => row.disabled)).toStrictEqual([undefined, undefined, undefined])
  })
})

describe('the sizes the two steps offer', () => {
  const rows = (one: Awaited<ReturnType<typeof startWindow>>, command: string) =>
    one.worn.sizes(command).flatMap((group) => group.items)

  it('walks each range from end to end, in quarters, with both ends on it', async () => {
    const one = await startWindow()

    expect(rows(one, INTERFACE_SCALE).map((row) => row.title)).toStrictEqual(TENTHS)
    // The far end of this one falls between two steps, and is offered there.
    expect(rows(one, TEXT_SCALE).map((row) => row.title)).toStrictEqual([
      ...TENTHS.slice(0, 10),
      '175%',
    ])
  })

  it('draws each in a group of its own, named for what that size moves', async () => {
    const one = await startWindow()

    expect(one.worn.sizes(INTERFACE_SCALE).map((group) => group.title)).toStrictEqual([words.drawing])
    expect(one.worn.sizes(TEXT_SCALE).map((group) => group.title)).toStrictEqual([words.setting])
  })

  it('offers nothing at all until the application has said how far a size goes', async () => {
    const one = await startWindow({
      bounds: { interfaceScale: { least: 0, most: 0 }, textScale: NARROW },
    })

    expect(rows(one, INTERFACE_SCALE)).toStrictEqual([])
    expect(rows(one, TEXT_SCALE).map((row) => row.title)).toStrictEqual([
      '100%',
      '110%',
      '120%',
      '130%',
      '140%',
      '150%',
    ])
  })

  it('says on one row that it is the size now, and says nothing on any other', async () => {
    const one = await startWindow({ sizes: { interfaceScale: 1.5, textScale: 1 } })

    expect(rows(one, INTERFACE_SCALE).map((row) => row.detail)).toStrictEqual(
      TENTHS.map((title) => (title === '150%' ? words.current : undefined)),
    )
  })

  it('holds the size the window is drawn at, wherever between the steps it falls', async () => {
    const one = await startWindow({ sizes: { interfaceScale: 1.17, textScale: 1 } })

    const rung = rows(one, INTERFACE_SCALE).find((row) => row.detail === words.current)
    expect(rung?.title).toBe('117%')
    expect(rows(one, INTERFACE_SCALE).map((row) => row.title)).toStrictEqual([
      ...TENTHS.slice(0, 4),
      '117%',
      ...TENTHS.slice(4),
    ])
  })

  it('says nothing beside 100%, which a person reading percentages knows', async () => {
    const one = await startWindow({ sizes: { interfaceScale: 1.5, textScale: 1 } })

    const getHundredDetail = (command: string) =>
      rows(one, command).find((row) => row.title === '100%')?.detail

    expect(getHundredDetail(INTERFACE_SCALE)).toBeUndefined()
    // The reading text is set at 100%, so on its list that row is the one.
    expect(getHundredDetail(TEXT_SCALE)).toBe(words.current)
  })
})

describe('the number a person types at a size', () => {
  const rows = (one: Awaited<ReturnType<typeof startWindow>>, command: string, typed: string) =>
    one.worn.sizes(command, typed).flatMap((group) => group.items)

  it('stands as a row of its own, in its place between the steps', async () => {
    const one = await startWindow()

    expect(rows(one, INTERFACE_SCALE, '137').map((row) => row.title)).toStrictEqual([
      ...TENTHS.slice(0, 6),
      '137%',
      ...TENTHS.slice(6),
    ])
    expect(rows(one, INTERFACE_SCALE, '137').find((row) => row.title === '137%')?.id).toBe(
      'interfaceScale:1.37',
    )
  })

  it('is taken with the sign a person reads on the rows, and with none', async () => {
    const one = await startWindow()

    const getTitles = (typed: string) => rows(one, INTERFACE_SCALE, typed).map((row) => row.title)
    expect(getTitles('137%')).toStrictEqual(getTitles('137'))
    expect(getTitles(' 137 ')).toStrictEqual(getTitles('137'))
  })

  it('is not offered at all where the range does not reach it', async () => {
    const one = await startWindow()

    expect(rows(one, INTERFACE_SCALE, '250').map((row) => row.title)).toStrictEqual(TENTHS)
    expect(rows(one, TEXT_SCALE, '190').map((row) => row.title)).not.toContain('190%')
  })

  it('is not offered twice where the list already holds that size', async () => {
    const one = await startWindow()

    expect(rows(one, INTERFACE_SCALE, '150').map((row) => row.title)).toStrictEqual(TENTHS)
  })

  it('is nothing at all where what was typed is not a whole number', async () => {
    const one = await startWindow()

    expect(rows(one, INTERFACE_SCALE, 'large').map((row) => row.title)).toStrictEqual(TENTHS)
    expect(rows(one, INTERFACE_SCALE, '1.37').map((row) => row.title)).toStrictEqual(TENTHS)
  })

  it('is drawn and written like any other row', async () => {
    const one = await startWindow()

    one.worn.previewItem('interfaceScale:1.37')
    await wait()
    await one.worn.chooseItem('interfaceScale:1.37')

    expect(one.chosen).toStrictEqual(['preset:numen system 1.37/1'])
    expect(sizes(one.sheet)).toBe(':root { --numen-interface-scale: 1.37; --numen-text-scale: 1; }')
  })
})

describe('the size the keyboard is standing on', () => {
  it('is not drawn while the keyboard is still walking over rows', async () => {
    const one = await startWindow()

    one.worn.previewItem('interfaceScale:1.5')
    await settle()
    one.worn.previewItem('interfaceScale:1.75')
    await settle()

    expect(sizes(one.sheet)).toBe(SIZED)
  })

  it('is drawn once the keyboard has stood on it', async () => {
    const one = await startWindow()

    one.worn.previewItem('interfaceScale:1.5')
    await wait()

    expect(sizes(one.sheet)).toBe(':root { --numen-interface-scale: 1.5; --numen-text-scale: 1; }')
  })

  it('is drawn once, at the row the keyboard came to rest on', async () => {
    const one = await startWindow()

    one.worn.previewItem('interfaceScale:1.25')
    one.worn.previewItem('interfaceScale:1.5')
    one.worn.previewItem('textScale:1.25')
    await wait()

    expect(sizes(one.sheet)).toBe(':root { --numen-interface-scale: 1; --numen-text-scale: 1.25; }')
  })

  it('gives way to the size the settings name once the keyboard stands nowhere', async () => {
    const one = await startWindow()
    one.worn.previewItem('textScale:1.5')
    await wait()

    one.worn.previewItem('')
    await settle()

    expect(sizes(one.sheet)).toBe(SIZED)
    expect(one.chosen).toStrictEqual([])
  })

  it('leaves the theme and the mode where they stand', async () => {
    const one = await startWindow()

    one.worn.previewItem('interfaceScale:2')
    await wait()

    expect(getHeadStyles(one.sheet).slice(0, 2)).toStrictEqual([PAIR, SERVED])
  })
})

describe('the size that was chosen', () => {
  it('is drawn at once, whatever the hold was waiting for, and written down', async () => {
    const one = await startWindow()
    one.worn.previewItem('interfaceScale:1.25')

    await one.worn.chooseItem('interfaceScale:1.5')

    expect(one.chosen).toStrictEqual(['preset:numen system 1.5/1'])
    expect(sizes(one.sheet)).toBe(':root { --numen-interface-scale: 1.5; --numen-text-scale: 1; }')
  })

  it('is written beside the other size, which stands where it was', async () => {
    const one = await startWindow({ sizes: { interfaceScale: 1.25, textScale: 1 } })

    await one.worn.chooseItem('textScale:1.75')

    expect(one.chosen).toStrictEqual(['preset:numen system 1.25/1.75'])
    expect(sizes(one.sheet)).toBe(':root { --numen-interface-scale: 1.25; --numen-text-scale: 1.75; }')
  })

  it('says what the settings refused, and goes back to the size they hold', async () => {
    const one = await startWindow()
    one.fails('appearance.interface_scale is 3, which is outside 0.8 to 2')

    await one.worn.chooseItem('interfaceScale:2')
    await settle()

    expect(one.told.at(-1)).toStrictEqual({
      text: 'appearance.interface_scale is 3, which is outside 0.8 to 2',
      kind: 'error',
    })
    expect(sizes(one.sheet)).toBe(SIZED)
  })

  it('is nothing at all where the range the window holds does not reach it', async () => {
    const one = await startWindow()

    await one.worn.chooseItem('interfaceScale:3')
    one.worn.previewItem('textScale:3')
    await wait()

    expect(one.chosen).toStrictEqual([])
    expect(sizes(one.sheet)).toBe(SIZED)
  })
})

describe('the row that was chosen', () => {
  it('is written into the settings, and is what the window wears from then on', async () => {
    const one = await startWindow()

    await one.worn.chooseItem('mine:sea')

    expect(one.chosen).toStrictEqual(['mine:sea system 1/1'])
    expect(one.worn.applied.value).toBe('mine:sea')
    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: mine:sea }')
  })

  it('is the mode, written beside the theme the settings already name', async () => {
    const one = await startWindow()

    await one.worn.chooseItem('mode:dark')

    expect(one.chosen).toStrictEqual(['preset:numen dark 1/1'])
    expect(one.worn.mode.value).toBe('dark')
    expect(getHeadStyles(one.sheet)).toStrictEqual([
      ':root { color-scheme: dark; }',
      SERVED,
      SIZED,
    ])
  })

  it('says what the settings could not be written, and puts back what they hold', async () => {
    const one = await startWindow()
    one.fails('the settings could not be written')

    await one.worn.chooseItem('mine:sea')

    expect(one.told.at(-1)).toStrictEqual({
      text: 'the settings could not be written',
      kind: 'error',
    })
    expect(one.worn.applied.value).toBe('preset:numen')
    expect(getHeadStyles(one.sheet)).toStrictEqual([PAIR, SERVED, SIZED])
  })

  it('is nothing at all where the window holds no such row', async () => {
    const one = await startWindow()

    await one.worn.chooseItem('mine:tide')

    expect(one.chosen).toStrictEqual([])
    expect(getHeadStyles(one.sheet)).toStrictEqual([PAIR, SERVED, SIZED])
  })
})

describe('the person editing their own theme file', () => {
  it('is followed: the file is read again, and the window wears what it now says', async () => {
    const one = await startWindow()
    one.worn.previewItem('mine:sea')
    await settle()

    one.writes('mine:sea', ':root { --numen-surface: #001 }')
    await one.says('mine:sea')

    expect(one.asked).toStrictEqual(['mine:sea', 'mine:sea'])
    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: #001 }')
  })

  it('is followed while the theme is the one the settings name', async () => {
    const one = await startWindow({ applied: 'mine:sea' })

    one.writes('mine:sea', ':root { --numen-surface: #002 }')
    await one.says('mine:sea')

    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: #002 }')
  })

  it('leaves a theme the folder said nothing about where it was', async () => {
    const one = await startWindow()
    one.worn.previewItem('mine:sea')
    await settle()

    await one.says('mine:tide')

    expect(one.asked).toStrictEqual(['mine:sea'])
    expect(getThemeCss(one.sheet)).toBe(':root { --numen-surface: mine:sea }')
  })

  it('lists what the folder holds again, so a file made or taken out is offered', async () => {
    const one = await startWindow()
    one.holds({ themes: [...APPEARANCE.themes.slice(0, 2)] })

    await one.says('mine:sea')

    expect(one.listed()).toBe(2)
    expect(one.worn.getThemeGroups().flatMap((group) => group.items.map((row) => row.id))).toStrictEqual([
      'preset:numen',
      'preset:dracula',
    ])
  })
})
