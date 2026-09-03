/**
 * What the palette does, not where it puts things. The negatives matter most:
 * a key that reaches an action nobody offered, a band's name the keyboard stops
 * on, and a list that quietly moves what is lit while a person is aiming at it
 * all look right in a picture.
 *
 * It is drawn at the end of the document, so everything here is read off the
 * document rather than off the wrapper.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import Palette from './Palette.vue'
import type { PaletteBand, PaletteKeys } from './model'
import { MANY } from './fixtures/actions'

/** A keystroke that reaches an item away from the palette. */
const OPTION_1: PaletteKeys = { icons: ['option'], letter: '1' }

const OPEN = [{ id: 'open', text: 'Open the note' }]
const BOTH = [
  { id: 'travel', text: 'Show in plex' },
  { id: 'open', text: 'Open the note' },
]

const OFFERING: PaletteBand[] = [
  {
    id: 'names',
    title: 'Names',
    items: [
      { id: 'entropy', title: 'Entropy', actions: MANY, keys: OPTION_1 },
      { id: 'enthalpy', title: 'Enthalpy', actions: OPEN },
    ],
  },
]

const SECTIONS: PaletteBand[] = [
  {
    id: 'names',
    title: 'Names',
    items: [
      { id: 'entropy', title: 'Entropy', at: [{ from: 0, to: 3 }], actions: BOTH },
      { id: 'enthalpy', title: 'Enthalpy', actions: BOTH },
    ],
  },
  {
    id: 'text',
    title: 'Text',
    items: [
      { id: 'stuck', title: 'Not this one', disabled: true, actions: OPEN },
      { id: 'engine', title: 'Heat engine', detail: 'a reversible engine', actions: OPEN },
    ],
  },
]

type PaletteProps = InstanceType<typeof Palette>['$props']

let mounted: { unmount: () => void } | null = null

const mountPalette = (props: Partial<PaletteProps> = {}) => {
  const palette = mount(Palette, {
    props: { bands: SECTIONS, open: true, ...props },
    attachTo: document.body,
  })
  mounted = palette
  return palette
}

/** Twice: it lands on an item and takes the keyboard a tick after it opens. */
const settle = async () => {
  await nextTick()
  await nextTick()
}

const drawn = () => document.body.querySelector<HTMLElement>('[data-palette="ground"]')
const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')
const options = () =>
  Array.from(document.body.querySelectorAll<HTMLElement>('[data-palette="list"] [role="option"]'))
const lit = () => document.body.querySelector<HTMLElement>('[data-here]')
const keys = () => Array.from(document.body.querySelectorAll<HTMLElement>('[data-palette="key"]'))

/** What a line says, with the runs it is written in run together. */
const said = (of: Element | null | undefined): string =>
  (of?.textContent ?? '').replace(/\s+/g, ' ').trim()

/** The keystroke a cap is announced as, which is all of it a reader hears. */
const spoken = (cap: Element | null | undefined): string => said(cap?.querySelector('.sr-only'))

/** The marks a cap draws, by the name Lucide files each under. */
const marksOf = (cap: Element | null | undefined): readonly string[] =>
  Array.from(cap?.querySelectorAll('svg') ?? []).map(
    (mark) => /lucide-([a-z-]+)-icon/.exec(mark.getAttribute('class') ?? '')?.[1] ?? '',
  )

const sheet = () => document.body.querySelector<HTMLElement>('[data-actions="panel"]')
const hunt = () => document.body.querySelector<HTMLInputElement>('[data-actions="hunt"]')
const deeds = () =>
  Array.from(document.body.querySelectorAll<HTMLElement>('[data-actions="list"] [role="option"]'))
const litDeed = () =>
  document.body.querySelector<HTMLElement>('[data-actions="list"] [role="option"][data-here]')

const pressOn = async (on: Element | null, key: string, more: KeyboardEventInit = {}) => {
  const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...more })
  on?.dispatchEvent(event)
  await nextTick()
  return event
}

