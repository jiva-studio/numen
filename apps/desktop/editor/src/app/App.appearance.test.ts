/**
 * The commands of the settings that decide how the window is drawn and how it works, made from the
 * declarations the core serves: the theme, light or dark, the two sizes, the sync of title and
 * filename, and the performance profile.
 *
 * A theme, a half and a size are applied as the keyboard moves over the list, so the tests check
 * what the window shows while the list is navigated and what it shows after the list is left.
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { requests, cards, mountWindowWithPalette, mockVaultState, settle } from '@/testing/window'
import { SETTINGS_WORDS, type PluginSettingInfo } from '@/entities/settings'

const MARKER = 'data-appearance'
const IS_MODE = 'mode'
const IS_THEME = 'theme'
const IS_SIZES = 'sizes'

/** What the core declares for the settings of the shell that the palette offers. */
const SHELL: readonly PluginSettingInfo[] = [
  {
    key: 'appearance.theme',
    type: 'enum',
    title: 'Theme',
    description: '',
    default: 'preset:numen',
    palette: true,
    icon: 'palette',
    preview: true,
    optionsFrom: 'themes',
  },
  {
    key: 'appearance.mode',
    type: 'enum',
    title: 'Light or dark',
    description: '',
    default: 'system',
    palette: true,
    icon: 'contrast',
    preview: true,
    options: [
      { value: 'system', title: 'System', detail: '' },
      { value: 'light', title: 'Light', detail: '' },
      { value: 'dark', title: 'Dark', detail: '' },
    ],
  },
  {
    key: 'appearance.interfaceScale',
    type: 'range',
    title: 'Interface size',
    description: '',
    default: 1,
    palette: true,
    icon: 'ruler',
    preview: true,
    min: 0.8,
    max: 2,
    step: 0.1,
  },
  {
    key: 'appearance.textScale',
    type: 'range',
    title: 'Reading size',
    description: '',
    default: 1,
    palette: true,
    icon: 'a-large-small',
    preview: true,
    min: 0.8,
    max: 1.75,
    step: 0.1,
  },
  {
    key: 'naming.syncTitleAndFilename',
    type: 'boolean',
    title: 'Sync title and filename',
    description: '',
    default: true,
    palette: true,
    icon: 'refresh-cw',
  },
  {
    key: 'performance.profile',
    type: 'enum',
    title: 'Performance profile',
    description: '',
    default: 'balanced',
    palette: true,
    icon: 'gauge',
    options: [
      { value: 'eco', title: 'Eco', detail: 'Minimal power.' },
      { value: 'balanced', title: 'Balanced', detail: 'Standard workload.' },
      { value: 'maximum', title: 'Maximum', detail: 'All cores.' },
    ],
  },
]

