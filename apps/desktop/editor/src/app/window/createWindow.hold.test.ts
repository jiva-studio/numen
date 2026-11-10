/**
 * The window changing vault: the writes of the open tabs are held while the other vault opens, so
 * a write cannot reach the vault the window is going to, and go on where the switch is refused.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { COMMAND, commands as commandRegistry } from '@/entities/command'
import { createInvocation } from '@/features/command-palette'
import {
  getTabs,
  getWindowParts,
  mountWindow,
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

/** The keystroke that closes the tab in front, which changes the open tabs. */
const closeFrontTab = (): void => {
  globalThis.dispatchEvent(
    new KeyboardEvent('keydown', { key: 'W', ctrlKey: true, shiftKey: true, cancelable: true }),
  )
}

/** The command that opens another vault, run as the palette runs it once a vault is chosen. */
const openOtherVault = async (window: Window): Promise<void> => {
  const { palette } = getWindowParts(window)
  const target = { ...palette.getTarget(), vault: { id: 'heat', name: 'Heat' } }
  const invocation = createInvocation(COMMAND.openVault, target)
  await commandRegistry.get(COMMAND.openVault)?.(invocation, palette.commandDeps)
}

describe('the window leaving its vault', () => {
  it('has written the open tabs as they are when the core is asked to open the other vault', async () => {
    const window = await mountWindow()
    const open = getTabs(window).length
    closeFrontTab()

    await openOtherVault(window)

    expect(requests.opened).toStrictEqual(['heat'])
    expect(requests.tabsWhenOpened).toStrictEqual([open - 1])
  })

  it('writes no tabs once the other vault is open', async () => {
    const window = await mountWindow()
    await openOtherVault(window)
    const written = requests.openTabs.length

    closeFrontTab()
    await settle()

    expect(requests.openTabs).toHaveLength(written)
  })

  it('writes the tabs again where the core refuses the other vault', async () => {
    vaultState.isOpenRefused = true
    const window = await mountWindow()
    await openOtherVault(window)
    const written = requests.openTabs.length

    closeFrontTab()
    await settle()

    expect(requests.openTabs.length).toBeGreaterThan(written)
  })
})
