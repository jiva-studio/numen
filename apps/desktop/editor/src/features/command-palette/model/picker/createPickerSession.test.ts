import { describe, expect, it, vi } from 'vitest'
import type { PickerItem, PickerOptions } from '@numen/plugin-loader'
import { createPickerSession } from './createPickerSession'

const TREES: readonly PickerItem[] = [
  { id: 'birch', title: 'Birch', state: 'unset' },
  { id: 'cedar', title: 'Cedar', state: 'set' },
]

const createOptions = (more: Partial<PickerOptions> = {}): PickerOptions => ({
  title: 'Trees',
  placeholder: 'Tree',
  items: TREES,
  recent: ['birch'],
  getCreateTitle: (text) => `Create ${text}`,
  onToggle: async (id) => ({ id, title: 'Birch', state: 'set' }),
  onCreate: async (text) => ({ id: text.toLowerCase(), title: text, state: 'set' }),
  ...more,
})

const COLORED: readonly PickerItem[] = [
  { id: 'birch', title: 'Birch', state: 'unset', color: 'var(--numen-tag-1)' },
  { id: 'cedar', title: 'Cedar', state: 'set', color: 'var(--numen-tag-3)' },
]

describe('a picker session', () => {
  it('has nothing to draw before a plugin opens a list', () => {
    expect(createPickerSession().multi.value).toBeUndefined()
  })

  it('draws the title, the placeholder, the items and the recent ids of the list a plugin opened', async () => {
    const session = createPickerSession()
    const opening = session.open(createOptions())

    expect(session.multi.value).toMatchObject({
      title: 'Trees',
      placeholder: 'Tree',
      items: TREES,
      recent: ['birch'],
    })
    expect(session.multi.value?.getCreateTitle('Harbour')).toBe('Create Harbour')

    session.dismiss()
    await opening
  })

  it('draws the colour of each item and the colour the plugin gives the create row', async () => {
    const session = createPickerSession()
    const opening = session.open(
      createOptions({
        items: COLORED,
        getCreateColor: (text) => `var(--numen-tag-${text.length})`,
      }),
    )

    expect(session.multi.value?.items.map((one) => one.color)).toStrictEqual([
      'var(--numen-tag-1)',
      'var(--numen-tag-3)',
    ])
    expect(session.multi.value?.getCreateColor?.('Harbour')).toBe('var(--numen-tag-7)')

    session.dismiss()
    await opening
  })

  it('draws the colour of the item a toggle or a create answers with', async () => {
    const session = createPickerSession()
    const opening = session.open(
      createOptions({
        items: COLORED,
        onToggle: async (id) => ({ id, title: 'Birch', state: 'set', color: 'var(--numen-tag-2)' }),
        onCreate: async (text) => ({
          id: text.toLowerCase(),
          title: text,
          state: 'set',
          color: 'var(--numen-tag-7)',
        }),
      }),
    )

    await session.toggle('birch')
    await session.create('Harbour')

    expect(session.multi.value?.items.map((one) => one.color)).toStrictEqual([
      'var(--numen-tag-2)',
      'var(--numen-tag-3)',
      'var(--numen-tag-7)',
    ])

    session.dismiss()
    await opening
  })

  it('draws no create colour where the plugin gives none', async () => {
    const session = createPickerSession()
    const opening = session.open(createOptions())

    expect(session.multi.value?.getCreateColor).toBeUndefined()

    session.dismiss()
    await opening
  })

  it('shows the item a toggle answers with in place of the one toggled', async () => {
    const session = createPickerSession()
    const opening = session.open(createOptions())

    await session.toggle('birch')

    expect(session.multi.value?.items.map((one) => one.state)).toStrictEqual(['set', 'set'])

    session.dismiss()
    await opening
  })

  it('adds the item a create answers with, and asks the plugin once for the text', async () => {
    const onCreate = vi.fn(async (text: string) => ({
      id: 'harbour',
      title: text,
      state: 'set' as const,
    }))
    const session = createPickerSession()
    const opening = session.open(createOptions({ onCreate }))

    await session.create('Harbour')

    expect(onCreate).toHaveBeenCalledExactlyOnceWith('Harbour')
    expect(session.multi.value?.items.map((one) => one.title)).toStrictEqual([
      'Birch',
      'Cedar',
      'Harbour',
    ])

    session.dismiss()
    await opening
  })

  it('settles the open call and draws nothing once it is dismissed', async () => {
    const session = createPickerSession()
    const closed = vi.fn()
    const opening = session.open(createOptions()).then(closed)

    await Promise.resolve()
    expect(closed).not.toHaveBeenCalled()

    session.dismiss()
    await opening
    expect(closed).toHaveBeenCalledTimes(1)
    expect(session.multi.value).toBeUndefined()
  })

  it('settles the list before it when another is opened', async () => {
    const session = createPickerSession()
    const closed = vi.fn()
    const first = session.open(createOptions()).then(closed)
    const second = session.open(createOptions({ title: 'Shrubs' }))

    await first
    expect(closed).toHaveBeenCalledTimes(1)
    expect(session.multi.value?.title).toBe('Shrubs')

    session.dismiss()
    await second
  })
})

describe('the questions of a picker session', () => {
  it('draws a question a plugin asks and settles it on dismiss', async () => {
    const session = createPickerSession()
    const asking = session.ask({ title: 'Owner' })

    expect(session.question.value?.placeholder).toBe('Owner')

    session.dismiss()
    await expect(asking).resolves.toBeUndefined()
    expect(session.question.value).toBeUndefined()
  })
})
