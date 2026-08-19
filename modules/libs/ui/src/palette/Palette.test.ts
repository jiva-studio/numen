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
import type { PaletteBand } from './model'

const OPEN = [{ id: 'open', text: 'Open the note' }]
const BOTH = [
  { id: 'travel', text: 'Show in plex' },
  { id: 'open', text: 'Open the note' },
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

const drawn = () => document.body.querySelector<HTMLElement>('.palette')
const field = () => document.body.querySelector<HTMLInputElement>('.palette__field')
const options = () => Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]'))
const lit = () => document.body.querySelector<HTMLElement>('[data-here]')
const keys = () => Array.from(document.body.querySelectorAll<HTMLElement>('.palette__key'))

const press = async (key: string, more: KeyboardEventInit = {}) => {
  field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, ...more }))
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
      .querySelector('.palette__panel')
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
    expect(keys().map((key) => key.textContent?.trim())).toEqual([
      '↵ Show in plex',
      '⇧↵ Open the note',
    ])

    await press('ArrowDown')
    await press('ArrowDown')
    expect(keys().map((key) => key.textContent?.trim())).toEqual(['↵ Open the note'])
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
  const bands = () => Array.from(document.body.querySelectorAll<HTMLElement>('.palette__band'))

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
    expect(document.body.querySelector('.palette__silence')?.textContent).toContain(
      'No model is set',
    )
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

    expect(options()[3]?.querySelector('.palette__detail')?.textContent).toBe(
      'a reversible engine',
    )
    expect(options()[0]?.querySelector('.palette__detail')).toBeNull()
  })
})

describe('which band stands where', () => {
  const bands = () =>
    Array.from(document.body.querySelectorAll<HTMLElement>('.palette__title')).map((band) =>
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
      bands: [{ id: 'names', title: 'Names', items: [] }, SECTIONS[1]!],
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
