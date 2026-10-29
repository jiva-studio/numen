/**
 * The strip a tile or a block is dragged by: what a gesture on it takes hold
 * of, and what a key struck on its handle does.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import CardHeader from './CardHeader.vue'

const mountHeader = (slots: Record<string, string> = {}) =>
  mount(CardHeader, { props: { drag: 'Drag this block' }, attachTo: document.body, slots })

let mountedHeader: ReturnType<typeof mountHeader> | null = null

afterEach(() => {
  mountedHeader?.unmount()
  mountedHeader = null
})

const header = (strip: ReturnType<typeof mountHeader>) => strip.get('.card-header')
const grip = (strip: ReturnType<typeof mountHeader>) => strip.get('.card-header__grip')

/** A press that lands on something inside the strip and rises to it. */
const pressOn = async (strip: ReturnType<typeof mountHeader>, selector: string) => {
  await strip.get(selector).trigger('pointerdown')
}

describe('the strip itself', () => {
  it('says to a pointer what taking hold of it does', () => {
    mountedHeader = mountHeader()
    expect(header(mountedHeader).attributes('title')).toBe('Drag this block')
  })

  // The header holds a name input and buttons, so it is a container and not a
  // control, and keyboard focus goes to the controls inside it.
  it('is no stop on the way round the screen, and is drawn as a header', () => {
    mountedHeader = mountHeader()
    expect(header(mountedHeader).attributes('tabindex')).toBeUndefined()
    expect(header(mountedHeader).attributes('role')).toBeUndefined()
    expect(header(mountedHeader).element.tagName).toBe('HEADER')
  })

  it('passes on the taking hold of it and the letting go', async () => {
    mountedHeader = mountHeader()
    await header(mountedHeader).trigger('dragstart')
    await header(mountedHeader).trigger('dragend')
    expect(mountedHeader.emitted('dragstart')).toHaveLength(1)
    expect(mountedHeader.emitted('dragend')).toHaveLength(1)
  })
})

describe('a press on something the strip holds', () => {
  const INSIDE = '<input class="inside" /><span class="plain">said</span>'

  it('lets a box to type in have the press, and not carrying the strip', async () => {
    mountedHeader = mountHeader({ default: INSIDE })
    expect(header(mountedHeader).attributes('draggable')).toBe('true')

    await pressOn(mountedHeader, '.inside')
    expect(header(mountedHeader).attributes('draggable')).toBe('false')
  })

  it('leaves the strip to be dragged where the press landed on nothing worked', async () => {
    mountedHeader = mountHeader({ default: INSIDE })
    await pressOn(mountedHeader, '.plain')
    expect(header(mountedHeader).attributes('draggable')).toBe('true')
  })

  // A press that began in a box may travel off the strip and be let go
  // anywhere, so the end of it is heard wherever it happens.
  it('may be dragged again once the press is release, wherever that was', async () => {
    mountedHeader = mountHeader({ default: INSIDE })
    await pressOn(mountedHeader, '.inside')

    window.dispatchEvent(new Event('pointerup'))
    await mountedHeader.vm.$nextTick()
    expect(header(mountedHeader).attributes('draggable')).toBe('true')
  })

  it('may be dragged again where the press was cancelled', async () => {
    mountedHeader = mountHeader({ default: INSIDE })
    await pressOn(mountedHeader, '.inside')

    window.dispatchEvent(new Event('pointercancel'))
    await mountedHeader.vm.$nextTick()
    expect(header(mountedHeader).attributes('draggable')).toBe('true')
  })
})

describe('the handle at the start of the strip', () => {
  it('is a control the keyboard stops on, and says what working it does', () => {
    mountedHeader = mountHeader()
    expect(grip(mountedHeader).attributes('role')).toBe('button')
    expect(grip(mountedHeader).attributes('tabindex')).toBe('0')
    expect(grip(mountedHeader).attributes('aria-label')).toBe('Drag this block')
    expect(grip(mountedHeader).attributes('aria-keyshortcuts')).toBe('ArrowUp ArrowDown')
  })

  it('carries what it heads one place along the order', async () => {
    mountedHeader = mountHeader()
    await grip(mountedHeader).trigger('keydown', { key: 'ArrowUp' })
    await grip(mountedHeader).trigger('keydown', { key: 'ArrowDown' })
    expect((mountedHeader.emitted('step') ?? []).map((turn) => (turn as unknown[])[0])).toEqual([
      'up',
      'down',
    ])
  })

  it('does nothing for a key that is no way along the order', async () => {
    mountedHeader = mountHeader()
    await grip(mountedHeader).trigger('keydown', { key: 'ArrowLeft' })
    await grip(mountedHeader).trigger('keydown', { key: 'Enter' })
    expect(mountedHeader.emitted('step')).toBeUndefined()
  })

  // A key struck in a box the strip holds is that box's, whatever it is.
  it('leaves a key struck in something the strip holds to that thing', async () => {
    mountedHeader = mountHeader({ default: '<input class="inside" />' })
    await mountedHeader.get('.inside').trigger('keydown', { key: 'ArrowUp' })
    expect(mountedHeader.emitted('step')).toBeUndefined()
  })

  // The pointer has the whole strip, so a press on the handle drags the strip.
  it('leaves the strip to be dragged by a press that lands on it', async () => {
    mountedHeader = mountHeader()
    await pressOn(mountedHeader, '.card-header__grip')
    expect(header(mountedHeader).attributes('draggable')).toBe('true')
  })
})

describe('what the strip is drawn with', () => {
  it('puts what it is given at its start and what it is pressed for at its end', () => {
    mountedHeader = mountHeader({
      default: '<span class="content">Title</span>',
      actions: '<button>Remove</button>',
    })
    expect(mountedHeader.get('.card-header__content .content').text()).toBe('Title')
    expect(mountedHeader.get('.card-header__actions button').text()).toBe('Remove')
  })
})
