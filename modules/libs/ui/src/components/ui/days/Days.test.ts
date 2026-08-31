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

  // The week is read as the work standing on it: a day carrying the whole of a
  // day is full colour, and a day carrying none has none.
  it('fills a day in step with what it carries', () => {
    const chips = mountDays({ modelValue: { sat: 0, sun: 75 } }).findAll('button')
    const filling = (at: number) => chips[at]?.attributes('style') ?? ''
    expect(filling(0)).toContain(`var(--numen-focus-bg) ${WHOLE}%`)
    expect(filling(5)).toContain('var(--numen-focus-bg) 0%')
    expect(filling(6)).toContain('var(--numen-focus-bg) 75%')
  })

  it('says a day offers the shares rather than turning on the spot', () => {
    expect(mountDays().get('button').attributes('aria-haspopup')).toBe('menu')
  })

  // A chip carries a share and offers a menu. It holds nothing down, so it
  // announces no state of its own beyond the share it is labelled with.
  it('is a row of buttons offering menus, and not a set of toggles', () => {
    const row = mountDays()
    expect(row.get('[data-slot="days"]').attributes('role')).toBe('toolbar')

    const chips = row.findAll('button')
    expect(chips).toHaveLength(7)
    for (const chip of chips) {
      expect(chip.attributes('aria-pressed')).toBeUndefined()
      expect(chip.attributes('aria-checked')).toBeUndefined()
      expect(chip.attributes('role')).toBeUndefined()
    }
  })

  it('says on the chip whether its shares are open', async () => {
    const row = mountDays()
    const chip = row.findAll('button')[5]
    expect(chip?.attributes('aria-expanded')).toBe('false')
    await chip?.trigger('click')
    expect(chip?.attributes('aria-expanded')).toBe('true')
    expect(row.findAll('button')[0]?.attributes('aria-expanded')).toBe('false')
  })
})

describe('a day carrying a share the offer does not name', () => {
  it('says the share it is drawn at, and is drawn at the share it says', async () => {
    const row = mountDays({ modelValue: { sat: 37, sun: 400 } })
    const chips = row.findAll('button')
    expect(chips[5]?.attributes('aria-label')).toBe('Saturday, 37%')
    expect(chips[5]?.attributes('style')).toContain('var(--numen-focus-bg) 37%')
    expect(chips[6]?.attributes('aria-label')).toBe('Sunday, 100%')
    expect(chips[6]?.attributes('style')).toContain('var(--numen-focus-bg) 100%')
  })

  it('offers that share too, in its place among them and as the one in force', async () => {
    const row = mountDays({ modelValue: { sat: 37 } })
    await row.findAll('button')[5]?.trigger('click')
    expect(offered()).toEqual(['0%', '10%', '25%', '37%', '50%', '75%', '90%', '100%'])

    const marked = [...document.body.querySelectorAll('.menu__item')]
      .filter((one) => one.getAttribute('aria-checked') === 'true')
      .map((one) => one.textContent?.trim())
    expect(marked).toEqual(['37%'])
  })
})

describe('the keyboard while the shares are offered', () => {
  /** The chip pressed, focused as the keyboard would leave it. */
  const asked = async (row: ReturnType<typeof mountDays>, at: number) => {
    const chip = row.findAll('button')[at]
    const element = chip?.element as HTMLElement
    element.focus()
    await chip?.trigger('click')
    return element
  }

  const items = (): readonly HTMLElement[] => [
    ...document.body.querySelectorAll<HTMLElement>('.menu__item'),
  ]

  it('opens on the share the day carries, and says which of them it is', async () => {
    const row = mountDays()
    await asked(row, 5)
    expect(document.activeElement).toBe(items()[3])
    expect(items().map((one) => one.getAttribute('role'))).toEqual(
      Array(7).fill('menuitemradio'),
    )
    expect(items().map((one) => one.getAttribute('aria-checked'))).toEqual([
      'false',
      'false',
      'false',
      'true',
      'false',
      'false',
      'false',
    ])
  })

  it('gives the keyboard back to the chip once a share is chosen', async () => {
    const row = mountDays()
    const chip = await asked(row, 5)
    items()[2]?.click()
    await row.vm.$nextTick()
    expect(document.activeElement).toBe(chip)
  })

  it('gives the keyboard back to the chip where nothing is chosen at all', async () => {
    const row = mountDays()
    const chip = await asked(row, 0)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await row.vm.$nextTick()
    expect(items()).toEqual([])
    expect(document.activeElement).toBe(chip)
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
