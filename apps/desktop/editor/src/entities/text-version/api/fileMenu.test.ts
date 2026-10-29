/**
 * The menu built from a file's text versions, as the window reads it, and an
 * entry as the window passes it back to be run.
 */
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/shared/clients', () => ({ textVersions: request }))

const request = {
  getFileMenu: vi.fn(),
  runFileMenuEntry: vi.fn(),
}

const { getFileMenu, runFileMenuEntry } = await import('./fileMenu')

const wire = (command: string, versionId: string, title: string, icon = '') => ({
  pluginId: 'fix',
  pluginName: 'Fix',
  command,
  title,
  icon,
  versionId,
  isLive: true,
})

describe('getFileMenu', () => {
  it('keeps the command, title and icon the plugin declares', async () => {
    request.getFileMenu.mockResolvedValue({
      entries: [
        wire('proofread', 'ocr/k', 'Proofread', 'spell-check'),
        wire('delete-proofreading', 'fix/k', 'Delete proofreading'),
      ],
    })

    const entries = await getFileMenu('books/Scan.pdf')

    expect(request.getFileMenu).toHaveBeenCalledWith({ path: 'books/Scan.pdf' })
    expect(entries).toStrictEqual([
      {
        pluginId: 'fix',
        pluginName: 'Fix',
        command: 'proofread',
        title: 'Proofread',
        icon: 'spell-check',
        versionId: 'ocr/k',
      },
      {
        pluginId: 'fix',
        pluginName: 'Fix',
        command: 'delete-proofreading',
        title: 'Delete proofreading',
        icon: '',
        versionId: 'fix/k',
      },
    ])
  })
})

describe('runFileMenuEntry', () => {
  it('sends the plugin, the command and the version of the entry', async () => {
    request.runFileMenuEntry.mockResolvedValue({})

    await runFileMenuEntry('books/Scan.pdf', {
      pluginId: 'fix',
      pluginName: 'Fix',
      command: 'proofread',
      title: 'Proofread',
      icon: 'spell-check',
      versionId: 'ocr/k',
    })

    expect(request.runFileMenuEntry).toHaveBeenCalledWith({
      path: 'books/Scan.pdf',
      pluginId: 'fix',
      command: 'proofread',
      versionId: 'ocr/k',
    })
  })
})
