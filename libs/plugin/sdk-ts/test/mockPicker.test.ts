import { describe, expect, it, vi } from 'vitest'
import * as testing from '../src/testing'
import type { PickerItem, PickerOptions } from '../src/picker'
import type { MockPickerHost } from '../src/testing/mockPicker'

const TREES: readonly PickerItem[] = [
  { id: 'birch', title: 'Birch', state: 'unset' },
  { id: 'cedar', title: 'Cedar', state: 'set' },
]

const createOptions = (more: Partial<PickerOptions> = {}): PickerOptions => ({
  title: 'Trees',
  placeholder: 'Tree',
  items: TREES,
  recent: [],
  getCreateTitle: (text) => `Create ${text}`,
  onToggle: async (id) => {
    const item = TREES.find((one) => one.id === id)!
    return { ...item, state: item.state === 'set' ? 'unset' : 'set' }
  },
  onCreate: async (text) => ({ id: text.toLowerCase(), title: text, state: 'set' }),
  ...more,
})

const createPicker = (): MockPickerHost => {
  const create = (testing as Record<string, unknown>)['createMockPickerHost']
  expect(create).toBeTypeOf('function')
  return (create as () => MockPickerHost)()
}

describe('the mock picker host', () => {
  it('keeps the options of the list a plugin opened', async () => {
    const picker = createPicker()
    const options = createOptions()
    const opening = picker.open(options)

    expect(picker.getOpened()).toBe(options)
    picker.close()
    await opening
  })

  it('holds the open call until the person closes the list', async () => {
    const picker = createPicker()
    const closed = vi.fn()
    const opening = picker.open(createOptions()).then(closed)

    await Promise.resolve()
    expect(closed).not.toHaveBeenCalled()

    picker.close()
    await opening
    expect(closed).toHaveBeenCalledTimes(1)
  })

  it('lists the items as the plugin gave them before anything is toggled', async () => {
    const picker = createPicker()
    const opening = picker.open(createOptions())

    expect(picker.getItems()).toStrictEqual(TREES)
    picker.close()
    await opening
  })

  it('asks the plugin to toggle an item and shows the state the plugin answers with', async () => {
    const picker = createPicker()
    const onToggle = vi.fn(async (id: string) => ({ id, title: 'Birch', state: 'mixed' as const }))
    const opening = picker.open(createOptions({ onToggle }))

    await picker.toggle('birch')

    expect(onToggle).toHaveBeenCalledExactlyOnceWith('birch')
    expect(picker.getItems().find((one) => one.id === 'birch')?.state).toBe('mixed')
    picker.close()
    await opening
  })

  it('keeps the list open after a toggle', async () => {
    const picker = createPicker()
    const closed = vi.fn()
    const opening = picker.open(createOptions()).then(closed)

    await picker.toggle('cedar')
    await Promise.resolve()

    expect(closed).not.toHaveBeenCalled()
    picker.close()
    await opening
  })

  it('asks the plugin to create an item from typed text and adds the item it answers with', async () => {
    const picker = createPicker()
    const onCreate = vi.fn(async (text: string) => ({
      id: 'harbour',
      title: text,
      state: 'set' as const,
    }))
    const opening = picker.open(createOptions({ onCreate }))

    await picker.create('Harbour')

    expect(onCreate).toHaveBeenCalledExactlyOnceWith('Harbour')
    expect(picker.getItems().map((one) => one.title)).toStrictEqual(['Birch', 'Cedar', 'Harbour'])
    picker.close()
    await opening
  })

  it('names the create row with the words the plugin gave', async () => {
    const picker = createPicker()
    const opening = picker.open(createOptions({ getCreateTitle: (text) => `Make ${text}` }))

    expect(picker.getCreateTitle('Harbour')).toBe('Make Harbour')
    picker.close()
    await opening
  })
})

describe('the mock host services', () => {
  it('offer the picker a test gave', () => {
    const picker = createPicker()
    const services = testing.createMockHostServices({ picker })

    expect(services.ui.picker).toBe(picker)
  })

  it('leave the picker out where a test gave none', () => {
    const services = testing.createMockHostServices()

    expect(services.ui.picker).toBeUndefined()
  })
})
