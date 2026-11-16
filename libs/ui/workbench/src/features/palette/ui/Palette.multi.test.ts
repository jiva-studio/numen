/**
 * What the palette does when a step asks for several items from a list. It is the palette as it is
 * for a command: the same panel, field, group, rows with an icon before the name, and key hints. The
 * icon is a dot in the colour of the item, a row that is on has a check mark at its end, and the keys
 * toggle the highlighted row and keep the palette open. It is drawn at the end of the document, so
 * everything here is read off the document.
 */
import { mount } from '@vue/test-utils'
import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import Palette from './Palette.vue'

beforeAll(() => {
  Element.prototype.scrollIntoView = () => undefined
})

const TREES = [
  { id: 'birch', title: 'Birch', state: 'unset', color: 'var(--numen-tag-1)' },
  { id: 'willow', title: 'Willow', state: 'unset', color: 'var(--numen-tag-2)' },
  { id: 'cedar', title: 'Cedar', state: 'mixed', color: 'var(--numen-tag-3)' },
  { id: 'alder', title: 'Alder', state: 'unset', color: 'var(--numen-tag-4)' },
  { id: 'maple', title: 'Maple', state: 'unset', color: 'var(--numen-tag-5)' },
  { id: 'fir', title: 'Fir', state: 'set', color: 'var(--numen-tag-6)' },
] as const

const MULTI = {
  title: 'Trees',
  placeholder: 'Tree',
  items: TREES,
  recent: ['fir', 'willow', 'maple'],
  getCreateTitle: (text: string) => `Create ${text}`,
  getCreateColor: (text: string) => `var(--numen-tag-${text.length})`,
}

type PaletteProps = InstanceType<typeof Palette>['$props']
type Tree = { id: string; title: string; state: 'set' | 'mixed' | 'unset'; color: string }

let mounted: { unmount: () => void } | null = null

const mountPalette = (props: Record<string, unknown> = {}) => {
  const palette = mount(Palette, {
    props: { isOpen: true, multi: MULTI, ...props } as PaletteProps,
    attachTo: document.body,
  })
  mounted = palette
  return palette
}

const settle = async () => {
  await nextTick()
  await nextTick()
}

const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')
const options = () => Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]'))
const getNames = () => options().map((one) => (one.textContent ?? '').replace(/\s+/g, ' ').trim())
const getRow = (name: string) => options()[getNames().indexOf(name)]
const getActiveName = () => getNames()[options().findIndex((one) => one.hasAttribute('data-here'))]

const press = async (key: string, more: KeyboardEventInit = {}) => {
  const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...more })
  field()?.dispatchEvent(event)
  await nextTick()
  return event
}

const typeIn = async (text: string) => {
  const input = field()
  if (!input) return
  input.value = text
  input.dispatchEvent(new Event('input'))
  await nextTick()
}

const withStates = (states: Record<string, Tree['state']>): readonly Tree[] =>
  TREES.map((one) => ({ ...one, state: states[one.id] ?? one.state }))

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

