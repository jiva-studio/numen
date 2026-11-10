import { describe, expect, it, vi } from 'vitest'
import { createTabNavigation } from './tabNavigation'

const create = () => {
  const openFile = vi.fn(() => Promise.resolve())
  const setShownPath = vi.fn()
  const navigateTabsAt = vi.fn((_from: string, _to: string, onMove?: (tabId: string) => void) => {
    onMove?.('graph:1')
    return Promise.resolve()
  })
  const navigation = createTabNavigation({
    tabOpeners: { openFile },
    windowTabs: { tabs: { navigateTabsAt } as never },
    tabSync: { setShownPath },
  })
  return { navigation, openFile, setShownPath, navigateTabsAt }
}

describe('a note travelled to', () => {
  it('is opened in a tab of the current window', async () => {
    const { navigation, openFile } = create()

    await navigation.travel('a.md')

    expect(openFile).toHaveBeenCalledExactlyOnceWith('a.md', '', 'current')
  })
})

describe('a note left', () => {
  it('moves every tab standing on it to the other note and records the path each shows', async () => {
    const { navigation, navigateTabsAt, setShownPath } = create()

    await navigation.leavePath('Gone.md', 'Root.md')

    expect(navigateTabsAt).toHaveBeenCalledWith('Gone.md', 'Root.md', expect.any(Function))
    expect(setShownPath).toHaveBeenCalledExactlyOnceWith('graph:1', 'Root.md')
  })
})
