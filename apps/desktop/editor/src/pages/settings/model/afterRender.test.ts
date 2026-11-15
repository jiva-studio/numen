import { nextTick, ref, shallowRef } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type {
  ConnectionFieldInfo,
  ConnectionInfo,
  ConnectionTypeInfo,
  RowDraft,
  RowOption,
  StoredRow,
} from '@/entities/connection'
import type { JsonValue } from '@/shared/api/json'
import type { Result } from '@/shared/lib/result'
import { asFailure, asValue, createInstallation } from '../testing/installation'
import { runAfterRender } from './afterRender'
import { useConnectionForm } from './connectionForm'
import { useAutomaticRows } from './useAutomaticRows'
import { useConnectionEntries } from './useConnectionEntries'
import { useConnectionsPage } from './useConnectionsPage'
import { useIndexQuestion } from './useIndexQuestion'
import { useRowChoices } from './useRowChoices'
import { useRowDrafts } from './useRowDrafts'
import { useRowOrder } from './useRowOrder'

const createField = (
  key: string,
  title: string,
  overrides: Partial<ConnectionFieldInfo> = {},
): ConnectionFieldInfo => ({
  key,
  type: 'string',
  title,
  description: '',
  isRequired: false,
  isAddress: false,
  options: [],
  ...overrides,
})

const createType = (
  id: string,
  title: string,
  overrides: Partial<ConnectionTypeInfo> = {},
): ConnectionTypeInfo => ({
  id,
  pluginId: 'p',
  pluginName: 'Plugin',
  title,
  description: '',
  operations: ['llm.complete'],
  fields: [],
  rowFields: [],
  isAvailable: true,
  ...overrides,
})

const createConnection = (
  id: string,
  typeId: string,
  name: string,
  overrides: Partial<ConnectionInfo> = {},
): ConnectionInfo => ({
  id,
  typeId,
  name,
  revision: 1,
  fields: {},
  secretKeysSet: [],
  isTypeAvailable: true,
  isComplete: true,
  isDetected: false,
  path: '',
  uses: [],
  ...overrides,
})

describe('runAfterRender', () => {
  it('runs the callback after the pending state is flushed, not before', async () => {
    const ran: string[] = []
    const state = ref('first')
    runAfterRender(() => ran.push(state.value))
    state.value = 'second'
    expect(ran).toEqual([])
    await nextTick()
    await nextTick()
    expect(ran).toEqual(['second'])
  })
})

