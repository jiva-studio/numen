/**
 * What the chips are announced as, and that pressing one offers the shares a
 * day can carry.
 *
 * A day carrying the whole of a day is a day nothing was said about, so the
 * model names only the days standing under it.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Days from './Days.vue'
import { weekFrom, WHOLE } from './week'

type DaysProps = InstanceType<typeof Days>['$props']

const mountDays = (props: Partial<DaysProps> = {}) =>
  mount(Days, { props: { modelValue: { sat: 50 }, ...props }, attachTo: document.body })

/** Every set of shares the row has handed on, in the order it handed them on. */
const handed = (row: ReturnType<typeof mountDays>): readonly unknown[] =>
  (row.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0])

/** The shares on offer, once a chip has been pressed. */
const offered = (): readonly string[] =>
  [...document.body.querySelectorAll('.menu__item')].map((one) => one.textContent?.trim() ?? '')

afterEach(() => {
  document.body.innerHTML = ''
})

describe('what is drawn', () => {
  it('draws the seven days, each under its whole name and what it carries', () => {
    const chips = mountDays().findAll('button')
    expect(chips).toHaveLength(7)
    expect(chips.map((chip) => chip.attributes('aria-label'))).toEqual([
      'Monday, 100%',
      'Tuesday, 100%',
      'Wednesday, 100%',
      'Thursday, 100%',
      'Friday, 100%',
      'Saturday, 50%',
      'Sunday, 100%',
    ])
    expect(chips.map((chip) => chip.text())).toEqual(['M', 'T', 'W', 'T', 'F', 'S', 'S'])
  })

  it('draws the week from the day it is given', () => {
    const chips = mountDays({ days: weekFrom('sun') }).findAll('button')
    expect(chips[0]?.attributes('aria-label')).toBe('Sunday, 100%')
  })

  // A day at the whole of it is the plain day it always was, and the further a
  // day stands under that the stronger it is filled.
  it('fills a day the further it stands under the whole of a day', () => {
    const chips = mountDays({ modelValue: { sat: 0, sun: 75 } }).findAll('button')
    const filling = (at: number) => chips[at]?.attributes('style') ?? ''
    expect(filling(0)).toContain('var(--numen-focus-bg) 0%')
    expect(filling(5)).toContain(`var(--numen-focus-bg) ${WHOLE}%`)
    expect(filling(6)).toContain('var(--numen-focus-bg) 25%')
  })

  it('says a day offers the shares rather than turning on the spot', () => {
    expect(mountDays().get('button').attributes('aria-haspopup')).toBe('menu')
  })
})

describe('giving a day a share', () => {
  it('offers the shares, from nothing to the whole of a day', async () => {
    const row = mountDays()
    await row.findAll('button')[5]?.trigger('click')
    expect(offered()).toEqual(['0%', '10%', '25%', '50%', '75%', '90%', '100%'])
  })

  it('offers the shares it was given, where a caller names its own', async () => {
    const row = mountDays({ shares: [0, 50, 100] })
    await row.findAll('button')[0]?.trigger('click')
    expect(offered()).toEqual(['0%', '50%', '100%'])
  })

  it('hands the day back at the share that was chosen, and leaves the rest', async () => {
    const row = mountDays()
    await row.findAll('button')[0]?.trigger('click')
    await document.body.querySelectorAll<HTMLElement>('.menu__item')[2]?.click()
    expect(handed(row)).toEqual([{ sat: 50, mon: 25 }])
  })

  // What carries the whole of a day is what nothing was said about.
  it('stops naming a day put back to the whole of a day', async () => {
    const row = mountDays()
    await row.findAll('button')[5]?.trigger('click')
    await document.body.querySelectorAll<HTMLElement>('.menu__item')[6]?.click()
    expect(handed(row)).toEqual([{}])
  })

  it('offers nothing while nobody may turn them', async () => {
    const row = mountDays({ disabled: true })
    await row.findAll('button')[1]?.trigger('click')
    expect(offered()).toEqual([])
    expect(handed(row)).toEqual([])
  })
})