describe('the commands of the settings over how the window is drawn and works', () => {
  /** What the mode's element holds while the tokens are read as a pair. */
  const PAIR = ':root { color-scheme: light dark; }'
  /** The file of the applied theme, as the page was served. */
  const SERVED = ':root { --numen-surface: #101014 }'
  /** What the page was served drawn at, which is as it is designed. */
  const SIZED = ':root { --numen-interface-scale: 1; --numen-text-scale: 1; }'

  const createStyle = (is: string, css: string) => {
    const one = document.createElement('style')
    one.setAttribute(MARKER, is)
    one.textContent = css
    return one
  }

  /** The elements of the head, in order. */
  const getHeadStyles = () =>
    [...document.head.querySelectorAll('style')].map((one) => one.textContent)

  const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settle()
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settle()
  }

  /** The page as the window's handler serves it, before the window is drawn. */
  beforeEach(() => {
    mockVaultState.app = SHELL
    for (const one of document.head.querySelectorAll('style')) one.remove()
    document.head.append(
      createStyle(IS_MODE, PAIR),
      createStyle(IS_THEME, SERVED),
      createStyle(IS_SIZES, SIZED),
    )
  })

  /** The keyboard still moving, and the keyboard settled on a row. */
  const waitUnderHold = () => new Promise((isDone) => setTimeout(isDone, 60))
  const wait = () => new Promise((isDone) => setTimeout(isDone, 200))

  /** Every tenth of the interface scale. */
  const TENTHS = [
    '0.8',
    '0.9',
    '1',
    '1.1',
    '1.2',
    '1.3',
    '1.4',
    '1.5',
    '1.6',
    '1.7',
    '1.8',
    '1.9',
    '2',
  ]

  /** The window drawn with the commands open. */
  const openPalette = async () => {
    const window = await mountWindowWithPalette()
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settle()
    return window
  }

  /** The commands open, and the one the words typed name taken up. */
  const runCommand = async (words: string) => {
    const window = await openPalette()
    await type(words)
    await press('Enter')
    return window
  }

  /** Every row the words typed leave, in the order they are drawn. */
  const getRemainingRows = () =>
    [...document.body.querySelectorAll('[data-palette="list"] [role="option"]')].map((one) =>
      one.querySelector('[data-palette="name"]')?.textContent?.trim(),
    )

  /** The groups standing, by the name each carries. */
  const groups = () =>
    [...document.body.querySelectorAll('[data-palette="title"]')].map((one) =>
      one.textContent?.trim(),
    )

  /** The row the keyboard is standing on, read off the document. */
  const getStandingRow = () =>
    document.body.querySelector('[data-here] [data-palette="name"]')?.textContent?.trim()

  describe('the words a person types for them', () => {
    it.each([
      ['theme', 'Theme'],
      ['light', 'Light or dark'],
      ['interface', 'Interface size'],
      ['reading', 'Reading size'],
      ['profile', 'Performance profile'],
      ['sync', 'Sync title and filename'],
    ])('find the command of “%s”', async (word, title) => {
      await openPalette()

      await type(word)

      expect(getRemainingRows()).toStrictEqual([title])
    })
  })

  describe('the step that offers the themes', () => {
    it('draws the themes in a group named for the setting, and opens on the applied theme', async () => {
      await runCommand('theme')

      expect(groups()).toStrictEqual(['Theme'])
      expect(getRemainingRows()).toStrictEqual(['numen', 'sea'])
      expect(getStandingRow()).toBe('numen')
      expect(getHeadStyles()).toStrictEqual([PAIR, SERVED, SIZED])
    })

    it('applies the theme the keyboard moves to', async () => {
      await runCommand('theme')

      await press('ArrowDown')

      expect(getHeadStyles()).toStrictEqual([PAIR, ':root { --numen-surface: mine:sea }', SIZED])
      expect(requests.settingsWritten).toStrictEqual([])
    })

    it('puts back the theme the settings name when the step is left', async () => {
      await runCommand('theme')
      await press('ArrowDown')

      await press('Escape')

      expect(getHeadStyles()).toStrictEqual([PAIR, SERVED, SIZED])
      expect(requests.settingsWritten).toStrictEqual([])
    })

    it('keeps the chosen theme applied, and writes it down', async () => {
      await runCommand('theme')
      await press('ArrowDown')

      await press('Enter')
      await wait()

      expect(requests.settingsWritten).toStrictEqual(['app.appearance.theme "mine:sea"'])
      expect(getHeadStyles()).toStrictEqual([PAIR, ':root { --numen-surface: mine:sea }', SIZED])
    })

    it('opens on the theme the settings name', async () => {
      mockVaultState.applied = 'mine:sea'

      await runCommand('theme')

      expect(getStandingRow()).toBe('sea')
    })
  })

  describe('the step that offers light and dark', () => {
    it('opens on the half the tokens are read as', async () => {
      await runCommand('light')

      expect(groups()).toStrictEqual(['Light or dark'])
      expect(getRemainingRows()).toStrictEqual(['System', 'Light', 'Dark'])
      expect(getStandingRow()).toBe('System')
    })

    it('reads the tokens as the half the keyboard walks onto', async () => {
      await runCommand('light')

      await press('ArrowDown')

      expect(getHeadStyles()).toStrictEqual([':root { color-scheme: light; }', SERVED, SIZED])
      expect(requests.settingsWritten).toStrictEqual([])
    })

    it('puts back the half the settings name when the step is left', async () => {
      await runCommand('light')
      await press('ArrowDown')

      await press('Escape')

      expect(getHeadStyles()).toStrictEqual([PAIR, SERVED, SIZED])
    })

    it('writes the half that was chosen, leaving the theme where it was', async () => {
      await runCommand('light')
      await press('End')

      await press('Enter')
      await wait()

      expect(requests.settingsWritten).toStrictEqual(['app.appearance.mode "dark"'])
      expect(getHeadStyles()).toStrictEqual([':root { color-scheme: dark; }', SERVED, SIZED])
    })
  })

  describe('the step that offers how large the interface is drawn', () => {
    it('opens on the size the window is drawn at, in a group of its own', async () => {
      await runCommand('interface')

      expect(groups()).toStrictEqual(['Interface size'])
      expect(getRemainingRows()).toStrictEqual(TENTHS)
      expect(getStandingRow()).toBe('1')
    })

    it('holds the size until the keyboard has stood on the row it walked to', async () => {
      await runCommand('interface')
      await press('End')

      await waitUnderHold()
      expect(getHeadStyles()).toStrictEqual([PAIR, SERVED, SIZED])

      await wait()
      expect(getHeadStyles().at(-1)).toBe(
        ':root { --numen-interface-scale: 2; --numen-text-scale: 1; }',
      )
      expect(requests.settingsWritten).toStrictEqual([])
    })

    it('puts back the size the settings name when the step is left', async () => {
      await runCommand('interface')
      await press('End')
      await wait()

      await press('Escape')

      expect(getHeadStyles()).toStrictEqual([PAIR, SERVED, SIZED])
    })

    it('keeps drawing at the size that was chosen, and writes it down', async () => {
      await runCommand('interface')
      await press('End')

      await press('Enter')
      await wait()

      expect(requests.settingsWritten).toStrictEqual(['app.appearance.interfaceScale 2'])
      expect(getHeadStyles().at(-1)).toBe(
        ':root { --numen-interface-scale: 2; --numen-text-scale: 1; }',
      )
    })

    it('says what the settings rejected, where the window says what it could not do', async () => {
      mockVaultState.writeError = 'appearance.interfaceScale is 2, which is outside 0.8 to 1.5'
      await runCommand('interface')
      await press('End')

      await press('Enter')
      await wait()

      expect(cards().join(' ')).toContain(SETTINGS_WORDS.unturned)
      expect(getHeadStyles()).toStrictEqual([PAIR, SERVED, SIZED])
    })
  })

  describe('the step that offers how large the text is set', () => {
    it('ends on the largest size the reading text can have, which falls between two steps', async () => {
      await runCommand('reading')

      expect(groups()).toStrictEqual(['Reading size'])
      expect(getRemainingRows().at(-1)).toBe('1.75')
    })

    it('writes the size that was chosen alongside the interface’s, which remains', async () => {
      await runCommand('reading')
      await press('End')

      await press('Enter')
      await wait()

      expect(requests.settingsWritten).toStrictEqual(['app.appearance.textScale 1.75'])
      expect(getHeadStyles().at(-1)).toBe(
        ':root { --numen-interface-scale: 1; --numen-text-scale: 1.75; }',
      )
    })
  })

  describe('the steps of the settings that are not previewed', () => {
    it('offers the two rows of the sync of title and filename, and writes the one chosen', async () => {
      await runCommand('sync')

      expect(getRemainingRows()).toStrictEqual(['On', 'Off'])
      await press('ArrowDown')
      await press('Enter')

      expect(requests.settingsWritten).toStrictEqual(['app.naming.syncTitleAndFilename false'])
    })

    it('offers the profiles with what each says, and writes the one chosen', async () => {
      await runCommand('profile')

      expect(getRemainingRows()).toStrictEqual(['Eco', 'Balanced', 'Maximum'])
      await press('End')
      await press('Enter')

      expect(requests.settingsWritten).toStrictEqual(['app.performance.profile "maximum"'])
    })
  })
})
