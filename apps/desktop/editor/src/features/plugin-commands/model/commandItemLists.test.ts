/**
 * The item lists of the commands the core lists them for: read in one request for the commands a
 * menu asks for at once, over one read of the kind of the file, and read again for every menu. Every
 * path is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import type { FileKind } from '@/shared/model/fileTypes'
import type { ItemListRequest } from '@/entities/plugin'
import { createCommandItemLists } from './commandItemLists'

const DELETE_TEXT = { pluginId: '', id: 'app:version.delete' }
const PROOFREAD = { pluginId: 'org.example.ink', id: 'ink.proofread' }
const OUTLINE = { pluginId: 'org.example.ink', id: 'ink.outline' }
const OTHER_PROOFREAD = { pluginId: 'org.example.quill', id: 'ink.proofread' }
const SCAN = 'scans/page.png'
const ITEMS = [{ id: 'org.example.ears/ears', title: 'transcript (live)' }]
const PROOFREAD_ITEMS = [{ id: 'ink-1', title: 'proofread' }]
const OUTLINE_ITEMS = [{ id: 'ink-2', title: 'outline' }]
const QUILL_ITEMS = [{ id: 'quill-1', title: 'quill proofread' }]

const getItemsFor = (one: ItemListRequest) => {
  if (one.pluginId === OUTLINE.pluginId && one.command === OUTLINE.id) return OUTLINE_ITEMS
  if (one.pluginId === OTHER_PROOFREAD.pluginId && one.command === OTHER_PROOFREAD.id) {
    return QUILL_ITEMS
  }
  if (one.pluginId === PROOFREAD.pluginId && one.command === PROOFREAD.id) return PROOFREAD_ITEMS
  return ITEMS
}

const setUp = (kind: () => Promise<FileKind | undefined> = async () => undefined) => {
  const requests: { path: string; commands: readonly ItemListRequest[] }[] = []
  const kindReads: (readonly string[])[] = []
  const lists = createCommandItemLists({
    listLists: vi.fn(async (path: string, commands: readonly ItemListRequest[]) => {
      requests.push({ path, commands })
      return commands
        .filter((one) => one.command !== 'ink.missing')
        .map((one) =>
          one.command === 'ink.gone'
            ? {
                ...one,
                result: { ok: false as const, error: 'no enabled plugin declares that command' },
              }
            : { ...one, result: { ok: true as const, value: getItemsFor(one) } },
        )
    }),
    readFileKinds: async (paths) => {
      kindReads.push(paths)
      const one = await kind()
      const firstPath = paths[0]
      return new Map(one && firstPath !== undefined ? [[firstPath, one]] : [])
    },
  })
  return { lists, requests, kindReads }
}

describe('the item lists of the commands of one menu', () => {
  it('lists the commands requested together over a file in one request', async () => {
    const { lists, requests } = setUp()

    const answers = Promise.all([
      lists.listItems(DELETE_TEXT, SCAN),
      lists.listItems(PROOFREAD, SCAN),
      lists.listItems(OUTLINE, SCAN),
      lists.listItems(OTHER_PROOFREAD, SCAN),
    ])

    expect(await answers).toStrictEqual([ITEMS, PROOFREAD_ITEMS, OUTLINE_ITEMS, QUILL_ITEMS])
    expect(requests).toStrictEqual([
      {
        path: SCAN,
        commands: [
          { pluginId: DELETE_TEXT.pluginId, command: DELETE_TEXT.id },
          { pluginId: PROOFREAD.pluginId, command: PROOFREAD.id },
          { pluginId: OUTLINE.pluginId, command: OUTLINE.id },
          { pluginId: OTHER_PROOFREAD.pluginId, command: OTHER_PROOFREAD.id },
        ],
      },
    ])
  })

  it('add a command requested after the read of the file kind to the same request, and read the kind once', async () => {
    let answerKind: (kind: FileKind) => void = () => undefined
    const { lists, requests, kindReads } = setUp(() => new Promise((done) => (answerKind = done)))

    const first = lists.listItems(PROOFREAD, SCAN)
    const second = lists.readFileKinds([SCAN]).then(() => lists.listItems(DELETE_TEXT, SCAN))
    const third = lists.readFileKinds([SCAN])
    await flushPromises()
    expect(requests).toStrictEqual([])
    answerKind({ isNote: false, type: 'note' })
    await Promise.all([first, second, third])

    expect(requests).toHaveLength(1)
    expect(requests[0]?.commands).toHaveLength(2)
    expect(kindReads).toStrictEqual([[SCAN]])
  })

  it('skips sending an item list request when only the file kind was read, and reads empty or multi-path kinds directly', async () => {
    const { lists, requests, kindReads } = setUp(async () => ({ isNote: true, type: 'note' }))

    await lists.readFileKinds([SCAN])
    await flushPromises()
    await lists.readFileKinds([])
    await lists.readFileKinds([SCAN, 'scans/two.png'])

    expect(requests).toStrictEqual([])
    expect(kindReads).toStrictEqual([[SCAN], [], [SCAN, 'scans/two.png']])
  })

  it('reject the list of a command the core answered with an error or omitted', async () => {
    const { lists } = setUp()

    await expect(
      lists.listItems({ pluginId: PROOFREAD.pluginId, id: 'ink.gone' }, SCAN),
    ).rejects.toThrow('no enabled plugin declares that command')
    await expect(
      lists.listItems({ pluginId: PROOFREAD.pluginId, id: 'ink.missing' }, SCAN),
    ).rejects.toThrow('no list of the items of ink.missing')
  })
})

describe('the item lists of a later menu', () => {
  it('are read again for the same command over the same file', async () => {
    const { lists, requests } = setUp()
    await lists.listItems(DELETE_TEXT, SCAN)

    expect(await lists.listItems(DELETE_TEXT, SCAN)).toStrictEqual(ITEMS)
    expect(requests).toHaveLength(2)
  })
})
