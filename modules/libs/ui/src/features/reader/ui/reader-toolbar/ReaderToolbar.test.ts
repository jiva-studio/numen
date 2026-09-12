/**
 * The page a person asks the controls for.
 *
 * The field holds the page in front until somebody types over it, and a browser
 * says a field was committed twice for one keystroke: once for the key and once
 * for the change it made.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ReaderToolbar from './ReaderToolbar.vue'
import { CLOSEST, FURTHEST, NEARER, READER_WORDS } from '../strip'

/** The controls over a document of that many pages, open at the first. */
const drawn = (pages = 200, at = 0) =>
  mount(ReaderToolbar, { props: { pages, at, 'onUpdate:at': (page: number) => void page } })

/**
 * The same, with a caller standing behind how close the page is drawn: it puts
 * back whatever it is handed, which is what walking the zoom from a press is.
 */
const zooming = (zoom = FURTHEST) => {
  const controls = mount(ReaderToolbar, { props: { pages: 200, at: 0, zoom } })
  const button = (label: string) => controls.get(`button[aria-label="${label}"]`)

  return {
    controls,
    button,
    /** Every zoom it has handed on, in the order it handed them on. */
    handed: (): readonly unknown[] =>
      (controls.emitted('update:zoom') ?? []).map((said) => (said as unknown[])[0]),
    press: async (label: string) => {
      await button(label).trigger('click')
      const last = controls.emitted('update:zoom')?.at(-1)
      if (last) await controls.setProps({ zoom: last[0] as number })
    },
  }
}

describe('the page asked for', () => {
  it('is the page typed, counted from one', async () => {
    const controls = drawn()
    const field = controls.get('input[type="number"]')

    await field.setValue('12')
    await field.trigger('keydown.enter')

    expect(controls.emitted('update:at')?.at(-1)).toEqual([11])
  })

  // A key and the change it made are two events on one field, and the page
  // stands for the second of them.
  it('is asked for once when one page is typed', async () => {
    const controls = drawn()
    const field = controls.get('input[type="number"]')

    await field.setValue('12')
    await field.trigger('keydown.enter')
    await field.trigger('change')

    expect(controls.emitted('update:at')).toHaveLength(1)
  })

  it('is nothing at all when the field was committed with nothing in it', async () => {
    const controls = drawn()
    const field = controls.get('input[type="number"]')

    await field.trigger('keydown.enter')
    await field.trigger('change')

    expect(controls.emitted('update:at')).toBeUndefined()
  })

  it('is nothing at all for a field holding only room', async () => {
    const controls = drawn()
    const field = controls.get('input[type="number"]')

    await field.setValue('   ')
    await field.trigger('change')

    expect(controls.emitted('update:at')).toBeUndefined()
  })
})

describe('how close the page is drawn', () => {
  it('is handed on a step at a time, and the caller is what walks it', async () => {
    const one = zooming()

    await one.press(READER_WORDS.closer)
    await one.press(READER_WORDS.closer)

    expect(one.handed()).toEqual([FURTHEST * NEARER, FURTHEST * NEARER * NEARER])
  })

  // The step is a multiplier, so the last one before an end overshoots it. What
  // is handed on is the end, never what the multiplication came to.
  it('goes no closer in than the closest, whatever the step comes to', async () => {
    const one = zooming(CLOSEST / NEARER + 0.5)

    await one.press(READER_WORDS.closer)

    expect(one.handed()).toEqual([CLOSEST])
  })

  it('goes no further out than a whole page in the room', async () => {
    const one = zooming(FURTHEST * 1.1)

    await one.press(READER_WORDS.further)

    expect(one.handed()).toEqual([FURTHEST])
  })

  it('offers no press at the end it already stands at', () => {
    const out = zooming(FURTHEST)
    expect(out.button(READER_WORDS.further).attributes('disabled')).toBeDefined()
    expect(out.button(READER_WORDS.closer).attributes('disabled')).toBeUndefined()

    const inward = zooming(CLOSEST)
    expect(inward.button(READER_WORDS.closer).attributes('disabled')).toBeDefined()
    expect(inward.button(READER_WORDS.further).attributes('disabled')).toBeUndefined()
  })
})