describe('useConnectionForm', () => {
  it('handles unavailable types, field updates, missing required fields, and in-flight save state', async () => {
    const unavailable = createType('t.unavail', 'Unavailable', { isAvailable: false })
    const typeA = createType('t.a', 'Type A', {
      fields: [
        createField('req1', 'First Key', { isRequired: true }),
        createField('opt2', 'Optional Key', { isRequired: false }),
        createField('req3', 'Third Key', { isRequired: true }),
      ],
    })
    const typeB = createType('t.b', 'Type B')

    const emptyForm = useConnectionForm({
      types: () => [unavailable],
      connection: () => undefined,
      create: vi.fn(() => Promise.resolve(asFailure('err'))),
      update: vi.fn(() => Promise.resolve(asFailure('err'))),
      onSaved: vi.fn(),
    })
    expect(emptyForm.state.typeId).toBe('')
    expect(emptyForm.state.problem).toBe('')
    expect(emptyForm.isEditing).toBe(false)
    expect(emptyForm.availableTypes.value).toStrictEqual([])
    expect(emptyForm.view.value).toStrictEqual({
      type: undefined,
      fields: [],
      storedKeys: [],
      isKeyLost: false,
    })
    emptyForm.actions.submit()
    await flushPromises()

    const stored = createConnection('c1', typeB.id, 'Saved B')
    const editForm = useConnectionForm({
      types: () => [unavailable, typeA, typeB],
      connection: () => stored,
      create: vi.fn(() => Promise.resolve(asFailure('err'))),
      update: vi.fn(() => Promise.resolve(asFailure('err'))),
      onSaved: vi.fn(),
    })
    expect(editForm.isEditing).toBe(true)
    expect(editForm.availableTypes.value).toStrictEqual([typeA, typeB])
    expect(editForm.view.value.type).toStrictEqual(typeB)

    let resolveCreate: ((res: Result<ConnectionInfo, string>) => void) | undefined
    const create = vi.fn(
      () =>
        new Promise<Result<ConnectionInfo, string>>((resolve) => {
          resolveCreate = resolve
        }),
    )
    const onSaved = vi.fn()
    const form = useConnectionForm({
      types: () => [unavailable, typeA, typeB],
      connection: () => undefined,
      create,
      update: vi.fn(() => Promise.resolve(asFailure('err'))),
      onSaved,
    })

    // Missing required fields filters only missing titles
    form.actions.updateValue('opt2', 'kept')
    form.actions.submit()
    await flushPromises()
    expect(form.state.problem).toBe('Required. First Key, Third Key')
    expect(create).not.toHaveBeenCalled()

    // Multiple updateValue calls preserve existing keys
    form.actions.updateValue('req1', 'v1')
    form.actions.updateValue('req3', 'v3')
    expect(form.state.values).toStrictEqual({ opt2: 'kept', req1: 'v1', req3: 'v3' })

    // Submit starts saving, clears problem, and blocks concurrent submit
    form.actions.submit()
    expect(form.state.isSaving).toBe(true)
    expect(form.state.problem).toBe('')
    form.actions.submit()
    expect(create).toHaveBeenCalledOnce()

    resolveCreate?.(asFailure('Denied'))
    await flushPromises()
    expect(form.state.isSaving).toBe(false)
    expect(form.state.problem).toBe('Denied')
    expect(onSaved).not.toHaveBeenCalled()

    // Second submit succeeds and clears problem
    form.actions.submit()
    expect(form.state.isSaving).toBe(true)
    expect(form.state.problem).toBe('')
    resolveCreate?.(asValue(stored))
    await flushPromises()
    expect(form.state.isSaving).toBe(false)
    expect(form.state.problem).toBe('')
    expect(onSaved).toHaveBeenCalledExactlyOnceWith(stored)
  })
})

describe('useAutomaticRows', () => {
  it('computes isAutomatic and triggers add when stored, drafts, or isSingle change', async () => {
    const conn = createConnection('c1', 't1', 'Work')
    const stored = ref<readonly ConnectionInfo[]>([])
    const discovered = ref<readonly ConnectionInfo[]>([conn])
    const isSingle = ref(false)
    const drafts = ref<readonly RowDraft[]>([])
    const isDiscovering = ref(true)
    const add = vi.fn()

    const auto = useAutomaticRows({
      isDiscovering: () => isDiscovering.value,
      getValue: () => [],
      stored,
      discovered,
      isSingle,
      drafts,
      add,
    })

    // No stored connections -> isAutomatic is false and add is not called
    expect(auto.isAutomatic.value).toBe(false)
    expect(add).not.toHaveBeenCalled()

    isDiscovering.value = false
    isSingle.value = true
    await nextTick()
    expect(add).not.toHaveBeenCalled()

    // Stored connection arrives -> triggers add
    stored.value = [conn]
    await nextTick()
    expect(add).toHaveBeenCalledTimes(1)

    // Draft added then emptied -> triggers add again
    drafts.value = [{ key: 'row-1', connection: 'c1', options: {} }]
    await nextTick()
    drafts.value = []
    await nextTick()
    expect(add).toHaveBeenCalledTimes(2)

    // isSingle toggled false then true -> triggers add again
    isSingle.value = false
    await nextTick()
    isSingle.value = true
    await nextTick()
    expect(add).toHaveBeenCalledTimes(3)
  })
})

