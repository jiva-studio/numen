import { describe, expect, it } from 'vitest'
import { createQuestionSession } from './createQuestionSession'

const OPTIONS = [
  { value: 'low', title: 'Low' },
  { value: 'high', title: 'High' },
]

describe('a question session', () => {
  it('draws nothing before a plugin asks', () => {
    expect(createQuestionSession().view.value).toBeUndefined()
  })

  it('starts the line with its value and offers it as the one row', async () => {
    const session = createQuestionSession()
    const asking = session.ask({ title: 'Owner', value: 'Ann' })

    expect(session.view.value).toMatchObject({ typed: 'Ann', placeholder: 'Owner' })
    expect(session.view.value?.groups[0]?.items.map((one) => one.title)).toStrictEqual([
      'Owner: “Ann”',
    ])
    session.dismiss()
    await asking
  })

  it('resolves the line with the text typed, trimmed', async () => {
    const session = createQuestionSession()
    const asking = session.ask({ title: 'Owner', value: 'Ann' })

    session.view.value?.type('  Bob ')
    session.view.value?.answer('answer')

    await expect(asking).resolves.toBe('Bob')
    expect(session.view.value).toBeUndefined()
  })

  it('accepts an empty line', async () => {
    const session = createQuestionSession()
    const asking = session.ask({ title: 'Owner' })

    session.view.value?.answer('answer')

    await expect(asking).resolves.toBe('')
  })

  it('offers the options in order and filters them by what is typed', async () => {
    const session = createQuestionSession()
    const choosing = session.choose({ title: 'Level', options: OPTIONS })

    expect(session.view.value?.groups[0]?.items.map((one) => one.id)).toStrictEqual(['low', 'high'])
    session.view.value?.type('hi')
    expect(session.view.value?.groups[0]?.items.map((one) => one.id)).toStrictEqual(['high'])
    session.dismiss()
    await choosing
  })

  it('resolves a choice with the value of the option picked', async () => {
    const session = createQuestionSession()
    const choosing = session.choose({ title: 'Level', options: OPTIONS })

    session.view.value?.answer('high')

    await expect(choosing).resolves.toBe('high')
  })

  it('resolves with undefined when dismissed, and when another question replaces it', async () => {
    const session = createQuestionSession()
    const first = session.ask({ title: 'Owner' })
    const second = session.choose({ title: 'Level', options: OPTIONS })

    await expect(first).resolves.toBeUndefined()
    session.dismiss()
    await expect(second).resolves.toBeUndefined()
    expect(session.view.value).toBeUndefined()
  })
})
