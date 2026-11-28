/** The list never loses a row: the stages are the same and in the same order whatever state each is in. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StageList from './StageList.vue'
import type { Stage } from './types'

const make = (states: Stage['state'][]): Stage[] =>
  states.map((state, at) => ({ id: `s${at}`, label: `Stage ${at + 1}`, state }))

const getLabels = (states: Stage['state'][]): string[] =>
  mount(StageList, { props: { stages: make(states) } })
    .findAll('li')
    .map((row) => row.find('.stage-list__label').text())

describe('a stage list', () => {
  it('shows every stage before anything started', () => {
    const list = mount(StageList, { props: { stages: make(['pending', 'pending', 'pending']) } })
    expect(list.findAll('li')).toHaveLength(3)
    expect(list.findAll('[data-state="pending"]')).toHaveLength(3)
  })

  it('keeps a done stage in its row with a check', () => {
    const list = mount(StageList, { props: { stages: make(['done', 'active', 'pending']) } })
    const first = list.findAll('li')[0]
    expect(first?.attributes('data-state')).toBe('done')
    expect(first?.find('svg').exists()).toBe(true)
    expect(first?.find('.stage-list__label').text()).toBe('Stage 1')
  })

  it('marks the active stage as the current step', () => {
    const list = mount(StageList, { props: { stages: make(['done', 'active', 'pending']) } })
    expect(list.findAll('li').map((row) => row.attributes('aria-current'))).toEqual([undefined, 'step', undefined])
  })

  it('keeps the order and the labels while the states change', () => {
    const expected = ['Stage 1', 'Stage 2', 'Stage 3']
    expect(getLabels(['pending', 'pending', 'pending'])).toEqual(expected)
    expect(getLabels(['done', 'active', 'pending'])).toEqual(expected)
    expect(getLabels(['done', 'done', 'done'])).toEqual(expected)
  })
})
