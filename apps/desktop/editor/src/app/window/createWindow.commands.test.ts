/**
 * The commands run through the window: the palette runs a command with the registered handler,
 * what the command objects to is written as a message of the window, and the files a plugin is
 * provided run their commands with the dependencies of the window.
 */
import { describe, expect, it, vi } from 'vitest'
import type { PluginServices } from '@/features/plugin-commands'
import type * as PluginSupportModule from '../plugins/pluginSupport'
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

/** What the window gives the plugins, read as the plugin host reads it. */
const provided = vi.hoisted(() => ({ read: undefined as (() => PluginServices) | undefined }))

vi.mock('../plugins/pluginSupport', async (original) => {
  const module = await original<typeof PluginSupportModule>()
  return {
    ...module,
    createPluginSupport: (deps: Parameters<typeof module.createPluginSupport>[0]) => {
      provided.read = deps.getServices
      return module.createPluginSupport(deps)
    },
  }
})

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
    await mountWindow()

    await provided.read?.().files.createFolder('Zoology')

    expect(requests.folders).toStrictEqual(['Zoology'])
  })
})
