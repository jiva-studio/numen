/**
 * The settings under the control: the rows a person types into, and what the
 * tab says about what it could not do.
 *
 * A row is written once the typing settles, and the words a row is drawn in
 * come from one place, which is what the two sources are read for.
 */
import { describe, expect, it } from 'vitest'
import { ref, shallowRef } from 'vue'
import { mount } from '@vue/test-utils'
import PresetTab from './PresetTab.vue'
import tabSource from './PresetTab.vue?raw'
import rowsSource from './preset-settings/PresetSettings.vue?raw'
import sliderSource from './curve-slider/CurveSlider.vue?raw'
import tilesSource from './curve-slider/CurveTiles.vue?raw'
import { NO_BOUNDS, type Field, type PresetTabState, type SettingValue } from '../types'
import { BOUNDS, mountPresetTab, rows, tabAt } from '../fixtures'
import { WORDS as words } from '../words'

describe('the settings under the control', () => {
  it('draws a row for each, with what it is beside it', () => {
    const { tab } = mountPresetTab()
    expect(tab.text()).toContain(words.fieldName('minutesADay'))
    expect(tab.text()).toContain(words.fieldDetail('minutesADay'))
    expect(tab.text()).toContain(words.fieldName('load'))
    expect(tab.findAll('[data-preset-row]')).toHaveLength(6)
  })

  // What a budget is spent on is a row like any other, and a person moving it
  // writes the group the way every other row does. It stands under the goal
  // whose budget is cards.
  it('offers the two things a budget is spent on', async () => {
    const { tab, done } = mountPresetTab({ goal: 'retention' }, { goal: 'retention' })
    expect(tab.text()).toContain(words.fieldName('counts'))

    const unit = tab
      .findAll('button')
      .find((one) => one.text() === words.budgetUnitName('shows'))
    await unit?.trigger('click')
    expect(done).toStrictEqual(['type counts shows', 'settle'])
  })

  // The share of a day the debt takes is moved along its whole range, so any
  // of the hundred stands and none of them is a named position. The track
  // draws no figure, so the per cent is read out beside it.
  it('moves the share of a day the debt takes along a track, and reads it out', async () => {
    const { tab, done } = mountPresetTab({}, { backlog: 70 })
    expect(tab.text()).toContain(words.fieldName('backlog'))
    const row = tab.get('[data-preset-row="backlog"]')

    await tab.vm.$nextTick()
    const track = row.get('[role="slider"]')
    expect(track.attributes('aria-valuenow')).toBe('70')
    // The track runs the ends the read answered, and nothing outside them is
    // taken.
    expect(track.attributes('aria-valuemin')).toBe(`${BOUNDS.backlog.least}`)
    expect(track.attributes('aria-valuemax')).toBe(`${BOUNDS.backlog.most}`)
    expect(track.attributes('aria-labelledby')).toBe('preset-backlog')
    expect(row.get('[data-preset="percent"]').text()).toBe(words.percent(70))

    // The value is handed on as the handle moves, and the row is written once
    // the key is let go of.
    await track.trigger('keydown', { key: 'ArrowLeft' })
    expect(done).toStrictEqual(['type backlog 69'])
    await track.trigger('keyup', { key: 'ArrowLeft' })
    expect(done).toStrictEqual(['type backlog 69', 'settle'])
  })

  // The rule stands over the one value it reads, and the value the other rule
  // reads is not drawn at all — the same rule a goal follows for its budgets.
  it('offers the two rules for the learned, and draws the value the chosen one reads', async () => {
    const byInterval = mountPresetTab({}, { learned: 'interval', interval: 21 })
    expect(rows(byInterval.tab)).toContain(words.fieldName('learned'))
    expect(rows(byInterval.tab)).toContain(words.fieldName('interval'))
    expect(rows(byInterval.tab)).not.toContain(words.fieldName('retention'))

    const byRetention = mountPresetTab({}, { learned: 'retention' })
    expect(rows(byRetention.tab)).toContain(words.fieldName('retention'))
    expect(rows(byRetention.tab)).not.toContain(words.fieldName('interval'))
  })

  // Under a goal of retention the target is the knob's own value and the rule
  // reads it too. It is one key, so the receipt draws it once.
  it('draws the target once where the goal and the rule both read it', () => {
    const { tab } = mountPresetTab({ goal: 'retention' }, { goal: 'retention', learned: 'retention' })
    const named = rows(tab).filter((one) => one === words.fieldName('retention'))
    expect(named).toHaveLength(1)
  })

  it('opens the rules on the line that says which one is in force', async () => {
    const { tab } = mountPresetTab({}, { learned: 'interval' })
    const line = tab.get('[data-preset="choice"]')
    expect(line.text()).toContain(words.ruleName('interval'))
    expect(line.attributes('aria-haspopup')).toBe('menu')
    expect(document.body.querySelectorAll('[role="menuitemradio"]')).toHaveLength(0)

    await line.trigger('click')
    const offered = [...document.body.querySelectorAll('[role="menuitemradio"]')].map(
      (one) => one.textContent?.trim() ?? '',
    )
    expect(offered).toStrictEqual([words.ruleName('interval'), words.ruleName('retention')])
  })

  it('hands on the rule that was chosen, and is done with it at once', async () => {
    const { tab, done } = mountPresetTab({}, { learned: 'interval' })
    await tab.get('[data-preset="choice"]').trigger('click')
    const chosen = [...document.body.querySelectorAll<HTMLElement>('[role="menuitemradio"]')].find(
      (one) => one.textContent?.trim() === words.ruleName('retention'),
    )
    chosen?.click()
    await tab.vm.$nextTick()
    expect(done).toStrictEqual(['type learned retention', 'settle'])
  })

  it('holds the days a card is put off inside what a preset may hold', async () => {
    const { tab, done } = mountPresetTab({}, { learned: 'interval', interval: 21 })
    const row = tab.get('[data-preset-row="interval"]')
    const field = row.get<HTMLInputElement>('input')
    expect(field.element.value).toBe('21')
    expect(field.attributes('aria-valuemin')).toBe(`${BOUNDS.interval.least}`)
    expect(field.attributes('aria-valuemax')).toBe(`${BOUNDS.interval.most}`)
    await field.setValue('30')
    expect(done).toStrictEqual(['type interval 30'])
  })

  // A tab draws its rows before the first read lands. Until the application
  // has said how far a field goes, the field stands at the ends it draws
  // itself with.
  it('leaves a field the application has said nothing about at its own ends', () => {
    const one = tabAt({}, { learned: 'interval', interval: 21 })
    const state: PresetTabState = { ...one.state, bounds: shallowRef(NO_BOUNDS) }
    const tab = mount(PresetTab, { props: { state } })
    const field = tab.get('[data-preset-row="interval"]').get<HTMLInputElement>('input')
    expect(field.element.value).toBe('21')
    expect(field.attributes('aria-valuemax')).not.toBe(`${BOUNDS.interval.most}`)
  })
})