describe('useConnectionEntries, useConnectionsPage, and useIndexQuestion', () => {
  it('resolves choice details, reasons, and rowFields per connection', () => {
    const fieldA = createField('modelA', 'Model A')
    const fieldB = createField('modelB', 'Model B')
    const typeA = createType('t.a', 'Title A', { rowFields: [fieldA] })
    const typeB = createType('t.b', 'Title B', { rowFields: [fieldB] })
    const typeEmbed = createType('t.embed', 'Embedder', { operations: ['llm.embed'] })

    const connA = createConnection('c.a', typeA.id, 'Conn A')
    const connB = createConnection('c.b', typeB.id, 'Conn B')
    const connEmbed = createConnection('c.embed', typeEmbed.id, 'Conn Embed')
    const connRetired = createConnection('c.retired', 't.missing', 'Conn Retired', {
      isTypeAvailable: false,
    })

    const installation = createInstallation({
      list: ref([connA, connB, connEmbed, connRetired]),
      types: ref([typeA, typeB, typeEmbed]),
      operationTitles: ref({ 'llm.complete': 'complete text' }),
    })

    const entries = useConnectionEntries(
      () => installation.connections,
      () => ['llm.complete'],
    )

    expect(entries.choices.value).toStrictEqual([
      { id: 'c.a', text: 'Conn A', detail: 'Title A' },
      { id: 'c.b', text: 'Conn B', detail: 'Title B' },
      {
        id: 'c.embed',
        text: 'Conn Embed',
        detail: 'This type cannot complete text',
        isDisabled: true,
      },
      {
        id: 'c.retired',
        text: 'Conn Retired',
        detail: 'Type unavailable',
        isDisabled: true,
      },
    ])
    expect(entries.getReason('c.a')).toBe('')
    expect(entries.getReason('c.retired')).toBe('Type unavailable')
    expect(entries.getReason('unknown')).toBe('')
    expect(entries.getRowFields('c.a')).toStrictEqual([fieldA])
    expect(entries.getRowFields('c.b')).toStrictEqual([fieldB])
    expect(entries.getRowFields('c.retired')).toStrictEqual([])
    expect(entries.getRowFields('unknown')).toStrictEqual([])
  })

  it('formats deletion prompts with and without keys, uses, and problems', async () => {
    const plain = createConnection('c1', 't1', 'Plain', { secretKeysSet: [], uses: [] })
    const withKey = createConnection('c2', 't1', 'Keyed', { secretKeysSet: ['apiKey'], uses: [] })
    const deleteFn = vi.fn<() => Promise<Result<undefined, string>>>(() =>
      Promise.resolve(asFailure('In use')),
    )
    const installation = createInstallation({
      list: ref([plain, withKey]),
      delete: deleteFn,
    })

    const page = useConnectionsPage(() => installation.connections)
    expect(page.deletion.text.value).toBe('')

    const onDeleted = vi.fn()
    page.deletion.confirm(onDeleted)
    await flushPromises()
    expect(deleteFn).not.toHaveBeenCalled()
    expect(onDeleted).not.toHaveBeenCalled()

    page.deletion.ask(plain)
    expect(page.deletion.text.value).toBe('Delete "Plain"?')

    page.deletion.confirm(onDeleted)
    await flushPromises()
    expect(onDeleted).not.toHaveBeenCalled()
    expect(page.deletion.text.value).toBe(
      'Delete "Plain"? The connection was not deleted: In use',
    )

    // Asking again clears the previous problem
    page.deletion.ask(withKey)
    expect(page.deletion.text.value).toBe(
      'Delete "Keyed"? Its key is removed from this machine.',
    )
  })

  it('ignores confirm when no index question is open', () => {
    const onConfirm = vi.fn()
    const indexQuestion = useIndexQuestion({
      getConnections: () => createInstallation().connections,
      getText: () => 'question',
      onConfirm,
      onCancel: vi.fn(),
    })

    indexQuestion.confirm()
    expect(onConfirm).not.toHaveBeenCalled()
  })
})

