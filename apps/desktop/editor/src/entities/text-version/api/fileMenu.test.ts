/**
 * The menu a file's text versions make, as the window reads it, and an entry
 * as the window hands it back to be run.
 */
import { describe, expect, it, vi } from 'vitest'
import { FileMenuAction } from '@numen/protocol'

vi.mock('@/shared/clients', () => ({ textVersions: request }))

const request = {
  getFileMenu: vi.fn(),
  runFileMenuEntry: vi.fn(),
}

const { getFileMenu, runFileMenuEntry } = await import('./fileMenu')

describe('getFileMenu', () => {
  it('maps the entries to window actions', async () => {
    request.getFileMenu.mockResolvedValue({
      entries: [
        { action: FileMenuAction.RECOGNISE, pluginId: 'ocr', pluginName: 'OCR', versionId: '' },
        {
          action: FileMenuAction.PROOFREAD,
          pluginId: 'fix',
          pluginName: 'Fix',
          versionId: 'ocr/k',
        },
        {
          action: FileMenuAction.DELETE_VERSION,
          pluginId: 'ocr',
          pluginName: 'OCR',
          versionId: 'ocr/k',
        },
        { action: FileMenuAction.UNSPECIFIED, pluginId: 'x', pluginName: 'X', versionId: '' },
      ],
    })

    const entries = await getFileMenu('books/Scan.pdf')

    expect(request.getFileMenu).toHaveBeenCalledWith({ path: 'books/Scan.pdf' })
    expect(entries.map((one) => one.action)).toStrictEqual([
      'recognise',
      'proofread',
      'delete-version',
    ])
    expect(entries[1]).toStrictEqual({
      action: 'proofread',
      pluginId: 'fix',
      pluginName: 'Fix',
      versionId: 'ocr/k',
    })
  })
})

describe('runFileMenuEntry', () => {
  it('sends the schema action of the entry', async () => {
    request.runFileMenuEntry.mockResolvedValue({})

    await runFileMenuEntry('books/Scan.pdf', {
      action: 'delete-version',
      pluginId: 'ocr',
      pluginName: 'OCR',
      versionId: 'ocr/k',
    })

    expect(request.runFileMenuEntry).toHaveBeenCalledWith({
      path: 'books/Scan.pdf',
      action: FileMenuAction.DELETE_VERSION,
      pluginId: 'ocr',
      versionId: 'ocr/k',
    })
  })
})
