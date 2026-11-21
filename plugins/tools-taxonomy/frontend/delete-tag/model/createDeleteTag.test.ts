import { describe, expect, it } from 'vitest'
import type { Api, CommandNote } from '@numen/plugin-sdk'
import { createDeleteTag } from './createDeleteTag'

const NOTE: CommandNote = {
  path: '',
  title: '',
  type: 'note',
  isFolder: false,
  others: [],
  tab: '',
}

interface Call {
  readonly method: string
  readonly body: unknown
}

const createFakeApi = (
  map: Record<string, readonly string[]>,
  written: readonly { path: string; tags: readonly string[]; error?: string }[] = [],
) => {
  const calls: Call[] = []
  const api = {
    call: async (method: string, body: unknown) => {
      calls.push({ method, body })
      if (method === 'GetTagMap') return { ok: true, value: { tags: map } }
      return { ok: true, value: { notes: written } }
    },
  } as unknown as Api
  return { api, calls }
}

const MAP = {
  'a.md': ['Port', 'Sea'],
  'b.md': ['port', 'Sea/Deep'],
  'c.md': ['Quay'],
}

describe('the command Delete tag', () => {
  it('lists each tag once with the number of notes that have it', async () => {
    const { api } = createFakeApi(MAP)

    const items = await createDeleteTag({ api }).listItems?.(NOTE)

    expect(items).toStrictEqual([
      { id: 'Port', title: 'Port', detail: '2 notes' },
      { id: 'Quay', title: 'Quay', detail: '1 note' },
      { id: 'Sea', title: 'Sea', detail: '1 note' },
      { id: 'Sea/Deep', title: 'Sea/Deep', detail: '1 note' },
    ])
  })

  it('removes the tag from every note that has it in one write', async () => {
    const written = [
      { path: 'a.md', tags: ['Sea'] },
      { path: 'b.md', tags: ['Sea/Deep'] },
    ]
    const { api, calls } = createFakeApi(MAP, written)
    const changes: Record<string, readonly string[]>[] = []

    await createDeleteTag({ api, setTags: (changed) => changes.push(changed) }).handler(
      NOTE,
      'Port',
    )

    expect(calls.at(-1)).toStrictEqual({
      method: 'SetTags',
      body: { paths: ['a.md', 'b.md'], toggles: ['Port'] },
    })
    expect(changes).toStrictEqual([{ 'a.md': ['Sea'], 'b.md': ['Sea/Deep'] }])
  })

  it('writes nothing for a tag no note has', async () => {
    const { api, calls } = createFakeApi(MAP)

    await createDeleteTag({ api }).handler(NOTE, 'Gone')

    expect(calls.map((one) => one.method)).toStrictEqual(['GetTagMap'])
  })

  it('fails with the reason of a note that could not be written', async () => {
    const { api } = createFakeApi(MAP, [
      { path: 'a.md', tags: ['Sea'] },
      { path: 'b.md', tags: ['port'], error: 'locked' },
    ])

    await expect(createDeleteTag({ api }).handler(NOTE, 'Port')).rejects.toThrow('b.md: locked')
  })
})
