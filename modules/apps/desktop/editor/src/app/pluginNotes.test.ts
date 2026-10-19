/** The reads and writes of a plugin's note tab, in the order the tab made them. */
import { describe, expect, it, vi } from 'vitest'
import { createPluginNotes } from './pluginNotes'
import type { NoteResult, WriteResult } from '@/entities/note'

const readOk = (body: string): NoteResult => ({ ok: true, value: { body } }) as NoteResult
const writeOk = (): WriteResult => ({ ok: true, value: { body: '' } }) as WriteResult

interface Pending {
  readonly body: string
  readonly finish: () => void
}

const getPending = (pending: readonly Pending[], at: number): Pending => {
  const one = pending[at]
  if (!one) throw new Error('write ' + at + ' was not made')
  return one
}

const createDeps = () => {
  const stored = new Map<string, string>([['a.md', 'original']])
  const pending: Pending[] = []
  const write = vi.fn(
    (path: string, body: string) =>
      new Promise<WriteResult>((resolve) => {
        pending.push({
          body,
          finish: () => {
            stored.set(path, body)
            resolve(writeOk())
          },
        })
      }),
  )
  const read = vi.fn((path: string) => Promise.resolve(readOk(stored.get(path) ?? '')))
  return { stored, pending, write, read }
}

const tick = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0))

describe('createPluginNotes', () => {
  it('has one write of a path in the air and writes only the newest body after it', async () => {
    const deps = createDeps()
    const notes = createPluginNotes(deps)

    const first = notes.writeNote('a.md', 'one')
    const middle = notes.writeNote('a.md', 'two')
    const last = notes.writeNote('a.md', 'three')
    expect(deps.write).toHaveBeenCalledTimes(1)

    getPending(deps.pending, 0).finish()
    await tick()
    expect(deps.write).toHaveBeenCalledTimes(2)
    expect(deps.write).toHaveBeenLastCalledWith('a.md', 'three', null)

    getPending(deps.pending, 1).finish()
    await Promise.all([first, middle, last])
    expect(deps.stored.get('a.md')).toBe('three')
  })

  it('answers a read after the writes of that path have landed', async () => {
    const deps = createDeps()
    const notes = createPluginNotes(deps)
    const write = notes.writeNote('a.md', 'edited')

    const read = notes.readNote('a.md')
    await tick()
    expect(deps.read).not.toHaveBeenCalled()

    getPending(deps.pending, 0).finish()
    expect(await read).toEqual({ body: 'edited' })
    await write
  })

  it('reads a path at once when it has no write in the air', async () => {
    const deps = createDeps()
    const notes = createPluginNotes(deps)
    const write = notes.writeNote('b.md', 'other')

    expect(await notes.readNote('a.md')).toEqual({ body: 'original' })
    getPending(deps.pending, 0).finish()
    await write
  })

  it('writes again once the previous writes of the path are done', async () => {
    const deps = createDeps()
    const notes = createPluginNotes(deps)

    const first = notes.writeNote('a.md', 'one')
    getPending(deps.pending, 0).finish()
    await first

    const second = notes.writeNote('a.md', 'two')
    expect(deps.write).toHaveBeenCalledTimes(2)
    getPending(deps.pending, 1).finish()
    await second
    expect(deps.stored.get('a.md')).toBe('two')
  })

  it('rejects a write the vault could not reach and still answers reads', async () => {
    const deps = createDeps()
    deps.write.mockRejectedValueOnce(new Error('unreachable'))
    const notes = createPluginNotes(deps)

    await expect(notes.writeNote('a.md', 'lost')).rejects.toThrow('unreachable')
    expect(await notes.readNote('a.md')).toEqual({ body: 'original' })
  })

  it('words the error of a note that cannot be read', async () => {
    const deps = createDeps()
    deps.read.mockResolvedValueOnce({ ok: false, error: 'unreadable' } as NoteResult)
    const notes = createPluginNotes(deps)

    const answered = await notes.readNote('a.md')
    expect(answered.body).toBe('')
    expect(answered.error).toBeTruthy()
  })
})
