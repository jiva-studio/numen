/**
 * The paths the window follows: a note the core asks to show opens in a tab, and a note renamed
 * is followed by the tabs and the palette step standing on it.
 */
import { describe, expect, it, vi } from 'vitest'
import { COMMAND } from '@/entities/command'
import {
  changed,
  getWindowParts,
  mountWindowWithPalette,
  registerNoteTabs,
  outside,
  requests,
  settle,
  submitPalette,
  typeInPalette,
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
  outside.ask({ path, start: 0, length: 0 })
  await vi.waitFor(() => expect(readSavedPaths()).toContain(path))
}

const renameNote = async (from: string, to: string): Promise<void> => {
  changed.report({ paths: [to], renamed: [{ from, to }] })
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
    await typeInPalette('Entropy')
    await submitPalette()

    await vi.waitFor(() => expect(readSavedPaths()).toContain('Entropy.md'))
    expect(requests.made).toStrictEqual(['Entropy'])
  })
})

describe('a note renamed', () => {
  it('moves the tab that holds it', async () => {
    await mountWithNotes()
    await showNote(ENTROPY)

    await renameNote(ENTROPY, HEAT)

    await vi.waitFor(() => expect(readSavedPaths()).toContain(HEAT))
    expect(readSavedPaths()).not.toContain(ENTROPY)
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
      await typeInPalette('Kelvin')
      await submitPalette()

      expect(requests.renamed).toStrictEqual([`${HEAT} Kelvin`])
    })
  })
})
