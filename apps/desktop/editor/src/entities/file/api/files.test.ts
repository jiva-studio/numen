/**
 * The calls of the file service, over the real client and a router in place of the core.
 */
import { describe, expect, it, vi } from 'vitest'
import { NOTE_ERROR_CODE } from '@numen/core-client'
import { createHandledClients } from '@/testing/coreAnswers'
import { createFileApi } from './files'

const request = {
  removeFile: vi.fn(),
  emptyTrash: vi.fn(),
  listFiles: vi.fn(),
  moveFile: vi.fn(),
  createFolder: vi.fn(),
  listFileKinds: vi.fn(),
}

const api = createFileApi(createHandledClients(request).files)

describe('a file removed', () => {
  it('is moved to the trash, and rejected with error', async () => {
    request.removeFile.mockResolvedValue({ trashed: '.trash/a.md', dangling: ['b.md'] })
    expect(await api.removeFile('a.md')).toEqual({
      ok: true,
      value: { trashed: '.trash/a.md', dangling: ['b.md'] },
    })
    expect(request.removeFile.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'a.md' })

    request.removeFile.mockResolvedValue({ error: NOTE_ERROR_CODE.NOT_FOUND })
    expect(await api.removeFile('a.md')).toEqual({ ok: false, error: 'notFound' })
  })

  it('is counted by the paths the trash could not delete, when it is emptied', async () => {
    request.emptyTrash.mockResolvedValue({ undeleted: ['.trash/a.md', '.trash/b/c.md'] })

    expect(await api.emptyTrash()).toBe(2)
    expect(request.emptyTrash.mock.lastCall?.[0]).toMatchObject({})
  })
})

describe('a folder listed', () => {
  it('is listed under its path, each row in the words of the window', async () => {
    request.listFiles.mockResolvedValue({
      entries: [
        { path: 'Notes/test.md', name: 'test.md', isFolder: false, isNote: true, type: '' },
      ],
    })

    expect(await api.listFiles('Notes')).toEqual([
      { path: 'Notes/test.md', name: 'test.md', isFolder: false, isNote: true, type: 'note' },
    ])
    expect(request.listFiles.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'Notes' })
  })
})

describe('a file moved', () => {
  it('is moved from one path to another, and rejected with error', async () => {
    request.moveFile.mockResolvedValue({ moved: { from: 'a.md', to: 'b.md', repaired: [] } })
    expect(await api.moveFile('a.md', 'b.md')).toEqual({
      ok: true,
      value: { from: 'a.md', to: 'b.md', repaired: [] },
    })
    expect(request.moveFile.mock.calls.at(-1)?.[0]).toMatchObject({ from: 'a.md', to: 'b.md' })

    request.moveFile.mockResolvedValue({})
    expect(await api.moveFile('a.md', 'a.md')).toEqual({ ok: true, value: null })

    request.moveFile.mockResolvedValue({ error: NOTE_ERROR_CODE.ALREADY_EXISTS })
    expect(await api.moveFile('a.md', 'b.md')).toEqual({ ok: false, error: 'alreadyExists' })
  })
})

describe('a folder created', () => {
  it('is created at its path, and refused with the error', async () => {
    request.createFolder.mockResolvedValue({})
    expect(await api.createFolder('Notes/New')).toStrictEqual({ ok: true, value: null })
    expect(request.createFolder.mock.calls.at(-1)?.[0]).toMatchObject({ path: 'Notes/New' })

    request.createFolder.mockResolvedValue({ error: NOTE_ERROR_CODE.INVALID_NAME })
    expect(await api.createFolder('Notes/?')).toStrictEqual({ ok: false, error: 'invalidName' })
  })
})

describe('the kinds of files', () => {
  it('are requested of every path given', async () => {
    request.listFileKinds.mockResolvedValue({
      kinds: [
        { path: 'a.bin', isNote: false, type: '' },
        { path: 'a.md', isNote: true, type: 'deck' },
      ],
    })

    const kinds = await api.getFileKinds(['a.bin', 'a.md'])

    expect(request.listFileKinds.mock.calls.at(-1)?.[0]).toMatchObject({
      paths: ['a.bin', 'a.md'],
    })
    expect(kinds.get('a.bin')).toStrictEqual({ isNote: false, type: 'note' })
    expect(kinds.get('a.md')).toStrictEqual({ isNote: true, type: 'deck' })
  })

  it('carry the properties of a note, and the tab kind that opens it', async () => {
    request.listFileKinds.mockResolvedValue({
      kinds: [
        {
          path: 'link.md',
          isNote: true,
          type: 'link',
          tabKind: 'links',
          properties: { media: 'plugin://x/a.mp4' },
        },
      ],
    })

    const kinds = await api.getFileKinds(['link.md'])

    expect(kinds.get('link.md')).toStrictEqual({
      isNote: true,
      type: 'link',
      tabKind: 'links',
      properties: { media: 'plugin://x/a.mp4' },
    })
  })
})