// A tab showing a failure is a tab with a way back: the file is read again,
// which is what a fixed permission bit or a restored folder wants.
describe('what the tab says went wrong', () => {
  const mountWithError = (words: string) => {
    const one = tabAt()
    const state: PresetTabState = { ...one.state, errorMessage: ref(words) }
    return { tab: mount(PresetTab, { props: { state } }), done: one.done }
  }

  it('offers reading the file again beside what it says', async () => {
    const { tab, done } = mountWithError(words.notRead('missing'))
    const alert = tab.get('[role="alert"]')
    expect(alert.text()).toContain(words.notRead('missing'))
    await alert.get('button').trigger('click')
    expect(done).toStrictEqual(['reload'])
  })

  it('offers it for a refused write as well as a refused read', async () => {
    const { tab, done } = mountWithError(words.notSaved('unreadable'))
    await tab.get('[role="alert"] button').trigger('click')
    expect(done).toStrictEqual(['reload'])
  })

  it('offers nothing where there is nothing to say', () => {
    const { tab } = mountPresetTab()
    expect(tab.findAll('[role="alert"]')).toHaveLength(0)
  })
})

// A file someone else wrote while the tab stood open is not a failure, so it
// is said as a status and not as an alert. The way out is the same one: read
// the file again.
describe('a file that changed under the tab', () => {
  it('says so, and offers reading the file again', async () => {
    const one = tabAt()
    const state: PresetTabState = { ...one.state, hasChanged: ref(true) }
    const tab = mount(PresetTab, { props: { state } })
    const said = tab.get('[role="status"].preset__answering')
    expect(said.text()).toContain(words.changed)
    await said.get('button').trigger('click')
    expect(one.done).toStrictEqual(['reload'])
  })

  it('says nothing where the file is the one the tab read', () => {
    const { tab } = mountPresetTab()
    expect(tab.findAll('[role="status"].preset__answering')).toHaveLength(0)
    expect(tab.text()).not.toContain(words.changed)
  })
})

