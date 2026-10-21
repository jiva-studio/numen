/**
 * How a line of work is laid out: which words go where, and what stands beside
 * them. The stories are where it is drawn.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Activity from './Activity.vue'

type Props = InstanceType<typeof Activity>['$props']

const line = (props: Props) => mount(Activity, { props })

describe('a line of work', () => {
  it('carries what is happening and what it is happening to on lines of their own', () => {
    const rendered = line({
      text: 'Proofreading the transcript',
      about: 'A Conversation in Vrindavan, 1972-11-04.md',
      isWorking: true,
      tally: { done: 9, total: 100 },
      left: '4:05',
    })

    expect(rendered.get('.activity__says').text()).toBe('Proofreading the transcript')
    expect(rendered.get('.activity__about').text()).toBe('A Conversation in Vrindavan, 1972-11-04.md')
    expect(rendered.get('.activity__words').classes()).toContain('flex-col')
  })

  it('draws how far and how long as figures, alongside the words', () => {
    const rendered = line({
      text: 'Learning what it says',
      about: 'Sabhaparva.epub',
      isWorking: true,
      tally: { done: 1200, total: 36560 },
      left: '1:58:20',
    })

    expect(rendered.get('.activity__percent').text()).toBe('3%')
    expect(rendered.get('.activity__left').text()).toBe('1:58:20')
  })

  it('keeps a count to two lines, and lets a reason run on', () => {
    const counting = line({
      text: 'Reading',
      about: 'a-note-nobody-shortened-before-they-filed-it-away.md',
      tally: { done: 2, total: 8 },
    })
    expect(counting.get('.activity__says').classes()).toContain('truncate')
    expect(counting.get('.activity__about').classes()).toContain('truncate')

    const resting = line({ text: 'Reading', about: 'permission denied' })
    expect(resting.get('.activity__about').classes()).toContain('line-clamp-2')

    const words = line({ text: 'Searching by words — no model to learn what it says' })
    expect(words.get('.activity__says').classes()).toContain('line-clamp-3')
  })

  it('says nothing about a count it was given none of', () => {
    const rendered = line({ text: 'Reading', about: 'Sabhaparva.epub', isWorking: true })

    expect(rendered.find('.activity__count').exists()).toBe(false)
  })

  it('waits and not draw a share of nothing', () => {
    const rendered = line({ text: 'Fetching models', about: 'inference.onnx', isWorking: true })

    expect(rendered.find('.activity__percent').exists()).toBe(false)
    expect(rendered.find('.activity__spinner').exists()).toBe(true)
  })

  it('follows what the work moved on to', async () => {
    const rendered = line({
      text: 'Indexing',
      about: 'Sabhaparva.epub',
      isWorking: true,
      tally: { done: 3, total: 12 },
    })
    expect(rendered.get('.activity__about').text()).toBe('Sabhaparva.epub')

    await rendered.setProps({ about: 'notes/Vrindavan.md', tally: { done: 8, total: 12 } })

    expect(rendered.get('.activity__about').text()).toBe('notes/Vrindavan.md')
    expect(rendered.get('.activity__percent').text()).toBe('66%')
  })
})
