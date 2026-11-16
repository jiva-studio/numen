/**
 * What the palette does when a step asks for several items from a list: the field and the title
 * the acceptance driver reads, the check marks, the keys that toggle a row and keep the list open,
 * the row that creates an item, and Escape. It is drawn at the end of the document, so everything
 * here is read off the document.
 */
import { mount } from '@vue/test-utils'
import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import Palette from './Palette.vue'

beforeAll(() => {
  Element.prototype.scrollIntoView = () => undefined
})

const TREES = [
  { id: 'birch', title: 'Birch', state: 'unset' },
  { id: 'willow', title: 'Willow', state: 'unset' },
  { id: 'cedar', title: 'Cedar', state: 'mixed' },
  { id: 'fir', title: 'Fir', state: 'set' },
] as const

const MULTI = {
  title: 'Trees',
  placeholder: 'Tree',
  items: TREES,
  recent: ['willow'],
  getCreateTitle: (text: string) => `Create ${text}`,
}

type PaletteProps = InstanceType<typeof Palette>['$props']

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
const title = () => document.body.querySelector<HTMLElement>('[data-palette="title"]')
const options = () => Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]'))
const getNames = () => options().map((one) => (one.textContent ?? '').replace(/\s+/g, ' ').trim())

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

afterEach(() => {
  mounted?.unmount()
  mounted = null
})

describe('a step that asks for several items', () => {
  it('asks in the field with the placeholder it was given, under the title it was given', async () => {
    mountPalette()
    await settle()
    expect(field()?.placeholder).toBe('Tree')
    expect(title()?.textContent?.trim()).toBe('Trees')
  })

  it('lists the set and the mixed items first, the recent next, and the rest by title', async () => {
    mountPalette()
    await settle()
    expect(getNames()).toStrictEqual(['Cedar', 'Fir', 'Willow', 'Birch'])
  })

  it('marks each row checked, mixed or not checked', async () => {
    mountPalette()
    await settle()
    expect(options().map((one) => one.getAttribute('aria-checked'))).toStrictEqual([
      'mixed',
      'true',
      'false',
      'false',
    ])
  })

  it('toggles the highlighted row on Enter, keeps the list open and does not dismiss', async () => {
    const palette = mountPalette()
    await settle()
    await press('Enter')
    expect(palette.emitted('toggle')).toStrictEqual([['cedar']])
    expect(palette.emitted('dismiss')).toBeUndefined()
    expect(palette.emitted('choose')).toBeUndefined()
    expect(options()).toHaveLength(4)
  })

  it('toggles the highlighted row on the action key and does not open the action panel', async () => {
    const palette = mountPalette()
    await settle()
    await press('ArrowDown')
    await press('k', { ctrlKey: true })
    expect(palette.emitted('toggle')).toStrictEqual([['fir']])
    expect(document.body.querySelector('[data-actions="panel"]')).toBeNull()
    expect(palette.emitted('dismiss')).toBeUndefined()
  })

  it('shows the state the caller gives after a toggle', async () => {
    const palette = mountPalette()
    await settle()
    await press('Enter')
    await palette.setProps({
      multi: {
        ...MULTI,
        items: TREES.map((one) => (one.id === 'cedar' ? { ...one, state: 'set' } : one)),
      },
    })
    expect(options()[0]?.getAttribute('aria-checked')).toBe('true')
  })

  it('makes a row that creates an item from the typed text, named as the caller names it', async () => {
    const palette = mountPalette()
    await settle()
    await typeIn('Harbour')
    expect(getNames()).toStrictEqual(['Create Harbour'])

    await press('Enter')
    expect(palette.emitted('create')).toStrictEqual([['Harbour']])
    expect(palette.emitted('dismiss')).toBeUndefined()
  })

  it('makes no create row for text that names an item in another case', async () => {
    mountPalette()
    await settle()
    await typeIn('FIR ')
    expect(getNames()).toStrictEqual(['Fir'])
  })

  it('asks to be dismissed on Escape', async () => {
    const palette = mountPalette()
    await settle()
    await press('Escape')
    expect(palette.emitted('dismiss')).toHaveLength(1)
    expect(palette.emitted('toggle')).toBeUndefined()
  })
})