const press = (key: string, more: KeyboardEventInit = {}) => pressOn(field(), key, more)

const typeIn = async (into: HTMLInputElement | null, text: string) => {
  if (!into) return
  into.value = text
  into.dispatchEvent(new Event('input'))
  await nextTick()
}

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

describe('being open and being closed', () => {
  it('draws nothing at all until it is opened', async () => {
    mountPalette({ open: false })
    await settle()
    expect(drawn()).toBeNull()
  })

  it('takes the keyboard into the field, and keeps it there while walking', async () => {
    mountPalette()
    await settle()
    expect(document.activeElement).toBe(field())

    await press('ArrowDown')
    expect(document.activeElement).toBe(field())
  })

  it('gives the keyboard back to whatever opened it', async () => {
    const opener = document.createElement('button')
    document.body.append(opener)

    const palette = mountPalette({ from: opener })
    await settle()
    await palette.setProps({ open: false })
    await settle()

    expect(document.activeElement).toBe(opener)
    opener.remove()
  })

  it('asks to be put away on Escape and on a press outside the panel', async () => {
    const palette = mountPalette()
    await settle()

    await press('Escape')
    expect(palette.emitted('dismiss')).toHaveLength(1)

    drawn()?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    await nextTick()
    expect(palette.emitted('dismiss')).toHaveLength(2)
  })

  it('does not ask to be put away when the press was on the panel itself', async () => {
    const palette = mountPalette()
    await settle()

    document.body
      .querySelector('[data-palette="panel"]')
      ?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    await nextTick()

    expect(palette.emitted('dismiss')).toBeUndefined()
  })
})

describe('typing', () => {
  it('says what was typed and chooses nothing by it', async () => {
    const palette = mountPalette()
    await settle()

    const input = field()!
    input.value = 'ent'
    input.dispatchEvent(new Event('input'))
    await nextTick()

    expect(palette.emitted('update:modelValue')?.at(-1)).toEqual(['ent'])
    expect(palette.emitted('choose')).toBeUndefined()
  })
})

describe('walking the list', () => {
  it('opens with the first item lit', async () => {
    mountPalette()
    await settle()
    expect(lit()?.textContent).toContain('Entropy')
  })

  it('crosses a band and never stops on the name of one', async () => {
    mountPalette()
    await settle()

    await press('ArrowDown')
    expect(lit()?.textContent).toContain('Enthalpy')

    await press('ArrowDown')
    expect(lit()?.textContent).toContain('Heat engine')
    expect(lit()?.textContent).not.toContain('Not this one')
  })

  it('names what is lit to a screen reader rather than focusing it', async () => {
    mountPalette()
    await settle()
    await press('ArrowDown')

    expect(field()?.getAttribute('aria-activedescendant')).toBe(options()[1]?.id)
    expect(options()[1]?.getAttribute('aria-selected')).toBe('true')
    expect(options()[0]?.getAttribute('aria-selected')).toBe('false')
  })

  it('draws an item that cannot be chosen and never lights it', async () => {
    mountPalette()
    await settle()

    expect(options()).toHaveLength(4)
    expect(options()[2]?.textContent).toContain('Not this one')

    for (const _ of options()) await press('ArrowDown')
    expect(document.body.querySelector('[data-off][data-here]')).toBeNull()
  })
})

