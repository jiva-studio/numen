/**
 * The four commands over how the window is drawn: its theme, which half of a
 * colour pair its tokens are read as, and how large it is drawn and set.
 *
 * A theme and a size are worn where the keyboard stands, so what is asked here
 * is what the window is wearing while a person walks the list, and what it is
 * wearing once they leave it.
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { Plex } from '@numen/ui'
import {
  asked,
  cards,
  drawnWithPalette,
  nodeInPlex,
  said,
  settles,
} from './testing/window'
import { IS_MODE, IS_SIZES, IS_THEME, MARKER } from './wearing'

describe('the four commands over how the window is drawn', () => {
  /** What the mode's element holds while the tokens are read as a pair. */
  const PAIR = ':root { color-scheme: light dark; }'
  /** What the page was served wearing, which is the applied theme's file. */
  const SERVED = ':root { --numen-surface: #101014 }'
  /** What the page was served drawn at, which is as it is designed. */
  const SIZED = ':root { --numen-interface-scale: 1; --numen-text-scale: 1; }'

  const styled = (is: string, css: string) => {
    const one = document.createElement('style')
    one.setAttribute(MARKER, is)
    one.textContent = css
    return one
  }

  /** What the head is wearing, in the order the elements stand in it. */
  const dressed = () =>
    [...document.head.querySelectorAll('style')].map((one) => one.textContent)

  const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settles()
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settles()
  }

  /** The page as the window's handler serves it, before the window is drawn. */
  beforeEach(() => {
    for (const one of document.head.querySelectorAll('style')) one.remove()
    document.head.append(
      styled(IS_MODE, PAIR),
      styled(IS_THEME, SERVED),
      styled(IS_SIZES, SIZED),
    )
  })

  /** The keyboard still walking, and the keyboard stood still on a row. */
  const walking = () => new Promise((done) => setTimeout(done, 60))
  const stands = () => new Promise((done) => setTimeout(done, 200))

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

  /** The commands open, and the one the words typed name taken up. */
  const over = async (typed: string) => {
    const window = await drawnWithPalette()
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type(typed)
    await press('Enter')
    return window
  }

  /** Every row the words typed leave, in the order they are drawn. */
  const left = () =>
    [...document.body.querySelectorAll('[data-palette="list"] [role="option"]')].map((one) =>
      one.querySelector('[data-palette="name"]')?.textContent?.trim(),
    )

  /** The groups standing, by the name each carries. */
  const groups = () =>
    [...document.body.querySelectorAll('[data-palette="title"]')].map((one) => one.textContent?.trim())

  /** The second line of every row drawn, and nothing for a row carrying none. */
  const beside = () =>
    [...document.body.querySelectorAll('[data-palette="list"] [role="option"]')].map((one) =>
      one.querySelector('[data-palette="detail"]')?.textContent?.trim(),
    )

  describe('the words a person types for them', () => {
    it('find the theme by “theme”, and light and dark by either word', async () => {
      await drawnWithPalette()
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()

      await type('theme')
      expect(left()).toStrictEqual(['Change the theme'])

      await type('light')
      expect(left()).toStrictEqual(['Light or dark'])

      await type('dark')
      expect(left()).toStrictEqual(['Light or dark'])
    })

    it('find the two sizes by “interface”, by “font” and by “reading”', async () => {
      await drawnWithPalette()
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()

      await type('interface')
      expect(left()).toStrictEqual(['Interface size'])

      await type('font')
      expect(left()).toStrictEqual(['Reading font size'])

      await type('reading')
      expect(left()).toStrictEqual(['Reading font size'])
    })

    it('turn up both of them, and nothing else, for the word they share', async () => {
      await drawnWithPalette()
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()

      await type('size')

      expect(left()).toStrictEqual(['Interface size', 'Reading font size'])
    })
  })

  describe('the step that offers the themes', () => {
    it('draws the shelves as groups, and opens on the theme the window wears', async () => {
      await over('theme')

      expect(groups()).toStrictEqual(['Ships with numen', 'Your own themes'])
      expect(document.body.querySelector('[data-here]')?.textContent).toContain('numen')
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
    })

    it('wears the theme the keyboard walks onto', async () => {
      await over('theme')

      await press('ArrowDown')

      expect(dressed()).toStrictEqual([PAIR, ':root { --numen-surface: mine:sea }', SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('puts back the theme the settings name when the step is left', async () => {
      await over('theme')
      await press('ArrowDown')

      await press('Escape')

      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('keeps wearing the theme that was chosen, and writes it down', async () => {
      await over('theme')
      await press('ArrowDown')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['mine:sea system 1/1'])
      expect(dressed()).toStrictEqual([PAIR, ':root { --numen-surface: mine:sea }', SIZED])
    })
  })

  describe('the step that offers light and dark', () => {
    it('opens on the half the tokens are read as, in a group of its own', async () => {
      await over('light')

      expect(groups()).toStrictEqual(['Light and dark'])
      expect(left()).toStrictEqual(['Follow the system', 'Light', 'Dark'])
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
    })

    it('reads the tokens as the half the keyboard walks onto', async () => {
      await over('light')

      await press('ArrowDown')

      expect(dressed()).toStrictEqual([':root { color-scheme: light; }', SERVED, SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('puts back the half the settings name when the step is left', async () => {
      await over('light')
      await press('ArrowDown')

      await press('Escape')

      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('writes the half that was chosen, leaving the theme where it was', async () => {
      await over('dark')
      await press('End')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['preset:numen dark 1/1'])
      expect(dressed()).toStrictEqual([':root { color-scheme: dark; }', SERVED, SIZED])
    })
  })

  /**
   * The row the keyboard is standing on, read off the document rather than out
   * of the list the window handed the palette.
   */
  const standingOn = () =>
    document.body.querySelector('[data-here] [data-palette="name"]')?.textContent?.trim()

  describe('a step opened over a setting', () => {
    /** The page as the handler serves it, dressed as the settings say. */
    const serves = (mode = PAIR, sizes = SIZED) => {
      for (const one of document.head.querySelectorAll('style')) one.remove()
      document.head.append(
        styled(IS_MODE, mode),
        styled(IS_THEME, SERVED),
        styled(IS_SIZES, sizes),
      )
      return [mode, SERVED, sizes]
    }

    it('stands on the theme the settings name, and goes on wearing it', async () => {
      said.applied = 'mine:sea'
      const was = serves()

      await over('theme')

      expect(standingOn()).toBe('sea')
      expect(dressed()).toStrictEqual(was)
      expect(asked.worn).toStrictEqual([])
    })

    it('stands on the half the tokens are read as, and goes on reading them so', async () => {
      said.mode = 'dark'
      const was = serves(':root { color-scheme: dark; }')

      await over('light')

      expect(standingOn()).toBe('Dark')
      expect(dressed()).toStrictEqual(was)
    })

    it('stands on the size the interface is drawn at, and leaves it there', async () => {
      said.sizes = { interfaceScale: 1.5, textScale: 1 }
      const was = serves(PAIR, ':root { --numen-interface-scale: 1.5; --numen-text-scale: 1; }')

      await over('interface')
      await stands()

      expect(standingOn()).toBe('150%')
      expect(dressed()).toStrictEqual(was)
    })

    it('stands on a size between two steps, which is the row put in for it', async () => {
      said.sizes = { interfaceScale: 1, textScale: 1.17 }
      const was = serves(PAIR, ':root { --numen-interface-scale: 1; --numen-text-scale: 1.17; }')

      await over('reading')
      await stands()

      expect(standingOn()).toBe('117%')
      expect(dressed()).toStrictEqual(was)
    })

    it('leaves the keyboard where typing puts it, and does not walk it back', async () => {
      said.sizes = { interfaceScale: 1.5, textScale: 1 }
      serves(PAIR, ':root { --numen-interface-scale: 1.5; --numen-text-scale: 1; }')
      await over('interface')

      await type('137')

      expect(standingOn()).toBe('137%')
    })
  })

  describe('the step that offers how large the interface is drawn', () => {
    it('opens on the size the window is drawn at, in a group of its own', async () => {
      await over('interface')

      expect(groups()).toStrictEqual(['How large the interface is drawn'])
      expect(left()).toStrictEqual(TENTHS)
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
    })

    it('draws a second line on the one row the window is drawn at, and no other', async () => {
      await over('interface')

      expect(beside()).toStrictEqual(
        TENTHS.map((title) => (title === '100%' ? 'Current' : undefined)),
      )
    })

    it('offers a number typed into the field, and narrows to it alone', async () => {
      await over('interface')

      await type('137')

      expect(left()).toStrictEqual(['137%'])
    })

    it('draws and writes a number typed, the way it does a step', async () => {
      await over('interface')
      await type('137')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['preset:numen system 1.37/1'])
      expect(dressed().at(-1)).toBe(':root { --numen-interface-scale: 1.37; --numen-text-scale: 1; }')
    })

    it('offers no row for a number the range does not reach, and says nothing', async () => {
      await over('interface')

      await type('250')

      expect(left()).toStrictEqual([])
      expect(document.body.querySelector('[data-palette="silence"]')?.textContent?.trim()).toBe('Nothing')
    })

    it('narrows the steps, and offers nothing of its own, for digits inside one', async () => {
      await over('interface')

      await type('15')

      expect(left()).toStrictEqual(['150%'])
    })

    it('holds the size until the keyboard has stood on the row it walked to', async () => {
      await over('interface')
      await press('End')

      await walking()
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])

      await stands()
      expect(dressed().at(-1)).toBe(':root { --numen-interface-scale: 2; --numen-text-scale: 1; }')
      expect(asked.worn).toStrictEqual([])
    })

    it('puts back the size the settings name when the step is left', async () => {
      await over('interface')
      await press('End')
      await stands()

      await press('Escape')

      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
      expect(asked.worn).toStrictEqual([])
    })

    it('keeps drawing at the size that was chosen, and writes it down', async () => {
      await over('interface')
      await press('End')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['preset:numen system 2/1'])
      expect(dressed().at(-1)).toBe(':root { --numen-interface-scale: 2; --numen-text-scale: 1; }')
    })

    it('says what the settings refused, where the window says what it could not do', async () => {
      said.refused = 'appearance.interface_scale is 2, which is outside 0.8 to 1.5'
      const window = await over('interface')
      await press('End')

      await press('Enter')
      await settles()

      expect(cards(window).join(' ')).toContain('outside 0.8 to 1.5')
      expect(dressed()).toStrictEqual([PAIR, SERVED, SIZED])
    })
  })

  describe('the editor of an open note', () => {
    /** A note in a tab of its own, with the editor's measurements taken. */
    const opened = async () => {
      const window = await drawnWithPalette()
      window.findComponent(Plex).vm.$emit('show', nodeInPlex(window), 'here')
      await settles()
      asked.measured = 0
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()
      return window
    }

    it('takes its measurements again at the size the keyboard is held on', async () => {
      await opened()
      await type('interface')
      await press('Enter')

      await press('End')
      await stands()

      expect(asked.measured).toBe(1)
    })

    it('takes them again at the size that was chosen', async () => {
      await opened()
      await type('reading')
      await press('Enter')

      await press('End')
      await press('Enter')
      await settles()

      expect(asked.measured).toBe(1)
    })

    it('is left alone while the keyboard is walking rows, and by a theme', async () => {
      await opened()
      await type('theme')
      await press('Enter')

      await press('ArrowDown')
      await walking()

      expect(asked.measured).toBe(0)
    })
  })

  describe('the step that offers how large the text is set', () => {
    it('opens on the sizes the reading text goes between', async () => {
      await over('reading')

      expect(groups()).toStrictEqual(['How large the text is set'])
      // The far end of this one falls between two steps, and is offered there.
      expect(left()).toStrictEqual([...TENTHS.slice(0, 10), '175%'])
    })

    it('writes the size that was chosen beside the interface’s, which stands', async () => {
      await over('reading')
      await press('End')

      await press('Enter')

      expect(asked.worn).toStrictEqual(['preset:numen system 1/1.75'])
      expect(dressed().at(-1)).toBe(':root { --numen-interface-scale: 1; --numen-text-scale: 1.75; }')
    })
  })
})