// The row of days draws a level and hands back a day and a level. That the
// whole of a day is a hundred, and that a day back at it stops being named,
// are the file's own way of writing the week and belong to this tab.
describe('the load of the week', () => {
  /** A tab whose row of days is watched for what it puts into the settings. */
  const mountWithLoad = (load: Record<string, number>) => {
    const one = tabAt({}, { load })
    const put: [Field, SettingValue][] = []
    const state: PresetTabState = {
      ...one.state,
      updateSetting: (field, value) => {
        put.push([field, value])
        one.done.push('type')
      },
    }
    return { tab: mount(PresetTab, { props: { state } }), put, done: one.done }
  }

  /** The chips of the row that draws the week. */
  const chips = (tab: ReturnType<typeof mount>) => tab.findAll('[data-slot="weekday-chips"] button')

  const getMenuItems = (): readonly HTMLElement[] => [
    ...document.body.querySelectorAll<HTMLElement>('[role="menuitemradio"]'),
  ]

  it('draws a day not named at the whole of a day, and the rest where they stand', () => {
    const { tab } = mountWithLoad({ sat: 50, sun: 0 })
    expect(chips(tab).map((chip) => chip.attributes('aria-label'))).toStrictEqual([
      'Monday, 100%',
      'Tuesday, 100%',
      'Wednesday, 100%',
      'Thursday, 100%',
      'Friday, 100%',
      'Saturday, 50%',
      'Sunday, 0%',
    ])
  })

  it('offers the shares of a day, from nothing to the whole of it', async () => {
    const { tab } = mountWithLoad({})
    await chips(tab)[0]?.trigger('click')
    expect(getMenuItems().map((one) => one.textContent?.trim())).toStrictEqual([
      '0%',
      '10%',
      '25%',
      '50%',
      '75%',
      '90%',
      '100%',
    ])
  })

  it('writes the day it was handed at the share chosen, leaving the others', async () => {
    const { tab, put, done } = mountWithLoad({ sat: 50, sun: 0 })
    await chips(tab)[0]?.trigger('click')
    getMenuItems()[2]?.click()
    await tab.vm.$nextTick()

    expect(put).toStrictEqual([['load', { sat: 50, sun: 0, mon: 25 }]])
    expect(done).toStrictEqual(['type', 'settle'])
  })

  it('stops naming a day put back to the whole of a day', async () => {
    const { tab, put } = mountWithLoad({ sat: 50, sun: 0 })
    await chips(tab)[5]?.trigger('click')
    getMenuItems()[6]?.click()
    await tab.vm.$nextTick()

    expect(put).toStrictEqual([['load', { sun: 0 }]])
  })
})