describe('where the keyboard is standing', () => {
  /** A pointer over an item, from wherever the one before it was. */
  const passOver = async (item: Element | undefined, at: number) => {
    item?.dispatchEvent(
      new PointerEvent('pointermove', { bubbles: true, clientX: at, clientY: at }),
    )
    await nextTick()
  }

  it('says the item it opens on, and every item walked to', async () => {
    const palette = mountPalette()
    await settle()
    expect(palette.emitted('lit')).toEqual([['entropy']])

    await press('ArrowDown')
    await press('ArrowDown')
    expect(palette.emitted('lit')).toEqual([['entropy'], ['enthalpy'], ['engine']])
  })

  it('says nothing where there is nothing to stand on', async () => {
    const palette = mountPalette({ bands: [{ id: 'names', title: 'Names', items: [] }] })
    await settle()

    await press('ArrowDown')
    expect(palette.emitted('lit')).toBeUndefined()
  })

  it('says the item a pointer that has moved is over, and nothing for one standing still', async () => {
    const palette = mountPalette()
    await settle()

    await passOver(options()[1], 40)
    expect(palette.emitted('lit')?.at(-1)).toEqual(['enthalpy'])

    // The list scrolled under a pointer that never moved, and the row under it
    // is another one.
    await passOver(options()[3], 40)
    expect(palette.emitted('lit')).toHaveLength(2)
  })

  it('says nothing for a row the keyboard passes over', async () => {
    const palette = mountPalette()
    await settle()

    await passOver(options()[2], 40)
    await press('ArrowDown')
    await press('ArrowDown')

    expect(palette.emitted('lit')?.flat()).not.toContain('stuck')
  })

  it('says once what a fresh list settled on, and not the emptiness before it', async () => {
    const palette = mountPalette()
    await settle()

    await palette.setProps({ modelValue: 'heat', bands: [SECTIONS[1]!] })
    await settle()

    expect(palette.emitted('lit')).toEqual([['entropy'], ['engine']])
  })

  it('says nothing when a band lands and what is lit stays where it was', async () => {
    const palette = mountPalette({ bands: [SECTIONS[1]!] })
    await settle()

    await palette.setProps({ bands: SECTIONS })
    await settle()

    expect(palette.emitted('lit')).toEqual([['engine']])
  })

  it('stands on nothing once it is put away', async () => {
    const palette = mountPalette()
    await settle()

    await palette.setProps({ open: false })
    await settle()

    expect(palette.emitted('lit')?.at(-1)).toEqual([''])
  })
})

describe('choosing', () => {
  it('reaches the first action with Enter and the second with Shift', async () => {
    const palette = mountPalette()
    await settle()

    await press('Enter')
    expect(palette.emitted('choose')?.at(-1)).toEqual(['entropy', 'travel'])

    await press('Enter', { shiftKey: true })
    expect(palette.emitted('choose')?.at(-1)).toEqual(['entropy', 'open'])
  })

  it('reaches nothing with Shift when the item offers one action', async () => {
    const palette = mountPalette()
    await settle()

    await press('ArrowDown')
    await press('ArrowDown')
    expect(lit()?.textContent).toContain('Heat engine')

    await press('Enter', { shiftKey: true })
    expect(palette.emitted('choose')).toBeUndefined()

    await press('Enter')
    expect(palette.emitted('choose')?.at(-1)).toEqual(['engine', 'open'])
  })

  it('chooses nothing when no band holds anything', async () => {
    const palette = mountPalette({ bands: [{ id: 'names', title: 'Names', items: [] }] })
    await settle()

    await press('Enter')
    expect(palette.emitted('choose')).toBeUndefined()
  })

  it('chooses nothing by pressing an item that cannot be chosen', async () => {
    const palette = mountPalette()
    await settle()

    options()[2]?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()

    expect(palette.emitted('choose')).toBeUndefined()
  })

  it('says what the keys reach for the item that is lit', async () => {
    mountPalette()
    await settle()
    expect(keys().map(said)).toEqual(['Return Show in plex', 'Shift Return Open the note'])

    await press('ArrowDown')
    await press('ArrowDown')
    expect(keys().map(said)).toEqual(['Return Open the note'])
  })

  it('draws each key held as its own mark, in the order it is held', async () => {
    mountPalette()
    await settle()
    expect(keys().map((key) => marksOf(key.querySelector('.cap')))).toEqual([
      ['corner-down-left'],
      ['arrow-big-up', 'corner-down-left'],
    ])
  })
})

