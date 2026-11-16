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

describe('a picker session', () => {
  it('has nothing to draw before a plugin opens a list', () => {
    expect(createPickerSession().multi.value).toBeUndefined()
  })

  it('draws the title, the placeholder, the items and the recent ids of the list a plugin opened', () => {
    const session = createPickerSession()
    void session.host.open(createOptions())

    expect(session.multi.value).toMatchObject({
      title: 'Trees',
      placeholder: 'Tree',
      items: TREES,
      recent: ['birch'],
    })
    expect(session.multi.value?.getCreateTitle('Harbour')).toBe('Create Harbour')
  })

  it('shows the item a toggle answers with in place of the one toggled', async () => {
    const session = createPickerSession()
    void session.host.open(createOptions())

    await session.toggle('birch')

    expect(session.multi.value?.items.map((one) => one.state)).toStrictEqual(['set', 'set'])
  })

  it('adds the item a create answers with, and asks the plugin once for the text', async () => {
    const onCreate = vi.fn(async (text: string) => ({
      id: 'harbour',
      title: text,
      state: 'set' as const,
    }))
    const session = createPickerSession()
    void session.host.open(createOptions({ onCreate }))

    await session.create('Harbour')

    expect(onCreate).toHaveBeenCalledExactlyOnceWith('Harbour')
    expect(session.multi.value?.items.map((one) => one.title)).toStrictEqual([
      'Birch',
      'Cedar',
      'Harbour',
    ])
  })

  it('settles the open call and draws nothing once it is dismissed', async () => {
    const session = createPickerSession()
    const closed = vi.fn()
    const opening = session.host.open(createOptions()).then(closed)

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
    const first = session.host.open(createOptions()).then(closed)

    void session.host.open(createOptions({ title: 'Shrubs' }))

    await first
    expect(closed).toHaveBeenCalledTimes(1)
    expect(session.multi.value?.title).toBe('Shrubs')
  })
})
