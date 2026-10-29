/**
 * The menu built from a file's text versions, as the window reads it, and an
 * entry as the window passes it back to be run.
 */
import { describe, expect, it, vi } from 'vitest'
import { FileMenuAction } from '@numen/protocol'

vi.mock('@/shared/clients', () => ({ textVersions: request }))

const request = {
  getFileMenu: vi.fn(),
  runFileMenuEntry: vi.fn(),
}

const { getFileMenu, runFileMenuEntry } = await import('./fileMenu')

const wire = (action: FileMenuAction, versionId: string, title = '', icon = '') => ({
  action,
  pluginId: 'fix',
  pluginName: 'Fix',
  title,
  icon,
  versionId,
})

describe('getFileMenu', () => {
  it('keeps the title and icon the plugin declares', async () => {
    request.getFileMenu.mockResolvedValue({
      entries: [
        wire(FileMenuAction.CREATE_TEXT, 'ocr/k', 'Proofread', 'spell-check'),
        wire(FileMenuAction.DELETE_VERSION, 'fix/k'),
        wire(FileMenuAction.UNSPECIFIED, ''),
      ],
    })

    const entries = await getFileMenu('books/Scan.pdf')

    expect(request.getFileMenu).toHaveBeenCalledWith({ path: 'books/Scan.pdf' })
    expect(entries).toStrictEqual([
      {
        action: 'create-text',
        pluginId: 'fix',
        pluginName: 'Fix',
        title: 'Proofread',
        icon: 'spell-check',
        versionId: 'ocr/k',
      },
      {
        action: 'delete-version',
        pluginId: 'fix',
        pluginName: 'Fix',
        title: '',
        icon: '',
        versionId: 'fix/k',
      },
    ])
  })
})

describe('runFileMenuEntry', () => {
  it('sends the schema action of the entry and the plugin that runs it', async () => {
    request.runFileMenuEntry.mockResolvedValue({})

    await runFileMenuEntry('books/Scan.pdf', {
      action: 'create-text',
      pluginId: 'fix',
      pluginName: 'Fix',
      title: 'Proofread',
      icon: 'spell-check',
      versionId: 'ocr/k',
    })

    expect(request.runFileMenuEntry).toHaveBeenCalledWith({
      path: 'books/Scan.pdf',
      action: FileMenuAction.CREATE_TEXT,
      pluginId: 'fix',
      versionId: 'ocr/k',
    })
  })
})