describe('a band arriving while it is being read', () => {
  it('keeps the keyboard on the item it was on', async () => {
    const palette = mountPalette({ bands: [SECTIONS[1]!] })
    await settle()
    expect(lit()?.textContent).toContain('Heat engine')

    await palette.setProps({ bands: SECTIONS })
    await settle()

    expect(lit()?.textContent).toContain('Heat engine')
    expect(options()).toHaveLength(4)
  })

  it('hands the keyboard to the first item when the one it was on is gone', async () => {
    const palette = mountPalette()
    await settle()
    await press('ArrowDown')
    expect(lit()?.textContent).toContain('Enthalpy')

    await palette.setProps({ bands: [SECTIONS[1]!] })
    await settle()

    expect(lit()?.textContent).toContain('Heat engine')
  })

  it('lights the first answer to a fresh question, however many have gone before', async () => {
    const palette = mountPalette()
    await settle()
    await press('ArrowDown')

    // What a person typed, and then the answers to it. The list holds what the
    // question before turned up until each band lands.
    await palette.setProps({ modelValue: 'heat' })
    await settle()
    await palette.setProps({ bands: [SECTIONS[1]!] })
    await settle()

    expect(lit()?.textContent).toContain('Heat engine')
    expect(field()?.getAttribute('aria-activedescendant')).not.toBeNull()
  })

  it('leaves what a person lit where it is when another band lands', async () => {
    const palette = mountPalette({ bands: [SECTIONS[1]!] })
    await settle()
    await press('ArrowDown')
    const moved = lit()?.textContent

    await palette.setProps({ bands: SECTIONS })
    await settle()

    expect(lit()?.textContent).toBe(moved)
  })
})

describe('what a band says about itself', () => {
  const bands = () => Array.from(document.body.querySelectorAll<HTMLElement>('[role="group"]'))

  it('draws a line under a band that is still filling, and marks the band busy', async () => {
    mountPalette({
      bands: [{ id: 'meaning', title: 'Meaning', items: [], working: true }],
    })
    await settle()

    expect(document.body.querySelector('.waiting')).not.toBeNull()
    expect(bands()[0]?.getAttribute('aria-busy')).toBe('true')
  })

  it('draws no line and marks nothing busy where nothing is filling', async () => {
    mountPalette()
    await settle()

    expect(document.body.querySelector('.waiting')).toBeNull()
    expect(bands().some((band) => band.hasAttribute('aria-busy'))).toBe(false)
  })

  it('leaves the line out of what a screen reader reads, since the band says it', async () => {
    mountPalette({
      bands: [{ id: 'meaning', title: 'Meaning', items: [], working: true }],
    })
    await settle()

    expect(document.body.querySelector('.waiting')?.getAttribute('aria-hidden')).toBe(
      'true',
    )
  })

  it("says a band came back with nothing, in the band's own words", async () => {
    mountPalette({
      bands: [{ id: 'meaning', title: 'Meaning', items: [], silence: 'No model is set' }],
    })
    await settle()
    expect(document.body.querySelector('[data-palette="silence"]')?.textContent).toContain(
      'No model is set',
    )
  })

  it('draws a band that answered with nothing nowhere', async () => {
    mountPalette({
      bands: [SECTIONS[0]!, { id: 'meaning', title: 'Meaning', items: [] }],
    })
    await settle()

    expect(bands()).toHaveLength(1)
    expect(document.body.querySelector('[data-palette="silence"]')).toBeNull()
  })

  it('draws a band holding nothing while it is still working', async () => {
    mountPalette({
      bands: [{ id: 'meaning', title: 'Meaning', items: [], working: true }],
    })
    await settle()

    expect(bands()).toHaveLength(1)
    expect(document.body.querySelector('[data-palette="silence"]')).toBeNull()
  })
})

