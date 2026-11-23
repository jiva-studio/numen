/**
 * The paths the window follows: a note the core asks to show opens in a tab, and a note renamed
 * is followed by the tabs and the palette step standing on it.
 */
import { describe, expect, it, vi } from 'vitest'
import { COMMAND } from '@/shared/model/command'
import {
  vaultChanges,
  getWindowParts,
  mountWindowWithPalette,
  registerNoteTabs,
  focusRequests,
  requests,
  settle,
  submitPalette,
  enterPaletteText,
} from '@/testing/window'

type Window = Awaited<ReturnType<typeof mountWindowWithPalette>>

const ENTROPY = 'physics/Entropy.md'
const HEAT = 'physics/Heat.md'

const mountWithNotes = async (): Promise<Window> => {
  const window = await mountWindowWithPalette()
  registerNoteTabs(getWindowParts(window))
  return window
}

const readSavedPaths = (): readonly string[] =>
  requests.openTabs.at(-1)?.tabs.map((one) => one.path) ?? []

const showNote = async (path: string): Promise<void> => {
  focusRequests.ask({ path, start: 0, length: 0 })
  await vi.waitFor(() => expect(readSavedPaths()).toContain(path))
}

const renameNote = async (from: string, to: string): Promise<void> => {
  vaultChanges.report({ paths: [to], moves: [{ from, to }] })
  await settle()
  await settle()
}

describe('a note the core asks the window to show', () => {
  it('opens in a tab, which the open tabs record', async () => {
    await mountWithNotes()

    await showNote(ENTROPY)

    expect(readSavedPaths()).toContain(ENTROPY)
  })
})

describe('a note created by a command of the window', () => {
  it('opens in a tab, which the open tabs record', async () => {
    const window = await mountWithNotes()
    const { palette } = getWindowParts(window)

    palette.runCommand(COMMAND.createNote, palette.getTarget())
    await settle()
    await enterPaletteText('Entropy')
    await submitPalette()

    await vi.waitFor(() => expect(readSavedPaths()).toContain('Entropy.md'))
    expect(requests.made).toStrictEqual(['Entropy'])
  })
})

describe('a note renamed', () => {
  it('moves the tab that holds it and notifies the plugin host of the vault change', async () => {
    const window = await mountWithNotes()
    const { pluginHost } = getWindowParts(window)
    const onVaultChange = vi.fn()
    pluginHost.contextHost.events.onVaultChange('md.example.watcher', onVaultChange)
    await showNote(ENTROPY)

    await renameNote(ENTROPY, HEAT)

    await vi.waitFor(() => expect(readSavedPaths()).toContain(HEAT))
    expect(readSavedPaths()).not.toContain(ENTROPY)
    expect(onVaultChange).toHaveBeenCalledWith({
      paths: [HEAT],
      renamed: [{ from: ENTROPY, to: HEAT }],
    })
  })

  describe('under a step of the palette that waits for its name', () => {
    const askForNewName = (window: Window) => {
      const { palette } = getWindowParts(window)
      palette.runCommand(COMMAND.renameNote, {
        ...palette.getTarget(),
        path: ENTROPY,
        title: 'Entropy',
      })
    }

    it('is carried out over the new path of the note', async () => {
      const window = await mountWithNotes()
      askForNewName(window)
      await settle()

      await renameNote(ENTROPY, HEAT)
      await enterPaletteText('Kelvin')
      await submitPalette()

      expect(requests.renamed).toStrictEqual([`${HEAT} Kelvin`])
    })
  })
})
