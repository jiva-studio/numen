/**
 * The commands run through the window: the palette runs a command with the registered handler,
 * what the command objects to is written as a message of the window, and the files a plugin is
 * provided run their commands with the dependencies of the window.
 */
import { describe, expect, it } from 'vitest'
import { COMMAND } from '@/entities/command'
import { PALETTE_WORDS } from '@/features/command-palette'
import {
  cards,
  getWindowParts,
  mountWindow,
  mountWindowWithPalette,
  requests,
  settle,
  submitPalette,
  typeInPalette,
} from '@/testing/window'

describe('a command run through the palette of the window', () => {
  it('is carried out by the handler registered for it', async () => {
    const window = await mountWindowWithPalette()
    const { palette } = getWindowParts(window)

    palette.runCommand(COMMAND.createNote, palette.getTarget())
    await settle()
    await typeInPalette('Entropy')
    await submitPalette()

    expect(requests.made).toStrictEqual(['Entropy'])
  })

  it('is written as a message of the window where the command is not offered over the target', async () => {
    const window = await mountWindow()
    const { palette } = getWindowParts(window)

    palette.runCommand(COMMAND.openFile, { ...palette.getTarget(), path: '' })
    await settle()

    expect(cards().join(' ')).toContain(PALETTE_WORDS.noNote)
  })
})

describe('the files a plugin is provided', () => {
  it('run their commands with the dependencies of the window', async () => {
    const window = await mountWindow()
    const { plugins } = getWindowParts(window)

    await plugins.host.pluginHost.contextHost.vault.getFiles()?.createFolder('Zoology')

    expect(requests.folders).toStrictEqual(['Zoology'])
  })
})