describe('the step is the palette', () => {
  it('asks in the field with the placeholder it was given, under the title it was given', async () => {
    mountPalette()
    await settle()
    expect(field()?.placeholder).toBe('Tree')
    const titles = Array.from(
      document.body.querySelectorAll('[data-palette="list"] [role="group"] [data-palette="title"]'),
    ).map((one) => one.textContent?.trim())
    expect(titles).toStrictEqual(['Trees'])
  })

  it('draws one panel with the field, the list and the key hints of the palette', async () => {
    mountPalette()
    await settle()
    const panels = document.body.querySelectorAll('[data-palette="panel"]')
    expect(panels).toHaveLength(1)
    expect(panels[0]?.querySelector('[data-palette="field"]')).not.toBeNull()
    expect(panels[0]?.querySelector('[data-palette="list"][role="listbox"]')).not.toBeNull()
    expect(panels[0]?.querySelector('[data-palette="more"]')).not.toBeNull()
  })

  it('marks the list as one where several rows can be checked', async () => {
    mountPalette()
    await settle()
    const list = document.body.querySelector('[data-palette="list"]')
    expect(list?.getAttribute('aria-multiselectable')).toBe('true')
  })

  it('draws every row with the name and the icon part of a row of the palette', async () => {
    mountPalette()
    await settle()
    expect(options()).toHaveLength(6)
    for (const row of options()) {
      expect(row.querySelector('[data-palette="name"]')).not.toBeNull()
      expect(row.querySelector('[data-palette="icon"]')).not.toBeNull()
    }
  })

  it('draws no check box', async () => {
    mountPalette()
    await settle()
    expect(document.body.querySelector('input[type="checkbox"]')).toBeNull()
    expect(document.body.querySelector('[role="checkbox"]')).toBeNull()
  })

  it('ties the field to the list and to the highlighted row', async () => {
    mountPalette()
    await settle()
    const list = document.body.querySelector('[data-palette="list"]')
    expect(field()?.getAttribute('aria-controls')).toBe(list?.id)
    expect(field()?.getAttribute('aria-activedescendant')).toBe(options()[0]?.id)
  })
})

describe('the icon of a row', () => {
  it('is a dot in the colour of the item', async () => {
    mountPalette()
    await settle()
    const icons = options().map((one) => one.querySelector('[data-palette="icon"]')?.outerHTML)
    for (const tree of TREES) {
      expect(icons[getNames().indexOf(tree.title)]).toContain(tree.color)
    }
  })

  it('keeps the place of the icon for an item that has no colour', async () => {
    mountPalette({ multi: { ...MULTI, items: [{ id: 'oak', title: 'Oak', state: 'unset' }] } })
    await settle()
    expect(getRow('Oak')?.querySelector('[data-palette="icon"]')).not.toBeNull()
  })

  it('follows the colour the caller gives an item', async () => {
    const palette = mountPalette()
    await settle()
    await palette.setProps({
      multi: {
        ...MULTI,
        items: TREES.map((one) =>
          one.id === 'fir' ? { ...one, color: 'var(--numen-tag-8)' } : one,
        ),
      },
    })
    expect(getRow('Fir')?.querySelector('[data-palette="icon"]')?.outerHTML).toContain(
      'var(--numen-tag-8)',
    )
  })
})

describe('the mark at the end of a row', () => {
  it('is a check for a set row and a dash for a mixed one, the last part of the row', async () => {
    mountPalette()
    await settle()
    const fir = getRow('Fir')?.lastElementChild
    const cedar = getRow('Cedar')?.lastElementChild
    expect(fir?.getAttribute('data-palette')).toBe('check')
    expect(fir?.getAttribute('data-state')).toBe('set')
    expect(cedar?.getAttribute('data-palette')).toBe('check')
    expect(cedar?.getAttribute('data-state')).toBe('mixed')
  })

  it('is not there for a row that is not set', async () => {
    mountPalette()
    await settle()
    for (const name of ['Birch', 'Willow', 'Alder', 'Maple']) {
      expect(getRow(name)?.querySelector('[data-palette="check"]')).toBeNull()
    }
  })

  it('marks each row checked, mixed or not checked for a screen reader', async () => {
    mountPalette()
    await settle()
    expect(options().map((one) => one.getAttribute('aria-checked'))).toStrictEqual([
      'mixed',
      'true',
      'false',
      'false',
      'false',
      'false',
    ])
  })

  it('follows the state the caller gives after a toggle', async () => {
    const palette = mountPalette()
    await settle()
    await press('Enter')
    await palette.setProps({ multi: { ...MULTI, items: withStates({ cedar: 'set' }) } })
    expect(getRow('Cedar')?.getAttribute('aria-checked')).toBe('true')
    expect(
      getRow('Cedar')?.querySelector('[data-palette="check"]')?.getAttribute('data-state'),
    ).toBe('set')
    await palette.setProps({ multi: { ...MULTI, items: withStates({ cedar: 'unset' }) } })
    expect(getRow('Cedar')?.querySelector('[data-palette="check"]')).toBeNull()
  })
})

