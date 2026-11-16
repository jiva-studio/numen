/**
 * The command Set tags as the person meets it: the list the picker opens with, and what each toggle and
 * each new tag writes. The backend is the JSON the Go messages encode to, read from testdata/api.
 */
import { describe, expect, it, vi } from 'vitest'
import { createApi, type CommandNote } from '@numen/plugin-sdk'
import { createMockPickerHost, type MockPickerHost } from '@numen/plugin-sdk/testing'
import { createSetTags } from './createSetTags'

const SERVICE = 'md.numen.tools.taxonomy'

const QUAY = 'notes/Quay.md'
const DOCK = 'notes/Dock.md'
const PIER = 'notes/Pier.md'

interface TagList {
  readonly tags: readonly { readonly tag: string; readonly count: number }[]
  readonly recent: readonly string[]
}

interface NoteTags {
  readonly path: string
  readonly tags: readonly string[]
}

interface Sent {
  readonly method: string
  readonly body: unknown
}

const fixtures = import.meta.glob('../../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const readFixture = <T>(name: string): T => fixtures[`../../../testdata/api/${name}.json`] as T

const TAGS_BY_PATH: Readonly<Record<string, readonly string[]>> = {
  [QUAY]: ['Port', 'Sea'],
  [DOCK]: ['Port', 'Sea', 'Quay'],
}

const createNote = (path: string, others: readonly string[] = []): CommandNote => ({
  path,
  title: path,
  type: 'note',
  isFolder: false,
  others,
  tab: '',
})

const answerJson = (value: unknown): Response =>
  new Response(JSON.stringify(value), { headers: { 'Content-Type': 'application/json' } })

interface Opened {
  readonly picker: MockPickerHost
  readonly sent: Sent[]
  readonly closed: Promise<void>
  readonly getState: (id: string) => string | undefined
}

interface Start {
  readonly note: CommandNote
  readonly list?: TagList
  readonly writes?: readonly (readonly NoteTags[])[]
}

const start = async ({ note, list, writes = [] }: Start): Promise<Opened> => {
  const sent: Sent[] = []
  const answers = [...writes]
  const api = createApi(SERVICE, {
    origin: 'http://window',
    fetch: async (url, init) => {
      const method = String(url).split('/').pop() ?? ''
      sent.push({ method, body: JSON.parse(String(init?.body)) })
      if (method === 'GetTags') return answerJson(list ?? readFixture<TagList>('GetTags.response'))
      return answerJson({ notes: answers.shift() ?? [] })
    },
  })
  const picker = createMockPickerHost()
  const command = createSetTags({ api, picker, getTags: (path) => TAGS_BY_PATH[path] ?? [] })
  const closed = Promise.resolve(command.handler(note))
  await vi.waitFor(() => expect(picker.getOpened()).toBeDefined())
  return {
    picker,
    sent,
    closed,
    getState: (id) => picker.getItems().find((one) => one.id === id)?.state,
  }
}

const close = async (opened: Opened): Promise<void> => {
  opened.picker.close()
  await opened.closed
}

const getWrites = (opened: Opened): Sent[] => opened.sent.filter((one) => one.method === 'SetTags')

describe('the command Set tags', () => {
  it('is the command the manifest declares', () => {
    const command = createSetTags({
      api: createApi(SERVICE, { origin: 'http://window', fetch: async () => answerJson({}) }),
      picker: createMockPickerHost(),
      getTags: () => [],
    })

    expect(command.id).toBe('tags.set')
  })

  it('opens the list of the tags with a field that asks for a tag', async () => {
    const opened = await start({ note: createNote(PIER) })

    expect(opened.picker.getOpened()?.title).toBe('Tags')
    expect(opened.picker.getOpened()?.placeholder).toBe('Tag')
    await close(opened)
  })

  it('lists every tag of the vault once', async () => {
    const opened = await start({ note: createNote(PIER) })

    expect(
      opened.picker
        .getItems()
        .map((one) => one.title)
        .sort(),
    ).toStrictEqual(['Port', 'Quay', 'Sea'])
    expect(opened.sent.filter((one) => one.method === 'GetTags')).toHaveLength(1)
    await close(opened)
  })

  it('shows the tags of the selected file as set and the other tags as unset', async () => {
    const opened = await start({ note: createNote(QUAY) })

    expect([
      opened.getState('Port'),
      opened.getState('Sea'),
      opened.getState('Quay'),
    ]).toStrictEqual(['set', 'set', 'unset'])
    await close(opened)
  })

  it('shows every tag as unset for a file that has none', async () => {
    const opened = await start({ note: createNote(PIER) })

    expect(opened.picker.getItems().map((one) => one.state)).toStrictEqual([
      'unset',
      'unset',
      'unset',
    ])
    await close(opened)
  })

  it('shows a tag that only some of the selected files have as mixed', async () => {
    const opened = await start({ note: createNote(QUAY, [PIER, DOCK]) })

    expect([
      opened.getState('Port'),
      opened.getState('Sea'),
      opened.getState('Quay'),
    ]).toStrictEqual(['mixed', 'mixed', 'mixed'])
    await close(opened)
  })

  it('shows a tag that every selected file has as set and one that only one has as mixed', async () => {
    const opened = await start({ note: createNote(QUAY, [DOCK]) })

    expect([
      opened.getState('Port'),
      opened.getState('Sea'),
      opened.getState('Quay'),
    ]).toStrictEqual(['set', 'set', 'mixed'])
    await close(opened)
  })

  it('offers the tags the backend gives as recent, the latest first', async () => {
    const opened = await start({
      note: createNote(PIER),
      list: { ...readFixture<TagList>('GetTags.response'), recent: ['Sea', 'Port'] },
    })

    expect(opened.picker.getOpened()?.recent).toStrictEqual(['Sea', 'Port'])
    await close(opened)
  })

  it('names the row that makes a tag from the text typed', async () => {
    const opened = await start({ note: createNote(PIER) })

    expect(opened.picker.getCreateTitle('Harbour')).toBe('Create Harbour')
    await close(opened)
  })

  it('stays open until the person closes the list', async () => {
    let isDone = false
    const opened = await start({ note: createNote(PIER) })
    const done = opened.closed.then(() => {
      isDone = true
    })
    await Promise.resolve()

    expect(isDone).toBe(false)
    opened.picker.close()
    await done
    expect(isDone).toBe(true)
  })
})

describe('a tag toggled in the list', () => {
  it('is written to the file at once and shows as set', async () => {
    const opened = await start({
      note: createNote(PIER),
      writes: [[{ path: PIER, tags: ['Port'] }]],
    })

    await opened.picker.toggle('Port')

    expect(getWrites(opened).map((one) => one.body)).toStrictEqual([
      { paths: [PIER], toggles: ['Port'] },
    ])
    expect(opened.getState('Port')).toBe('set')
    await close(opened)
  })

  it('is cleared from the file when the file had it and shows as unset, the others stay', async () => {
    const opened = await start({
      note: createNote(DOCK),
      writes: [[{ path: DOCK, tags: ['Port', 'Quay'] }]],
    })

    await opened.picker.toggle('Sea')

    expect(getWrites(opened).map((one) => one.body)).toStrictEqual([
      { paths: [DOCK], toggles: ['Sea'] },
    ])
    expect([
      opened.getState('Port'),
      opened.getState('Sea'),
      opened.getState('Quay'),
    ]).toStrictEqual(['set', 'unset', 'set'])
    await close(opened)
  })

  it('is set on every selected file when it was mixed', async () => {
    const opened = await start({
      note: createNote(QUAY, [PIER]),
      writes: [
        [
          { path: QUAY, tags: ['Port', 'Sea'] },
          { path: PIER, tags: ['Port'] },
        ],
      ],
    })

    await opened.picker.toggle('Port')

    expect(getWrites(opened).map((one) => one.body)).toStrictEqual([
      { paths: [QUAY, PIER], toggles: ['Port'] },
    ])
    expect(opened.getState('Port')).toBe('set')
    await close(opened)
  })

  it('is cleared from every selected file when all had it', async () => {
    const opened = await start({
      note: createNote(QUAY, [DOCK]),
      writes: [
        [
          { path: QUAY, tags: ['Sea'] },
          { path: DOCK, tags: ['Sea', 'Quay'] },
        ],
      ],
    })

    await opened.picker.toggle('Port')

    expect(opened.getState('Port')).toBe('unset')
    await close(opened)
  })

  it('is written again for each toggle of the same tag in the open list', async () => {
    const opened = await start({
      note: createNote(PIER),
      writes: [[{ path: PIER, tags: ['Port'] }], [{ path: PIER, tags: [] }]],
    })

    await opened.picker.toggle('Port')
    const afterFirst = opened.getState('Port')
    await opened.picker.toggle('Port')

    expect([afterFirst, opened.getState('Port')]).toStrictEqual(['set', 'unset'])
    expect(getWrites(opened)).toHaveLength(2)
    await close(opened)
  })

  it('sets one tag and clears another in the same open list', async () => {
    const opened = await start({
      note: createNote(QUAY),
      writes: [[{ path: QUAY, tags: ['Sea', 'Quay'] }], [{ path: QUAY, tags: ['Quay'] }]],
    })

    await opened.picker.toggle('Port')
    await opened.picker.toggle('Sea')

    expect(getWrites(opened).map((one) => one.body)).toStrictEqual([
      { paths: [QUAY], toggles: ['Port'] },
      { paths: [QUAY], toggles: ['Sea'] },
    ])
    expect([opened.getState('Port'), opened.getState('Sea')]).toStrictEqual(['unset', 'unset'])
    await close(opened)
  })
})

describe('a tag made from the typed text', () => {
  it('is written to the file at once and shows as set', async () => {
    const opened = await start({
      note: createNote(PIER),
      writes: [[{ path: PIER, tags: ['Harbour'] }]],
    })

    await opened.picker.create('Harbour')

    expect(getWrites(opened).map((one) => one.body)).toStrictEqual([
      { paths: [PIER], toggles: ['Harbour'] },
    ])
    expect(opened.picker.getItems()).toContainEqual({
      id: 'Harbour',
      title: 'Harbour',
      state: 'set',
    })
    await close(opened)
  })

  it('is written to every selected file', async () => {
    const opened = await start({
      note: createNote(PIER, [QUAY]),
      writes: [
        [
          { path: PIER, tags: ['Harbour'] },
          { path: QUAY, tags: ['Port', 'Sea', 'Harbour'] },
        ],
      ],
    })

    await opened.picker.create('Harbour')

    expect(getWrites(opened).map((one) => one.body)).toStrictEqual([
      { paths: [PIER, QUAY], toggles: ['Harbour'] },
    ])
    await close(opened)
  })
})