describe('marking why an item is here', () => {
  it('marks the run that matched, and marks nothing where nothing did', async () => {
    mountPalette()
    await settle()

    const marked = options()[0]?.querySelectorAll('[data-hit]')
    expect(Array.from(marked ?? []).map((run) => run.textContent)).toEqual(['Ent'])
    expect(options()[1]?.querySelectorAll('[data-hit]')).toHaveLength(0)
  })

  it('draws the second line for an item that carries one, and none for one that does not', async () => {
    mountPalette()
    await settle()

    expect(options()[3]?.querySelector('[data-palette="detail"]')?.textContent).toBe(
      'a reversible engine',
    )
    expect(options()[0]?.querySelector('[data-palette="detail"]')).toBeNull()
  })
})

describe('which band stands where', () => {
  const bands = () =>
    Array.from(document.body.querySelectorAll<HTMLElement>('[data-palette="title"]')).map((band) =>
      band.textContent?.trim(),
    )

  it('draws a band that came back with nothing at the foot', async () => {
    mountPalette({
      bands: [
        { id: 'names', title: 'Names', items: [], silence: 'Nothing' },
        SECTIONS[1]!,
      ],
    })
    await settle()

    expect(bands()).toEqual(['Text', 'Names'])
  })

  it('lights the first item there is, wherever its band was offered', async () => {
    mountPalette({
      bands: [
        { id: 'names', title: 'Names', items: [], silence: 'Nothing' },
        SECTIONS[1]!,
      ],
    })
    await settle()

    expect(lit()?.textContent).toContain('Heat engine')
  })

  it('brings a band back up the moment it holds something', async () => {
    const palette = mountPalette({
      bands: [{ id: 'names', title: 'Names', items: [], silence: 'Nothing' }, SECTIONS[1]!],
    })
    await settle()
    expect(bands()).toEqual(['Text', 'Names'])

    await palette.setProps({ bands: SECTIONS })
    await settle()
    expect(bands()).toEqual(['Names', 'Text'])
  })
})

describe('what the palette says about itself', () => {
  it('says the list is open only while there is one', async () => {
    const palette = mountPalette()
    await settle()
    expect(field()?.getAttribute('aria-expanded')).toBe('true')

    await palette.setProps({ bands: [] })
    await settle()
    expect(field()?.getAttribute('aria-expanded')).toBe('false')
  })

  it('gives the list bands it owns, each named by its own title', async () => {
    mountPalette()
    await settle()

    const list = document.body.querySelector('[role="listbox"]')!
    const groups = Array.from(list.querySelectorAll('[role="group"]'))
    expect(groups).toHaveLength(2)

    for (const group of groups) {
      const named = document.getElementById(group.getAttribute('aria-labelledby') ?? '')
      expect(named?.textContent).toContain(group === groups[0] ? 'Names' : 'Text')
    }
  })

  it('passes a pointer over an item that cannot be chosen', async () => {
    mountPalette()
    await settle()

    const off = options()[2]!
    expect(off.textContent).toContain('Not this one')

    off.dispatchEvent(new PointerEvent('pointermove', { bubbles: true, clientX: 40, clientY: 40 }))
    await nextTick()

    expect(off.getAttribute('data-here')).toBeNull()
    expect(lit()?.textContent).toContain('Entropy')
  })
})

describe('an item offering more than two actions', () => {
  it('says at the foot only what a key reaches, and where the rest are', async () => {
    mountPalette({ bands: OFFERING })
    await settle()

    expect(keys().map(said)).toEqual(['Return Show in plex', 'Shift Return Open the note'])
    expect(document.body.querySelector('[data-palette="more"]')?.textContent).toContain('Actions')
  })

  it('reaches the first two by key, and nothing past them', async () => {
    const palette = mountPalette({ bands: OFFERING })
    await settle()

    await press('Enter')
    expect(palette.emitted('choose')?.at(-1)).toEqual(['entropy', 'travel'])

    await press('Enter', { shiftKey: true })
    expect(palette.emitted('choose')?.at(-1)).toEqual(['entropy', 'open'])
    expect(palette.emitted('choose')).toHaveLength(2)
  })

  it('writes the keystroke that reaches an item away from the palette', async () => {
    mountPalette({ bands: OFFERING })
    await settle()

    const hint = options()[0]?.querySelector('[data-palette="hint"]')
    expect(marksOf(hint)).toEqual(['option'])
    expect(spoken(hint)).toBe('Option 1')
    expect(options()[1]?.querySelector('[data-palette="hint"]')).toBeNull()
  })
})

