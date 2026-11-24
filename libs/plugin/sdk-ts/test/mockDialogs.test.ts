import { describe, expect, it } from 'vitest'
import { createMockDialogHost } from '../src/testing/mockDialogs'

const TEXT = { title: 'Save', fileName: 'account.json', content: '{}', extension: '.json' }

describe('createMockDialogHost', () => {
  it('keeps the texts the person was asked to save and answers the file chosen', async () => {
    const dialogs = createMockDialogHost()
    dialogs.chooseSavePath('/home/me/account.json')

    const saved = await dialogs.saveText(TEXT)

    expect(saved).toStrictEqual({ path: '/home/me/account.json' })
    expect(dialogs.getSaved()).toStrictEqual([TEXT])
  })

  it('answers undefined when the person closes the dialog', async () => {
    const dialogs = createMockDialogHost()
    dialogs.chooseSavePath(undefined)

    await expect(dialogs.saveText(TEXT)).resolves.toBeUndefined()
  })
})
