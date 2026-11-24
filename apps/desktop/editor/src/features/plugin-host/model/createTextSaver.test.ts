import { describe, expect, it, vi } from 'vitest'
import type { WindowClients } from '@/shared/api/clients'
import { createTextSaver } from './createTextSaver'

const createDialog = (answer: { isSaved: boolean; path: string }) => {
  const saveTextFile = vi.fn(() => Promise.resolve(answer))
  return { saveTextFile, folderDialog: { saveTextFile } as unknown as WindowClients['folderDialog'] }
}

describe('createTextSaver', () => {
  it('sends the text to the save dialog and returns the path chosen', async () => {
    const { saveTextFile, folderDialog } = createDialog({ isSaved: true, path: '/home/me/a.json' })

    const saved = await createTextSaver(folderDialog)({
      title: 'Save',
      fileName: 'a.json',
      content: '{}',
      extension: '.json',
    })

    expect(saved).toStrictEqual({ path: '/home/me/a.json' })
    expect(saveTextFile).toHaveBeenCalledWith({
      title: 'Save',
      fileName: 'a.json',
      content: '{}',
      extension: '.json',
    })
  })

  it('sends an empty extension when there is none', async () => {
    const { saveTextFile, folderDialog } = createDialog({ isSaved: true, path: '/a' })

    await createTextSaver(folderDialog)({ title: 'Save', fileName: 'a', content: '' })

    expect(saveTextFile).toHaveBeenCalledWith(expect.objectContaining({ extension: '' }))
  })

  it('returns undefined when the dialog is closed', async () => {
    const { folderDialog } = createDialog({ isSaved: false, path: '' })

    const saved = await createTextSaver(folderDialog)({ title: 'Save', fileName: 'a', content: '' })

    expect(saved).toBeUndefined()
  })
})