describe('useRowChoices', () => {
  it('formats choice details and groups, tracks waiting and disabled state, and drops stale option responses', async () => {
    const modelField = createField('model', 'Model', {
      type: 'string',
      isRequired: true,
      optionsFrom: 'llm.models',
    })
    const reasoningField = createField('reasoning', 'Reasoning', {
      type: 'enum',
      isRequired: false,
      optionsFrom: 'llm.reasonings',
    })
    const noteField = createField('note', 'Note', {
      type: 'string',
      isRequired: false,
      options: [
        { value: 'plain', title: '' },
        { value: 'same', title: 'same' },
        { value: 'titled', title: 'Titled' },
      ],
    })

    const row = ref<RowDraft>({ key: 'row-1', connection: '', options: {} })
    const resolvers = new Map<string, ((res: Result<readonly RowOption[], string>) => void)[]>()
    const listRowOptions = vi.fn(
      (_conn: string, field: string) =>
        new Promise<Result<readonly RowOption[], string>>((resolve) => {
          const list = resolvers.get(field) ?? []
          list.push(resolve)
          resolvers.set(field, list)
        }),
    )

    const hasList = ref(true)
    const { fields } = useRowChoices({
      getRow: () => row.value,
      getRowFields: () => [modelField, reasoningField, noteField],
      getOperations: () => ['llm.complete'],
      getListRowOptions: () => (hasList.value ? listRowOptions : undefined),
    })

    // While connection is empty, all fields are waiting and disabled, and no request was sent
    expect(listRowOptions).not.toHaveBeenCalled()
    expect(fields.value.map((f) => f.isDisabled)).toStrictEqual([true, true, true])
    expect(fields.value[0]?.choices).toStrictEqual([])
    expect(fields.value[2]?.choices).toStrictEqual([
      { id: '', text: 'Default' },
      { id: 'plain', text: 'plain' },
      { id: 'same', text: 'same' },
      { id: 'titled', text: 'Titled', detail: 'titled' },
    ])

    // Choose connection c1 while model is empty: model asks, reasoning still waits
    row.value = { key: 'row-1', connection: 'c1', options: {} }
    await nextTick()
    expect(listRowOptions).toHaveBeenCalledTimes(1)
    expect(listRowOptions).toHaveBeenLastCalledWith('c1', 'model', {}, ['llm.complete'])
    expect(fields.value[0]?.isDisabled).toBe(false)
    expect(fields.value[1]?.isDisabled).toBe(true)

    // Switch connection to c2 before c1 resolves -> c1 answer is dropped when it arrives later
    row.value = { key: 'row-1', connection: 'c2', options: {} }
    await nextTick()
    expect(listRowOptions).toHaveBeenCalledTimes(2)

    resolvers.get('model')?.[1]?.(
      asValue([
        { value: 'm1', title: '', group: '' },
        { value: 'm2', title: 'm2', group: '' },
        { value: 'm3', title: 'Model 3', group: 'Fast' },
      ]),
    )
    resolvers.get('model')?.[0]?.(asValue([{ value: 'stale', title: 'Stale', group: '' }]))
    await flushPromises()

    expect(fields.value[0]?.choices).toStrictEqual([
      { id: 'm1', text: 'm1' },
      { id: 'm2', text: 'm2' },
      { id: 'm3', text: 'Model 3', detail: 'm3', group: 'Fast' },
    ])

    // Setting model value triggers ask for reasoning (and not for model again)
    row.value = { key: 'row-1', connection: 'c2', options: { model: 'm3' } }
    await nextTick()
    expect(listRowOptions).toHaveBeenCalledTimes(3)
    expect(listRowOptions).toHaveBeenLastCalledWith(
      'c2',
      'reasoning',
      { model: 'm3' },
      ['llm.complete'],
    )
    // Optional enum field with only Default choice (length 1) is disabled, while optional string field is enabled
    expect(fields.value[1]?.choices).toStrictEqual([{ id: '', text: 'Default' }])
    expect(fields.value[1]?.isDisabled).toBe(true)
    expect(fields.value[2]?.isDisabled).toBe(false)

    resolvers.get('reasoning')?.[0]?.(asValue([{ value: 'high', title: 'High', group: '' }]))
    await flushPromises()
    expect(fields.value[1]?.choices).toStrictEqual([
      { id: '', text: 'Default' },
      { id: 'high', text: 'High', detail: 'high' },
    ])
    expect(fields.value[1]?.isDisabled).toBe(false)

    // Changing note (last field) does not re-ask model or reasoning
    row.value = { key: 'row-1', connection: 'c2', options: { model: 'm3', note: 'plain' } }
    await nextTick()
    expect(listRowOptions).toHaveBeenCalledTimes(3)

    // Changing model clears reasoning's listed options while keeping model's listed options
    row.value = { key: 'row-1', connection: 'c2', options: { model: 'm1', note: 'plain' } }
    await nextTick()
    expect(fields.value[0]?.choices).toHaveLength(3)
    expect(fields.value[1]?.choices).toStrictEqual([{ id: '', text: 'Default' }])

    // Failure response sets empty options and problem message
    resolvers.get('reasoning')?.[1]?.(asFailure('offline'))
    await flushPromises()
    expect(fields.value[1]?.choices).toStrictEqual([{ id: '', text: 'Default' }])
    expect(fields.value[1]?.problem).toBe('The values of Reasoning could not be listed: offline')

    // Missing listRowOptions returns cleanly
    hasList.value = false
    row.value = { key: 'row-1', connection: 'c3', options: {} }
    await nextTick()
    expect(listRowOptions).toHaveBeenCalledTimes(4)
  })
})

