import { describe, expect, it } from 'vitest'
import { ReportedError } from '@numen/plugin-sdk'
import { createMockDialogHost } from '@numen/plugin-sdk/testing'
import { createSaveFile } from './saveFile'

describe('createSaveFile', () => {
  it('gives the text to the save dialog and answers the file chosen', async () => {
    const dialogs = createMockDialogHost()
    dialogs.chooseSavePath('/home/me/data.json')

    const path = await createSaveFile(dialogs)('data.json', '{}')

    expect(path).toBe('/home/me/data.json')
    expect(dialogs.getSaved()).toMatchObject([
      { fileName: 'data.json', extension: '.json', content: '{}' },
    ])
  })

  it('answers undefined when the person closes the dialog', async () => {
    const dialogs = createMockDialogHost()
    dialogs.chooseSavePath(undefined)

    await expect(createSaveFile(dialogs)('data.json', '{}')).resolves.toBeUndefined()
  })

  it('says in plain words that a window with no dialogs cannot save', async () => {
    await expect(createSaveFile(undefined)('data.json', '{}')).rejects.toThrow(
      'This window cannot save files.',
    )
  })

  it('says why the file could not be written', async () => {
    const dialogs = createMockDialogHost()
    dialogs.saveText = () => Promise.reject(new ReportedError('the disk is full'))

    await expect(createSaveFile(dialogs)('data.json', '{}')).rejects.toThrow(
      'Could not save the file: the disk is full',
    )
  })
})
