/**
 * The strip a tile or a block is carried by: what a gesture on it takes hold
 * of, and what a key struck on it does.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import CardHeader from './CardHeader.vue'

const mountHeader = (slots: Record<string, string> = {}) =>
  mount(CardHeader, { props: { carry: 'Carry this block' }, attachTo: document.body, slots })

let drawn: ReturnType<typeof mountHeader> | null = null

afterEach(() => {
  drawn?.unmount()
  drawn = null
})

const header = (strip: ReturnType<typeof mountHeader>) => strip.get('[role="group"]')

/** A press that lands on something inside the strip and rises to it. */
const pressOn = async (strip: ReturnType<typeof mountHeader>, selector: string) => {
  await strip.get(selector).trigger('pointerdown')
}

describe('the strip itself', () => {
  it('says what taking hold of it does, to a reader and to a pointer', () => {
    drawn = mountHeader()
    expect(header(drawn).attributes('aria-label')).toBe('Carry this block')
    expect(header(drawn).attributes('title')).toBe('Carry this block')
    expect(header(drawn).attributes('role')).toBe('group')
  })

  it('is a stop on the way round the screen, and is drawn as a header', () => {
    drawn = mountHeader()
    expect(header(drawn).attributes('tabindex')).toBe('0')
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

  it('leaves the strip to be carried where the press landed on nothing worked', async () => {
    drawn = mountHeader({ default: INSIDE })
    await pressOn(drawn, '.plain')
    expect(header(drawn).attributes('draggable')).toBe('true')
  })

  // A press that began in a box may travel off the strip and be let go
  // anywhere, so the end of it is heard wherever it happens.
  it('may be carried again once the press is let go of, wherever that was', async () => {
    drawn = mountHeader({ default: INSIDE })
    await pressOn(drawn, '.inside')

    window.dispatchEvent(new Event('pointerup'))
    await drawn.vm.$nextTick()
    expect(header(drawn).attributes('draggable')).toBe('true')
  })

  it('may be carried again where the press was cancelled', async () => {
    drawn = mountHeader({ default: INSIDE })
    await pressOn(drawn, '.inside')

    window.dispatchEvent(new Event('pointercancel'))
    await drawn.vm.$nextTick()
    expect(header(drawn).attributes('draggable')).toBe('true')
  })
})

describe('a key struck on the strip', () => {
  it('carries it one place along the order', async () => {
    drawn = mountHeader()
    await header(drawn).trigger('keydown', { key: 'ArrowUp' })
    await header(drawn).trigger('keydown', { key: 'ArrowDown' })
    expect((drawn.emitted('step') ?? []).map((said) => (said as unknown[])[0])).toEqual([
      'up',
      'down',
    ])
  })

  it('does nothing for a key that is no way along the order', async () => {
    drawn = mountHeader()
    await header(drawn).trigger('keydown', { key: 'ArrowLeft' })
    await header(drawn).trigger('keydown', { key: 'Enter' })
    expect(drawn.emitted('step')).toBeUndefined()
  })

  // A key struck in a box the strip holds is that box's, whatever it is.
  it('leaves a key struck in something it holds to that thing', async () => {
    drawn = mountHeader({ default: '<input class="inside" />' })
    await drawn.get('.inside').trigger('keydown', { key: 'ArrowUp' })
    expect(drawn.emitted('step')).toBeUndefined()
  })
})

describe('what the strip is drawn with', () => {
  it('puts what it is given at its start and what it is pressed for at its end', () => {
    drawn = mountHeader({ default: '<span class="held">Title</span>', deeds: '<button>Remove</button>' })
    expect(drawn.get('.card-header__held .held').text()).toBe('Title')
    expect(drawn.get('.card-header__deeds button').text()).toBe('Remove')
  })
})