describe('the action panel', () => {
  const open = async () => {
    const palette = mountPalette({ bands: OFFERING })
    await settle()
    await press('k', { ctrlKey: true })
    await settle()
    return palette
  }

  it('opens on the chord and lists everything the lit item offers', async () => {
    await open()

    expect(deeds().map((deed) => said(deed.querySelector('[data-actions="name"]')))).toEqual([
      'Show in plex',
      'Open the note',
      'Open beside',
      'Rename',
      'Move to trash',
    ])
    expect(deeds().map((deed) => spoken(deed.querySelector('[data-actions="hint"]')))).toEqual([
      'Return',
      'Shift Return',
      '',
      '',
      '',
    ])
  })

  it('opens on the same chord held with the other key', async () => {
    mountPalette({ bands: OFFERING })
    await settle()

    await press('k', { metaKey: true })
    await settle()
    expect(sheet()).not.toBeNull()
  })

  it('opens nothing for a list with nothing to be chosen in it', async () => {
    mountPalette({ bands: [{ id: 'names', title: 'Names', items: [] }] })
    await settle()

    await press('k', { ctrlKey: true })
    await settle()
    expect(sheet()).toBeNull()
  })

  it('takes the keyboard into a field of its own', async () => {
    await open()
    expect(document.activeElement).toBe(hunt())
  })

  it('is a second list, and takes the active one from the list underneath', async () => {
    await open()

    expect(field()?.getAttribute('aria-activedescendant')).toBeNull()
    expect(hunt()?.getAttribute('aria-activedescendant')).toBe(deeds()[0]?.id)
    expect(hunt()?.getAttribute('aria-controls')).toBe(
      document.body.querySelector('[data-actions="list"]')?.id,
    )
    expect(deeds()[0]?.getAttribute('aria-selected')).toBe('true')
    expect(deeds()[1]?.getAttribute('aria-selected')).toBe('false')
  })

  it('walks its own list, and leaves the list underneath where it was', async () => {
    await open()

    await pressOn(hunt(), 'ArrowDown')
    expect(litDeed()?.textContent).toContain('Open the note')
    expect(lit()?.textContent).toContain('Entropy')

    await pressOn(hunt(), 'ArrowUp')
    await pressOn(hunt(), 'ArrowUp')
    expect(litDeed()?.textContent).toContain('Move to trash')
  })

  it('keeps to the actions the words in its field name', async () => {
    await open()

    await typeIn(hunt(), 'open')
    expect(deeds().map((deed) => said(deed.querySelector('[data-actions="name"]')))).toEqual([
      'Open the note',
      'Open beside',
    ])
    expect(litDeed()?.textContent).toContain('Open the note')

    await typeIn(hunt(), 'nowhere')
    expect(deeds()).toHaveLength(0)
    expect(document.body.querySelector('[data-actions="silence"]')).not.toBeNull()
  })

  it('runs the action it is on and puts itself away', async () => {
    const palette = await open()

    await typeIn(hunt(), 'trash')
    await pressOn(hunt(), 'Enter')
    await settle()

    expect(palette.emitted('choose')?.at(-1)).toEqual(['entropy', 'remove'])
    expect(sheet()).toBeNull()
    expect(document.activeElement).toBe(field())
  })

  it('lights the action a pointer that has moved is over', async () => {
    await open()

    deeds()[2]?.dispatchEvent(
      new PointerEvent('pointermove', { bubbles: true, clientX: 80, clientY: 80 }),
    )
    await nextTick()

    expect(litDeed()?.textContent).toContain('Open beside')
  })

  it('runs the action a press lands on', async () => {
    const palette = await open()

    deeds()[3]?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await settle()

    expect(palette.emitted('choose')?.at(-1)).toEqual(['entropy', 'rename'])
    expect(sheet()).toBeNull()
  })

  it('closes on Escape and leaves the palette open', async () => {
    const palette = await open()

    await pressOn(hunt(), 'Escape')
    await settle()

    expect(palette.emitted('dismiss')).toBeUndefined()
    expect(drawn()).not.toBeNull()
    expect(sheet()).toBeNull()
    expect(document.activeElement).toBe(field())
  })

  it('closes on the chord that opened it', async () => {
    await open()

    await pressOn(hunt(), 'k', { ctrlKey: true })
    await settle()
    expect(sheet()).toBeNull()
  })

  it('closes on a press anywhere but on itself', async () => {
    await open()

    document.body
      .querySelector('[data-palette="list"]')
      ?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    await settle()

    expect(sheet()).toBeNull()
  })

  it('keeps the item it is about while a pointer crosses the list', async () => {
    await open()

    options()[1]?.dispatchEvent(
      new PointerEvent('pointermove', { bubbles: true, clientX: 40, clientY: 40 }),
    )
    await nextTick()

    expect(lit()?.textContent).toContain('Entropy')
    expect(deeds()).toHaveLength(5)
  })

  it('goes when the item it is about stops offering anything', async () => {
    const palette = await open()

    await palette.setProps({ bands: [{ id: 'names', title: 'Names', items: [] }] })
    await settle()

    expect(sheet()).toBeNull()
  })

  it('goes when the palette closes, and does not come back with it', async () => {
    const palette = await open()

    await palette.setProps({ open: false })
    await settle()
    await palette.setProps({ open: true })
    await settle()

    expect(sheet()).toBeNull()
  })

  it('stays on the action it is on when the list is offered again', async () => {
    const palette = await open()
    await pressOn(hunt(), 'ArrowDown')
    await pressOn(hunt(), 'ArrowDown')
    expect(litDeed()?.textContent).toContain('Open beside')

    // The same bands in arrays of their own, as a caller building them from
    // what the window holds hands over.
    await palette.setProps({
      bands: [
        {
          id: 'names',
          title: 'Names',
          items: [
            {
              id: 'entropy',
              title: 'Entropy',
              actions: MANY.map((one) => ({ ...one })),
              keys: OPTION_1,
            },
            { id: 'enthalpy', title: 'Enthalpy', actions: OPEN },
          ],
        },
      ],
    })
    await settle()

    expect(litDeed()?.textContent).toContain('Open beside')
  })

  it('stands on the first action left when the one it was on is gone', async () => {
    const palette = await open()
    await pressOn(hunt(), 'ArrowDown')
    expect(litDeed()?.textContent).toContain('Open the note')

    await palette.setProps({
      bands: [
        {
          id: 'names',
          title: 'Names',
          items: [{ id: 'entropy', title: 'Entropy', actions: [{ id: 'travel', text: 'Show in plex' }] }],
        },
      ],
    })
    await settle()

    expect(litDeed()?.textContent).toContain('Show in plex')
  })

  it('goes on a press on the ground, and the palette stays where it is', async () => {
    const palette = await open()

    drawn()?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    await settle()

    expect(sheet()).toBeNull()
    expect(drawn()).not.toBeNull()
    expect(palette.emitted('dismiss')).toBeUndefined()
  })

  it('keeps the keyboard in its own field on Tab', async () => {
    await open()

    const event = await pressOn(hunt(), 'Tab')
    expect(event.defaultPrevented).toBe(true)
  })
})