describe('useRowDrafts', () => {
  it('assigns incrementing keys, tracks in-flight writes, and preserves incomplete drafts', async () => {
    const modelField = createField('model', 'Model', { isRequired: true })
    const value = shallowRef<JsonValue>([
      { connection: 'c1', model: 'a' },
      { connection: 'c2', model: 'b' },
    ])
    const onChange = vi.fn<(rows: readonly StoredRow[]) => void>()

    const rowDrafts = useRowDrafts({
      getValue: () => value.value,
      getRowFields: () => [modelField],
      onChange,
    })

    expect(rowDrafts.drafts.value.map((d) => d.key)).toStrictEqual(['row-1', 'row-2'])

    // Writing when complete rows already match stored value and nothing is in flight is a no-op
    rowDrafts.write()
    expect(onChange).not.toHaveBeenCalled()

    // Add an incomplete draft row (no connection)
    rowDrafts.add()
    expect(rowDrafts.drafts.value.map((d) => d.key)).toStrictEqual(['row-1', 'row-2', 'row-3'])

    // Two sequential updates put two entries in inFlight
    rowDrafts.update('row-1', { options: { model: 'a2' } })
    rowDrafts.update('row-1', { options: { model: 'a3' } })
    expect(onChange).toHaveBeenCalledTimes(2)

    // First in-flight value arrives (at === 0): ignored, keeps 'a3' and incomplete 'row-3'
    value.value = [
      { connection: 'c1', model: 'a2' },
      { connection: 'c2', model: 'b' },
    ]
    await nextTick()
    expect(rowDrafts.drafts.value).toHaveLength(3)
    expect(rowDrafts.drafts.value[0]?.options['model']).toBe('a3')

    // Second in-flight value arrives: inFlight is now empty
    value.value = [
      { connection: 'c1', model: 'a3' },
      { connection: 'c2', model: 'b' },
    ]
    await nextTick()
    expect(rowDrafts.drafts.value).toHaveLength(3)

    // Same stored value emitted again when inFlight is empty: does not clobber incomplete 'row-3'
    value.value = [
      { connection: 'c1', model: 'a3' },
      { connection: 'c2', model: 'b' },
    ]
    await nextTick()
    expect(rowDrafts.drafts.value).toHaveLength(3)

    // External value arrival clears any stale inFlight entries
    rowDrafts.update('row-1', { options: { model: 'a4' } })
    value.value = [{ connection: 'c2', model: 'external' }]
    await nextTick()
    expect(rowDrafts.drafts.value).toHaveLength(1)
    rowDrafts.add()
    // Now if 'a4' arrives later, it is not treated as in-flight because inFlight was cleared
    value.value = [
      { connection: 'c1', model: 'a4' },
      { connection: 'c2', model: 'b' },
    ]
    await nextTick()
    expect(rowDrafts.drafts.value).toHaveLength(2)

    // revert() also clears inFlight
    rowDrafts.update('row-4', { options: { model: 'a5' } })
    rowDrafts.revert()
    rowDrafts.add()
    value.value = [
      { connection: 'c1', model: 'a5' },
      { connection: 'c2', model: 'b' },
    ]
    await nextTick()
    expect(rowDrafts.drafts.value).toHaveLength(2)
    expect(rowDrafts.drafts.value[0]?.options['model']).toBe('a5')
  })
})