describe('the order of the rows', () => {
  it('puts the set and the mixed items first, the recent ones next in the order given, and the rest by title', async () => {
    mountPalette()
    await settle()
    expect(getNames()).toStrictEqual(['Cedar', 'Fir', 'Willow', 'Maple', 'Alder', 'Birch'])
  })

  it('shows an item that is set and recent once, among the set ones', async () => {
    mountPalette({ multi: { ...MULTI, recent: ['fir'] } })
    await settle()
    expect(getNames().filter((name) => name === 'Fir')).toHaveLength(1)
    expect(getNames().indexOf('Fir')).toBeLessThan(getNames().indexOf('Alder'))
  })

  it('ignores a recent id that no item has', async () => {
    mountPalette({ multi: { ...MULTI, recent: ['gone', 'maple'] } })
    await settle()
    expect(getNames()).toStrictEqual(['Cedar', 'Fir', 'Maple', 'Alder', 'Birch', 'Willow'])
  })

  it('keeps that order among the items a typed word keeps', async () => {
    mountPalette()
    await settle()
    await typeIn('a')
    expect(getNames().filter((name) => !name.startsWith('Create'))).toStrictEqual([
      'Cedar',
      'Maple',
      'Alder',
    ])
  })

  it('moves an item up among the set ones when the caller sets it', async () => {
    const palette = mountPalette({
      multi: { ...MULTI, recent: [], items: withStates({ fir: 'unset', cedar: 'unset' }) },
    })
    await settle()
    await palette.setProps({
      multi: {
        ...MULTI,
        recent: [],
        items: withStates({ fir: 'unset', cedar: 'unset', maple: 'set' }),
      },
    })
    expect(getNames()[0]).toBe('Maple')
  })

  it('keeps the keyboard on the row it toggled when that row moves', async () => {
    const palette = mountPalette({
      multi: { ...MULTI, recent: [], items: withStates({ cedar: 'unset', fir: 'unset' }) },
    })
    await settle()
    await press('ArrowDown')
    await press('ArrowDown')
    expect(getActiveName()).toBe('Cedar')

    await press('Enter')
    await palette.setProps({
      multi: { ...MULTI, recent: [], items: withStates({ cedar: 'set', fir: 'unset' }) },
    })
    expect(getNames()[0]).toBe('Cedar')
    await press('ArrowDown')
    expect(getActiveName()).toBe('Alder')
  })
})

describe('the keys', () => {
  it('toggles the highlighted row on Enter, keeps the palette open and does not choose', async () => {
    const palette = mountPalette()
    await settle()
    await press('Enter')
    expect(palette.emitted('toggle')).toStrictEqual([['cedar']])
    expect(palette.emitted('dismiss')).toBeUndefined()
    expect(palette.emitted('choose')).toBeUndefined()
    expect(options()).toHaveLength(6)
  })

  it('toggles the row the arrows moved to', async () => {
    const palette = mountPalette()
    await settle()
    await press('ArrowDown')
    expect(getActiveName()).toBe('Fir')
    await press('Enter')
    expect(palette.emitted('toggle')).toStrictEqual([['fir']])
  })

  it('toggles the same row on each Enter and keeps the rows in place', async () => {
    const palette = mountPalette()
    await settle()
    await press('Enter')
    await press('Enter')
    expect(palette.emitted('toggle')).toStrictEqual([['cedar'], ['cedar']])
    expect(getNames()).toStrictEqual(['Cedar', 'Fir', 'Willow', 'Maple', 'Alder', 'Birch'])
  })

  it('toggles the highlighted row on the action key with Control or with Meta, and opens no action panel', async () => {
    const palette = mountPalette()
    await settle()
    await press('k', { ctrlKey: true })
    await press('ArrowDown')
    await press('k', { metaKey: true })
    expect(palette.emitted('toggle')).toStrictEqual([['cedar'], ['fir']])
    expect(document.body.querySelector('[data-actions="panel"]')).toBeNull()
    expect(palette.emitted('dismiss')).toBeUndefined()
  })

  it('does not toggle on the letter k alone', async () => {
    const palette = mountPalette()
    await settle()
    await press('k')
    expect(palette.emitted('toggle')).toBeUndefined()
  })

  it('toggles the row that is clicked, and keeps the palette open', async () => {
    const palette = mountPalette()
    await settle()
    getRow('Fir')?.click()
    await nextTick()
    expect(palette.emitted('toggle')).toStrictEqual([['fir']])
    expect(palette.emitted('dismiss')).toBeUndefined()
    expect(palette.emitted('choose')).toBeUndefined()
  })

  it('asks to be dismissed on Escape and toggles nothing', async () => {
    const palette = mountPalette()
    await settle()
    await press('Escape')
    expect(palette.emitted('dismiss')).toHaveLength(1)
    expect(palette.emitted('toggle')).toBeUndefined()
  })
})