describe('the line the tab is read against', () => {
  /** What one selector declares, as the file it is written in writes it. */
  const styleOf = (sheet: string, selector: string): string => {
    const opens = sheet.indexOf(`\n${selector} {`)
    expect(opens, `no rule for ${selector}`).toBeGreaterThan(-1)
    return sheet.slice(opens, sheet.indexOf('}', opens))
  }

  /** What a rule holds a block off the column by, on the two sides it has. */
  const getSpacing = (style: string, names: readonly string[]): readonly string[] =>
    [...style.matchAll(/^\s*([a-z-]+):\s*([^;]+);/gm)]
      .filter(([, name]) => names.includes(name ?? ''))
      .map(([, name, value]) => `${name}: ${value}`)
      .filter((one) => !one.endsWith(': 0'))

  const APART = ['margin', 'margin-inline', 'margin-inline-start', 'margin-inline-end']
  const AIR = ['padding', 'padding-inline', 'padding-inline-start', 'padding-inline-end']

  /** The blocks of the column that draw a box, and the ones that draw none. */
  const BOXED = [
    [tilesSource, '.curve-slider__material'],
    [sliderSource, '.curve-slider__island'],
    [tabSource, '.preset__stopped'],
  ] as const
  const BARE = [
    [tabSource, '.preset__label'],
    [tabSource, '.preset__unpointed'],
    [rowsSource, '.preset-settings__row'],
  ] as const

  it('puts the edge of every box on the edge of the column', () => {
    for (const [sheet, selector] of BOXED) {
      expect(getSpacing(styleOf(sheet, selector), APART), selector).toStrictEqual([])
    }
  })

  it('puts the text of every block that draws no box on that same edge', () => {
    for (const [sheet, selector] of BARE) {
      expect(getSpacing(styleOf(sheet, selector), [...APART, ...AIR]), selector).toStrictEqual([])
    }
  })
})

// Every figure a person reads of a chance of recall is a percentage: the mark
// on the curve, the words beside it, the axis. The row it is typed into is one.
describe('the row a chance of recall is typed into', () => {
  it('stands in per cent, and is bounded and stepped in per cent', () => {
    const { tab } = mountPresetTab({ goal: 'retention' }, { goal: 'retention', retention: 0.87 })
    const field = tab
      .findAll('[role="spinbutton"]')
      .find((one) => one.attributes('aria-valuemax') === '99')

    expect(field).toBeDefined()
    expect((field?.element as HTMLInputElement).value).toBe('87')
    expect(field?.attributes('aria-valuemin')).toBe('70')
    expect(field?.attributes('aria-valuenow')).toBe('87')
  })

  it('hands a percentage back as the share the settings hold', async () => {
    const { tab, done } = mountPresetTab({ goal: 'retention' }, { goal: 'retention', retention: 0.87 })
    const field = tab
      .findAll('[role="spinbutton"]')
      .find((one) => one.attributes('aria-valuemax') === '99')

    await field?.setValue('90')
    expect(done).toStrictEqual(['type retention 0.9'])
  })
})

// The rules are a choice between two, and the line that asked for them is where
// the keyboard comes back to.
describe('the rules under the learned row', () => {
  it('marks the rule in force and returns the focus to the line that asked', async () => {
    // Attached to the page, because taking the focus back is what is measured.
    const one = tabAt({}, { learned: 'interval' })
    const tab = mount(PresetTab, { props: { state: one.state }, attachTo: document.body })
    const line = tab.get('[data-preset="choice"]')
    await line.trigger('click')

    // The menu is drawn onto the page rather than inside the tab.
    const items = [...document.querySelectorAll('[role="menuitemradio"]')]
    expect(items.map((one) => one.getAttribute('aria-checked'))).toStrictEqual(['true', 'false'])

    ;(items[1] as HTMLElement).click()
    await tab.vm.$nextTick()
    await tab.vm.$nextTick()
    expect(document.activeElement).toBe(line.element)
  })
})