describe('the chord that opens the action panel', () => {
  it('is left to whoever else answers it when nothing is lit', async () => {
    mountPalette({ bands: [{ id: 'names', title: 'Names', items: [] }] })
    await settle()

    const event = await press('k', { ctrlKey: true })

    expect(sheet()).toBeNull()
    expect(event.defaultPrevented).toBe(false)
  })

  it('is taken when there is something lit to act on', async () => {
    mountPalette({ bands: OFFERING })
    await settle()

    const event = await press('k', { ctrlKey: true })

    expect(sheet()).not.toBeNull()
    expect(event.defaultPrevented).toBe(true)
  })
})

describe('the keyboard while the palette stands', () => {
  it('stays in the field on Tab', async () => {
    mountPalette({ bands: OFFERING })
    await settle()

    const event = await press('Tab')

    expect(event.defaultPrevented).toBe(true)
  })
})

describe('what the action panel is called', () => {
  it('is said by whoever offers it', async () => {
    mountPalette({
      bands: OFFERING,
      actionWords: {
        name: 'Deeds',
        placeholder: 'Look for a deed',
        silence: 'No deed by that name',
      },
    })
    await settle()
    await press('k', { ctrlKey: true })
    await settle()

    expect(sheet()?.getAttribute('aria-label')).toBe('Deeds')
    expect(hunt()?.getAttribute('aria-label')).toBe('Look for a deed')
    expect(hunt()?.placeholder).toBe('Look for a deed')
    expect(document.body.querySelector('[data-palette="more"]')?.textContent).toContain('Deeds')

    await typeIn(hunt(), 'zzz')
    expect(document.body.querySelector('[data-actions="silence"]')?.textContent?.trim()).toBe(
      'No deed by that name',
    )
  })
})