describe('useRowOrder', () => {
  it('reorders rows via pointer drag and keyboard with announcements and focus restoration', async () => {
    const drafts = ref<readonly RowDraft[]>([
      { key: 'row-1', connection: 'c1', options: {} },
      { key: 'row-2', connection: 'c2', options: {} },
      { key: 'row-3', connection: 'c3', options: {} },
    ])
    const listEl = document.createElement('ul')
    const item0 = document.createElement('li')
    const grip0 = document.createElement('button')
    grip0.setAttribute('data-grip', 'row-1')
    item0.append(grip0)
    const item1 = document.createElement('li')
    const grip1 = document.createElement('button')
    grip1.setAttribute('data-grip', 'row-2')
    item1.append(grip1)
    const item2 = document.createElement('li')
    const grip2 = document.createElement('button')
    grip2.setAttribute('data-grip', 'row-3')
    item2.append(grip2)
    listEl.append(item0, item1, item2)
    document.body.append(listEl)

    item0.getBoundingClientRect = () => new DOMRect(0, 0, 100, 30)
    item1.getBoundingClientRect = () => new DOMRect(0, 30, 100, 30)
    item2.getBoundingClientRect = () => new DOMRect(0, 60, 100, 30)

    let currentList: HTMLElement | null = listEl
    const write = vi.fn()
    const move = vi.fn()

    const order = useRowOrder({
      drafts,
      getList: () => currentList,
      write,
      move,
    })

    expect(order.announcement.value).toBe('')

    // Pointer drag: below threshold (delta = 4) does not move; past threshold (delta = 35) commits move(0, 1)
    order.onGripPress(0, new PointerEvent('pointerdown', { button: 0, clientX: 0, clientY: 10 }))
    window.dispatchEvent(new PointerEvent('pointermove', { clientX: 0, clientY: 14 }))
    expect(order.drag.dragging.value?.hasMoved).toBe(false)
    window.dispatchEvent(new PointerEvent('pointermove', { clientX: 0, clientY: 45 }))
    expect(order.drag.dragging.value?.hasMoved).toBe(true)
    expect(order.drag.at.value).toBe(1)
    window.dispatchEvent(new PointerEvent('pointerup', { clientX: 0, clientY: 45 }))
    expect(move).toHaveBeenCalledExactlyOnceWith(0, 1)

    // Keyboard: Escape or ArrowUp when not held does nothing
    const unheldEsc = new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    })
    await order.onGripKey('row-2', unheldEsc)
    expect(unheldEsc.defaultPrevented).toBe(false)

    // Pick up row-2 (index 1) with Space
    const spacePick = new KeyboardEvent('keydown', { key: ' ', bubbles: true, cancelable: true })
    await order.onGripKey('row-2', spacePick)
    expect(spacePick.defaultPrevented).toBe(true)
    expect(order.announcement.value).toBe(
      'Picked up row 2 of 3. Use the arrow keys to move it and Space to drop it.',
    )

    // Move up to index 0
    const upToZero = new KeyboardEvent('keydown', {
      key: 'ArrowUp',
      bubbles: true,
      cancelable: true,
    })
    await order.onGripKey('row-2', upToZero)
    expect(upToZero.defaultPrevented).toBe(true)
    expect(order.announcement.value).toBe('Moved to position 1 of 3.')
    expect(document.activeElement).toBe(grip1)

    // Move up past index 0 (target = -1) is ignored
    await order.onGripKey(
      'row-2',
      new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true, cancelable: true }),
    )
    expect(order.announcement.value).toBe('Moved to position 1 of 3.')

    // Move down to index 1 and index 2, then past index 2 (target = 3)
    await order.onGripKey(
      'row-2',
      new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true }),
    )
    expect(order.announcement.value).toBe('Moved to position 2 of 3.')
    await order.onGripKey(
      'row-2',
      new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true }),
    )
    expect(order.announcement.value).toBe('Moved to position 3 of 3.')
    await order.onGripKey(
      'row-2',
      new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true }),
    )
    expect(order.announcement.value).toBe('Moved to position 3 of 3.')

    // Cancel with Escape while list element has no matching grip and when list is null
    grip1.removeAttribute('data-grip')
    const escHeld = new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    })
    const stopSpy = vi.spyOn(escHeld, 'stopPropagation')
    await order.onGripKey('row-2', escHeld)
    expect(escHeld.defaultPrevented).toBe(true)
    expect(stopSpy).toHaveBeenCalledOnce()
    expect(order.announcement.value).toBe('Move cancelled.')
    expect(drafts.value.map((d) => d.key)).toStrictEqual(['row-1', 'row-2', 'row-3'])

    currentList = null
    await order.onGripKey('row-1', new KeyboardEvent('keydown', { key: ' ', cancelable: true }))
    await order.onGripKey(
      'row-1',
      new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }),
    )

    listEl.remove()
  })
})
