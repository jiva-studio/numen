/**
 * The fields and choosers a row of settings is made of stand at the one height
 * the row names.
 *
 * The height is a class, so it is read off the class and not off a measurement:
 * nothing here lays anything out.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { NumberField } from './number-field'
import { SegmentedControl } from './segmented'
import { Select } from './select'
import { TimeField } from './time-field'

/** The height a row of controls shares, as the utility that paints it. */
const ROW = 'h-action'

const CHOICES = [
  { id: 'one', text: 'One' },
  { id: 'other', text: 'Other' },
]

describe('the height a row shares', () => {
  it('is what a field of digits stands at', () => {
    expect(mount(NumberField).get('input').classes()).toContain(ROW)
  })

  it('is what a field of the clock stands at', () => {
    expect(mount(TimeField).get('input').classes()).toContain(ROW)
  })

  it('is what a line of choices stands at', () => {
    expect(mount(Select, { props: { choices: CHOICES } }).get('button').classes()).toContain(ROW)
  })

  it('is what a row of segments stands at', () => {
    expect(mount(SegmentedControl, { props: { choices: CHOICES } }).classes()).toContain(ROW)
  })
})

describe('the clock the machine draws', () => {
  it('is drawn without the spinner and the cross of its own', () => {
    expect(mount(TimeField).get('input').classes()).toContain('time-field')
  })
})