describe('one step of several', () => {
  it('says which step the field is on, and says it describes the field', async () => {
    mountPalette({ bands: OFFERING, crumb: 'New name for «Entropy»' })
    await settle()

    const crumb = document.body.querySelector('[data-palette="crumb"]')
    expect(crumb?.textContent).toBe('New name for «Entropy»')
    expect(field()?.getAttribute('aria-describedby')).toBe(crumb?.id)
  })

  it('draws no chip and describes the field with nothing without one', async () => {
    mountPalette({ bands: OFFERING })
    await settle()

    expect(document.body.querySelector('[data-palette="crumb"]')).toBeNull()
    expect(field()?.getAttribute('aria-describedby')).toBeNull()
  })

  it('takes the keyboard back and selects what stands there when the step changes', async () => {
    const palette = mountPalette({ bands: OFFERING, step: 'find' })
    await settle()

    await palette.setProps({ step: 'rename', modelValue: 'Entropy' })
    await settle()

    expect(document.activeElement).toBe(field())
    expect(field()?.value).toBe('Entropy')
    expect(field()?.selectionStart).toBe(0)
    expect(field()?.selectionEnd).toBe('Entropy'.length)
  })

  it('puts the action panel away when the step changes', async () => {
    const palette = mountPalette({ bands: OFFERING, step: 'find' })
    await settle()
    await press('k', { ctrlKey: true })
    await settle()

    await palette.setProps({ step: 'rename' })
    await settle()

    expect(sheet()).toBeNull()
  })

  it('asks for the step before on Backspace in an empty field', async () => {
    const palette = mountPalette({ bands: OFFERING })
    await settle()

    await press('Backspace')
    expect(palette.emitted('back')).toHaveLength(1)
  })

  it('asks for nothing on Backspace while something is typed', async () => {
    const palette = mountPalette({ bands: OFFERING, modelValue: 'ent' })
    await settle()

    await press('Backspace')
    expect(palette.emitted('back')).toBeUndefined()
  })

  it('asks for nothing on Backspace in the action panel', async () => {
    const palette = mountPalette({ bands: OFFERING })
    await settle()
    await press('k', { ctrlKey: true })
    await settle()

    await pressOn(hunt(), 'Backspace')
    expect(palette.emitted('back')).toBeUndefined()
  })
})
