/**
 * The strip a tile or a block is dragged by: what a gesture on it takes hold
 * of, and what a key struck on its handle does.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import CardHeader from './CardHeader.vue'

const mountHeader = (slots: Record<string, string> = {}) =>
  mount(CardHeader, { props: { drag: 'Drag this block' }, attachTo: document.body, slots })

let drawn: ReturnType<typeof mountHeader> | null = null

afterEach(() => {
  drawn?.unmount()
  drawn = null
})

const header = (strip: ReturnType<typeof mountHeader>) => strip.get('.card-header')
const grip = (strip: ReturnType<typeof mountHeader>) => strip.get('.card-header__grip')

/** A press that lands on something inside the strip and rises to it. */
const pressOn = async (strip: ReturnType<typeof mountHeader>, selector: string) => {
  await strip.get(selector).trigger('pointerdown')
}

describe('the strip itself', () => {
  it('says to a pointer what taking hold of it does', () => {
    drawn = mountHeader()
    expect(header(drawn).attributes('title')).toBe('Drag this block')
  })

  // A strip holds a name box and buttons, so it is a container and no control,
  // and the keyboard is owed a control wherever it stops.
  it('is no stop on the way round the screen, and is drawn as a header', () => {
    drawn = mountHeader()
    expect(header(drawn).attributes('tabindex')).toBeUndefined()
    expect(header(drawn).attributes('role')).toBeUndefined()
    expect(header(drawn).element.tagName).toBe('HEADER')
  })

  it('hands on the taking hold of it and the letting go', async () => {
    drawn = mountHeader()
    await header(drawn).trigger('dragstart')
    await header(drawn).trigger('dragend')
    expect(drawn.emitted('dragstart')).toHaveLength(1)
    expect(drawn.emitted('dragend')).toHaveLength(1)
  })
})

describe('a press on something the strip holds', () => {
  const INSIDE = '<input class="inside" /><span class="plain">said</span>'

  it('lets a box to type in have the press, rather than carrying the strip', async () => {
    drawn = mountHeader({ default: INSIDE })
    expect(header(drawn).attributes('draggable')).toBe('true')

    await pressOn(drawn, '.inside')
    expect(header(drawn).attributes('draggable')).toBe('false')
  })

  it('leaves the strip to be dragged where the press landed on nothing worked', async () => {
    drawn = mountHeader({ default: INSIDE })
    await pressOn(drawn, '.plain')
    expect(header(drawn).attributes('draggable')).toBe('true')
  })

  // A press that began in a box may travel off the strip and be let go
  // anywhere, so the end of it is heard wherever it happens.
  it('may be dragged again once the press is let go of, wherever that was', async () => {
    drawn = mountHeader({ default: INSIDE })
    await pressOn(drawn, '.inside')

    window.dispatchEvent(new Event('pointerup'))
    await drawn.vm.$nextTick()
    expect(header(drawn).attributes('draggable')).toBe('true')
  })

  it('may be dragged again where the press was cancelled', async () => {
    drawn = mountHeader({ default: INSIDE })
    await pressOn(drawn, '.inside')

    window.dispatchEvent(new Event('pointercancel'))
    await drawn.vm.$nextTick()
    expect(header(drawn).attributes('draggable')).toBe('true')
  })
})

describe('the handle at the start of the strip', () => {
  it('is a control the keyboard stops on, and says what working it does', () => {
    drawn = mountHeader()
    expect(grip(drawn).attributes('role')).toBe('button')
    expect(grip(drawn).attributes('tabindex')).toBe('0')
    expect(grip(drawn).attributes('aria-label')).toBe('Drag this block')
    expect(grip(drawn).attributes('aria-keyshortcuts')).toBe('ArrowUp ArrowDown')
  })

  it('carries what it heads one place along the order', async () => {
    drawn = mountHeader()
    await grip(drawn).trigger('keydown', { key: 'ArrowUp' })
    await grip(drawn).trigger('keydown', { key: 'ArrowDown' })
    expect((drawn.emitted('step') ?? []).map((said) => (said as unknown[])[0])).toEqual([
      'up',
      'down',
    ])
  })

  it('does nothing for a key that is no way along the order', async () => {
    drawn = mountHeader()
    await grip(drawn).trigger('keydown', { key: 'ArrowLeft' })
    await grip(drawn).trigger('keydown', { key: 'Enter' })
    expect(drawn.emitted('step')).toBeUndefined()
  })

  // A key struck in a box the strip holds is that box's, whatever it is.
  it('leaves a key struck in something the strip holds to that thing', async () => {
    drawn = mountHeader({ default: '<input class="inside" />' })
    await drawn.get('.inside').trigger('keydown', { key: 'ArrowUp' })
    expect(drawn.emitted('step')).toBeUndefined()
  })

  // The pointer has the whole strip, so a press on the handle drags the strip.
  it('leaves the strip to be dragged by a press that lands on it', async () => {
    drawn = mountHeader()
    await pressOn(drawn, '.card-header__grip')
    expect(header(drawn).attributes('draggable')).toBe('true')
  })
})

describe('what the strip is drawn with', () => {
  it('puts what it is given at its start and what it is pressed for at its end', () => {
    drawn = mountHeader({
      default: '<span class="held">Title</span>',
      actions: '<button>Remove</button>',
    })
    expect(drawn.get('.card-header__held .held').text()).toBe('Title')
    expect(drawn.get('.card-header__actions button').text()).toBe('Remove')
  })
})
