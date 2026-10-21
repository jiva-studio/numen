/**
 * The strip a tile or a block is dragged by: what a gesture on it takes hold
 * of, and what a key struck on its handle does.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import CardHeader from './CardHeader.vue'

const mountHeader = (slots: Record<string, string> = {}) =>
  mount(CardHeader, { props: { drag: 'Drag this block' }, attachTo: document.body, slots })

let rendered: ReturnType<typeof mountHeader> | null = null

afterEach(() => {
  rendered?.unmount()
  rendered = null
})

const header = (strip: ReturnType<typeof mountHeader>) => strip.get('.card-header')
const grip = (strip: ReturnType<typeof mountHeader>) => strip.get('.card-header__grip')

/** A press that lands on something inside the strip and rises to it. */
const pressOn = async (strip: ReturnType<typeof mountHeader>, selector: string) => {
  await strip.get(selector).trigger('pointerdown')
}

describe('the strip itself', () => {
  it('says to a pointer what taking hold of it does', () => {
    rendered = mountHeader()
    expect(header(rendered).attributes('title')).toBe('Drag this block')
  })

  // A strip holds a name box and buttons, so it is a container and no control,
  // and the keyboard is owed a control wherever it stops.
  it('is no stop on the way round the screen, and is drawn as a header', () => {
    rendered = mountHeader()
    expect(header(rendered).attributes('tabindex')).toBeUndefined()
    expect(header(rendered).attributes('role')).toBeUndefined()
    expect(header(rendered).element.tagName).toBe('HEADER')
  })

  it('passes on the taking hold of it and the letting go', async () => {
    rendered = mountHeader()
    await header(rendered).trigger('dragstart')
    await header(rendered).trigger('dragend')
    expect(rendered.emitted('dragstart')).toHaveLength(1)
    expect(rendered.emitted('dragend')).toHaveLength(1)
  })
})

describe('a press on something the strip holds', () => {
  const INSIDE = '<input class="inside" /><span class="plain">said</span>'

  it('lets a box to type in have the press, and not carrying the strip', async () => {
    rendered = mountHeader({ default: INSIDE })
    expect(header(rendered).attributes('draggable')).toBe('true')

    await pressOn(rendered, '.inside')
    expect(header(rendered).attributes('draggable')).toBe('false')
  })

  it('leaves the strip to be dragged where the press landed on nothing worked', async () => {
    rendered = mountHeader({ default: INSIDE })
    await pressOn(rendered, '.plain')
    expect(header(rendered).attributes('draggable')).toBe('true')
  })

  // A press that began in a box may travel off the strip and be let go
  // anywhere, so the end of it is heard wherever it happens.
  it('may be dragged again once the press is release, wherever that was', async () => {
    rendered = mountHeader({ default: INSIDE })
    await pressOn(rendered, '.inside')

    window.dispatchEvent(new Event('pointerup'))
    await rendered.vm.$nextTick()
    expect(header(rendered).attributes('draggable')).toBe('true')
  })

  it('may be dragged again where the press was cancelled', async () => {
    rendered = mountHeader({ default: INSIDE })
    await pressOn(rendered, '.inside')

    window.dispatchEvent(new Event('pointercancel'))
    await rendered.vm.$nextTick()
    expect(header(rendered).attributes('draggable')).toBe('true')
  })
})

describe('the handle at the start of the strip', () => {
  it('is a control the keyboard stops on, and says what working it does', () => {
    rendered = mountHeader()
    expect(grip(rendered).attributes('role')).toBe('button')
    expect(grip(rendered).attributes('tabindex')).toBe('0')
    expect(grip(rendered).attributes('aria-label')).toBe('Drag this block')
    expect(grip(rendered).attributes('aria-keyshortcuts')).toBe('ArrowUp ArrowDown')
  })

  it('carries what it heads one place along the order', async () => {
    rendered = mountHeader()
    await grip(rendered).trigger('keydown', { key: 'ArrowUp' })
    await grip(rendered).trigger('keydown', { key: 'ArrowDown' })
    expect((rendered.emitted('step') ?? []).map((turn) => (turn as unknown[])[0])).toEqual([
      'up',
      'down',
    ])
  })

  it('does nothing for a key that is no way along the order', async () => {
    rendered = mountHeader()
    await grip(rendered).trigger('keydown', { key: 'ArrowLeft' })
    await grip(rendered).trigger('keydown', { key: 'Enter' })
    expect(rendered.emitted('step')).toBeUndefined()
  })

  // A key struck in a box the strip holds is that box's, whatever it is.
  it('leaves a key struck in something the strip holds to that thing', async () => {
    rendered = mountHeader({ default: '<input class="inside" />' })
    await rendered.get('.inside').trigger('keydown', { key: 'ArrowUp' })
    expect(rendered.emitted('step')).toBeUndefined()
  })

  // The pointer has the whole strip, so a press on the handle drags the strip.
  it('leaves the strip to be dragged by a press that lands on it', async () => {
    rendered = mountHeader()
    await pressOn(rendered, '.card-header__grip')
    expect(header(rendered).attributes('draggable')).toBe('true')
  })
})

describe('what the strip is drawn with', () => {
  it('puts what it is given at its start and what it is pressed for at its end', () => {
    rendered = mountHeader({
      default: '<span class="held">Title</span>',
      actions: '<button>Remove</button>',
    })
    expect(rendered.get('.card-header__held .held').text()).toBe('Title')
    expect(rendered.get('.card-header__actions button').text()).toBe('Remove')
  })
})