describe('the row that creates an item', () => {
  it('is first when the typed text names no item, named as the caller names it', async () => {
    mountPalette()
    await settle()
    await typeIn('Harbour')
    expect(getNames()).toStrictEqual(['Create Harbour'])
  })

  it('has the icon in the colour the caller gives for the typed text', async () => {
    mountPalette()
    await settle()
    await typeIn('Harbour')
    expect(getRow('Create Harbour')?.querySelector('[data-palette="icon"]')?.outerHTML).toContain(
      'var(--numen-tag-7)',
    )
  })

  it('has no check mark at its end', async () => {
    mountPalette()
    await settle()
    await typeIn('Harbour')
    expect(getRow('Create Harbour')?.querySelector('[data-palette="check"]')).toBeNull()
  })

  it('is what Enter asks for, with the text as typed, and the palette stays open', async () => {
    const palette = mountPalette()
    await settle()
    await typeIn('Harbour')
    await press('Enter')
    expect(palette.emitted('create')).toStrictEqual([['Harbour']])
    expect(palette.emitted('toggle')).toBeUndefined()
    expect(palette.emitted('dismiss')).toBeUndefined()
  })

  it('asks for the text as typed on the action key', async () => {
    const palette = mountPalette()
    await settle()
    await typeIn('Harbour')
    await press('k', { ctrlKey: true })
    expect(palette.emitted('create')).toStrictEqual([['Harbour']])
  })

  it('asks when it is clicked', async () => {
    const palette = mountPalette()
    await settle()
    await typeIn('Harbour')
    getRow('Create Harbour')?.click()
    await nextTick()
    expect(palette.emitted('create')).toStrictEqual([['Harbour']])
  })

  it('is not there while nothing is typed, and every item is', async () => {
    mountPalette()
    await settle()
    expect(getNames()).toHaveLength(6)
    expect(getNames().some((name) => name.startsWith('Create'))).toBe(false)
  })

  it('is not there while only spaces are typed', async () => {
    mountPalette()
    await settle()
    await typeIn('   ')
    expect(getNames().some((name) => name.startsWith('Create'))).toBe(false)
  })

  it('is not there when an item has the typed text as its title, in any case and with spaces around', async () => {
    mountPalette()
    await settle()
    await typeIn('  cedar ')
    expect(getNames()).toStrictEqual(['Cedar'])
  })

  it('is there while the typed text only begins an item title', async () => {
    mountPalette()
    await settle()
    await typeIn('ced')
    expect(getNames()).toStrictEqual(['Create ced', 'Cedar'])
  })

  it('is ahead of the items found', async () => {
    mountPalette()
    await settle()
    await typeIn('a')
    expect(getNames()).toStrictEqual(['Create a', 'Cedar', 'Maple', 'Alder'])
  })
})
