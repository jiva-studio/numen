/**
 * The window changing vault: the writes of the open tabs are held while the other vault opens, so
 * a write cannot reach the vault the window is going to, and go on where the switch is refused.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { COMMAND } from '@/entities/command'
import { createInvocation } from '@/features/command-palette'
import {
  getTabs,
  getWindowParts,
  mountWindow,
  openNoteTab,
  registerNoteTabs,
  requests,
  settle,
  vaultState,
} from '@/testing/window'

type Window = Awaited<ReturnType<typeof mountWindow>>

beforeEach(() => {
  vi.stubGlobal('location', { ...globalThis.location, reload: vi.fn() })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const closeFrontTab = (): void => {
  globalThis.dispatchEvent(
    new KeyboardEvent('keydown', { key: 'W', ctrlKey: true, shiftKey: true, cancelable: true }),
  )
}

const mountWithNotes = async (): Promise<Window> => {
  const window = await mountWindow()
  const parts = getWindowParts(window)
  registerNoteTabs(parts)
  await openNoteTab(parts, 'physics/Entropy.md')
  await openNoteTab(parts, 'physics/Heat.md')
  await settle()
  return window
}

const openOtherVault = async (window: Window): Promise<void> => {
  const { palette } = getWindowParts(window)
  const target = { ...palette.getTarget(), vault: { id: 'heat', name: 'Heat' } }
  const invocation = createInvocation(COMMAND.openVault, target)
  await palette.commandDeps.registry.get(COMMAND.openVault)?.(invocation, palette.commandDeps)
}

describe('the window leaving its vault', () => {
  it('writes the open tabs before the core opens the other vault', async () => {
    const window = await mountWithNotes()
    const open = getTabs(window).length
    closeFrontTab()
    await settle()

    await openOtherVault(window)

    expect(requests.opened).toStrictEqual(['heat'])
    expect(requests.tabsWhenOpened).toStrictEqual([open - 1])
  })

  it('writes no tabs once the other vault is open', async () => {
    const window = await mountWithNotes()
    await openOtherVault(window)
    const savedCount = requests.openTabs.length

    closeFrontTab()
    await settle()

    expect(requests.openTabs).toHaveLength(savedCount)
  })

  it('writes the tabs again where the core refuses the other vault', async () => {
    vaultState.isOpenRefused = true
    const window = await mountWithNotes()
    await openOtherVault(window)
    const savedCount = requests.openTabs.length

    closeFrontTab()
    await settle()

    expect(requests.openTabs.length).toBeGreaterThan(savedCount)
  })
})
