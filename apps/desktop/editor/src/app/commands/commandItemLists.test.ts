/**
 * The item lists of the commands the core lists them for: read in one request for the commands a
 * menu asks for at once, over one read of the kind of the file, and kept until the file, its text
 * versions or the commands change. Every path is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import type { FileKind } from '@/shared/model/fileTypes'
import type { ItemListRequest } from '@/entities/plugin'
import { createCommandItemLists } from './commandItemLists'

const DELETE_TEXT = { pluginId: 'org.example.ears', id: 'app:version.delete' }
const PROOFREAD = { pluginId: 'org.example.ink', id: 'ink.proofread' }
const SCAN = 'scans/page.png'
const TALK = 'talks/a.mp3'
const ITEMS = [{ id: 'org.example.ears/ears', title: 'transcript (live)' }]

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const setUp = (kind: () => Promise<FileKind | undefined> = async () => undefined) => {
  const requests: { path: string; commands: readonly ItemListRequest[] }[] = []
  const kindReads: (readonly string[])[] = []
  const lists = createCommandItemLists({
    listLists: vi.fn(async (path: string, commands: readonly ItemListRequest[]) => {
      requests.push({ path, commands })
      return commands.map((one) =>
        one.command === 'ink.gone'
          ? {
              ...one,
              result: { ok: false as const, error: 'no enabled plugin declares that command' },
            }
          : { ...one, result: { ok: true as const, value: ITEMS } },
      )
    }),
    fileKinds: async (paths) => {
      kindReads.push(paths)
      const one = await kind()
      return new Map(one ? [[paths[0]!, one]] : [])
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
    ])

    expect(await answers).toStrictEqual([ITEMS, ITEMS])
    expect(requests).toStrictEqual([
      {
        path: SCAN,
        commands: [
          { pluginId: DELETE_TEXT.pluginId, command: DELETE_TEXT.id },
          { pluginId: PROOFREAD.pluginId, command: PROOFREAD.id },
        ],
      },
    ])
  })

  it('add a command requested after the read of the file kind to the same request, and read the kind once', async () => {
    let answerKind: (kind: FileKind) => void = () => undefined
    const { lists, requests, kindReads } = setUp(() => new Promise((done) => (answerKind = done)))

    const first = lists.listItems(PROOFREAD, SCAN)
    const second = lists.fileKinds([SCAN]).then(() => lists.listItems(DELETE_TEXT, SCAN))
    const third = lists.fileKinds([SCAN])
    answerKind({ isNote: false, type: 'note' })
    await Promise.all([first, second, third])

    expect(requests).toHaveLength(1)
    expect(requests[0]?.commands).toHaveLength(2)
    expect(kindReads).toStrictEqual([[SCAN]])
  })

  it('reject the list of a command the core answered with an error', async () => {
    const { lists } = setUp()

    await expect(
      lists.listItems({ pluginId: PROOFREAD.pluginId, id: 'ink.gone' }, SCAN),
    ).rejects.toThrow('no enabled plugin declares that command')
  })
})

describe('the item lists kept', () => {
  it('answer the same command over the same file without a request', async () => {
    const { lists, requests } = setUp()
    await lists.listItems(DELETE_TEXT, SCAN)

    expect(await lists.listItems(DELETE_TEXT, SCAN)).toStrictEqual(ITEMS)
    expect(requests).toHaveLength(1)
  })

  it('are read again for a file that changed, and kept for the others', async () => {
    const { lists, requests } = setUp()
    await Promise.all([lists.listItems(DELETE_TEXT, SCAN), lists.listItems(DELETE_TEXT, TALK)])

    lists.dropPaths([SCAN])
    await lists.listItems(DELETE_TEXT, SCAN)
    await lists.listItems(DELETE_TEXT, TALK)

    expect(requests.map((one) => one.path)).toStrictEqual([SCAN, TALK, SCAN])
  })

  it('are all read again once the commands change', async () => {
    const { lists, requests } = setUp()
    await lists.listItems(DELETE_TEXT, SCAN)

    lists.dropAll()
    await lists.listItems(DELETE_TEXT, SCAN)

    expect(requests).toHaveLength(2)
  })

  it('do not keep a list read while its file changed', async () => {
    const { lists, requests } = setUp()
    const request = lists.listItems(DELETE_TEXT, SCAN)
    await settle()

    lists.dropPaths([SCAN])
    await request
    await lists.listItems(DELETE_TEXT, SCAN)

    expect(requests).toHaveLength(2)
  })

  it('do not keep a list that failed', async () => {
    const { lists, requests } = setUp()
    const unknownCommand = { pluginId: PROOFREAD.pluginId, id: 'ink.gone' }
    await expect(lists.listItems(unknownCommand, SCAN)).rejects.toThrow()

    await expect(lists.listItems(unknownCommand, SCAN)).rejects.toThrow()
    expect(requests).toHaveLength(2)
  })
})
