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

/** The controls over a document of that many pages, open at the first. */
const drawn = (pages = 200, at = 0) =>
  mount(ReaderToolbar, { props: { pages, at, 'onUpdate:at': (page: number) => void page } })

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
