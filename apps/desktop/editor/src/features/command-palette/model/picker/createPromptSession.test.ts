import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPromptSession } from './createPromptSession'

const OPTIONS = [
  { value: 'low', title: 'Low' },
  { value: 'high', title: 'High' },
]

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('a question session', () => {
  it('draws nothing before a plugin asks', () => {
    expect(createPromptSession().view.value).toBeUndefined()
  })

  it('starts the line with its value, names the question in the placeholder and heading and lists no row', async () => {
    const session = createPromptSession()
    const asking = session.promptText({ title: 'Owner', value: 'Ann' })

    expect(session.view.value).toMatchObject({
      typed: 'Ann',
      placeholder: 'Owner',
      heading: 'Owner',
      groups: [],
    })
    session.dismiss()
    await asking
  })

  it('starts an empty line with no row either', async () => {
    const session = createPromptSession()
    const asking = session.promptText({ title: 'Owner' })

    expect(session.view.value).toMatchObject({ typed: '', groups: [] })
    session.dismiss()
    await asking
  })

  it('resolves the line with the text typed, trimmed', async () => {
    const session = createPromptSession()
    const asking = session.promptText({ title: 'Owner', value: 'Ann' })

    session.view.value?.enterText('  Bob ')
    session.view.value?.submit()

    await expect(asking).resolves.toBe('Bob')
  })

  it('accepts an empty line', async () => {
    const session = createPromptSession()
    const asking = session.promptText({ title: 'Owner' })

    session.view.value?.submit()

    await expect(asking).resolves.toBe('')
  })

  it('offers the options in order, with no row to clear, and filters them by what is typed', async () => {
    const session = createPromptSession()
    const choosing = session.selectOption({ title: 'Level', options: OPTIONS })

    expect(session.view.value?.groups[0]?.items.map((one) => one.id)).toStrictEqual(['low', 'high'])
    session.view.value?.enterText('hi')
    expect(session.view.value?.groups[0]?.items.map((one) => one.id)).toStrictEqual(['high'])
    session.dismiss()
    await choosing
  })

  it('shows the question only as the heading and leaves the options group untitled', async () => {
    const session = createPromptSession()
    const choosing = session.selectOption({ title: 'Level', options: OPTIONS })

    expect(session.view.value?.heading).toBe('Level')
    expect(session.view.value?.groups).toHaveLength(1)
    expect(session.view.value?.groups[0]?.title).toBe('')
    session.dismiss()
    await choosing
  })

  it('keeps the question out of the placeholder of a choice, which only filters its options', async () => {
    const session = createPromptSession()
    const choosing = session.selectOption({ title: 'Level', options: OPTIONS })

    expect(session.view.value?.heading).toBe('Level')
    expect(session.view.value?.placeholder).not.toBe('Level')
    session.dismiss()
    await choosing
  })

  it('resolves a choice with the value of the option picked', async () => {
    const session = createPromptSession()
    const choosing = session.selectOption({ title: 'Level', options: OPTIONS })

    session.view.value?.answer('high')

    await expect(choosing).resolves.toBe('high')
  })

  it('resolves with undefined when dismissed, and when another question replaces it', async () => {
    const session = createPromptSession()
    const first = session.promptText({ title: 'Owner' })
    const second = session.selectOption({ title: 'Level', options: OPTIONS })

    await expect(first).resolves.toBeUndefined()
    session.dismiss()
    await expect(second).resolves.toBeUndefined()
    expect(session.view.value).toBeUndefined()
  })

  it('gives each question a step of its own', async () => {
    const session = createPromptSession()
    const first = session.promptText({ title: 'Owner' })
    const firstStep = session.view.value?.step
    const second = session.promptText({ title: 'Client' })

    expect(session.view.value?.step).not.toBe(firstStep)
    session.dismiss()
    await Promise.all([first, second])
  })

  describe('between two questions', () => {
    it('keeps the palette up until the next question arrives', async () => {
      const session = createPromptSession()
      const first = session.promptText({ title: 'Owner' })
      session.view.value?.submit()
      await first

      expect(session.view.value).toBeDefined()
      const second = session.promptText({ title: 'Client' })
      vi.runAllTimers()

      expect(session.view.value?.placeholder).toBe('Client')
      session.dismiss()
      await second
    })

    it('closes the palette once no question follows', async () => {
      const session = createPromptSession()
      const asking = session.promptText({ title: 'Owner' })
      session.view.value?.submit()
      await asking

      vi.runAllTimers()

      expect(session.view.value).toBeUndefined()
    })

    it('answers a question once', async () => {
      const session = createPromptSession()
      const asking = session.promptText({ title: 'Owner', value: 'Ann' })
      session.view.value?.submit()
      session.view.value?.enterText('Bob')
      session.view.value?.submit()

      await expect(asking).resolves.toBe('Ann')
    })
  })
})
