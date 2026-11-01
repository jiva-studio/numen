/**
 * Playwright's `test`, with the open tabs the core keeps for the vault cleared before each test.
 * The window opens the tabs the core kept, and every test shares the vault, so each one starts
 * from the tabs a new vault opens with.
 */
import { test as base, type APIRequestContext } from 'playwright/test'

export * from 'playwright/test'

/** Clears the open tabs the core keeps for the vault in the window. */
export const forgetOpenTabs = async (request: APIRequestContext): Promise<void> => {
  const answer = await request.post('/numen.v1.WorkspaceService/WriteOpenTabs', {
    data: { tabs: [], activeId: '' },
    headers: { 'Content-Type': 'application/json' },
  })
  if (!answer.ok()) throw new Error(`clearing the open tabs answered ${answer.status()}`)
}

/** The saved state of each open tab the core keeps for the vault in the window, by the tab's path. */
export const readSavedStates = async (request: APIRequestContext): Promise<Map<string, string>> => {
  const answer = await request.post('/numen.v1.WorkspaceService/ReadOpenTabs', {
    data: {},
    headers: { 'Content-Type': 'application/json' },
  })
  if (!answer.ok()) throw new Error(`reading the open tabs answered ${answer.status()}`)
  const { tabs = [] } = (await answer.json()) as { tabs?: { path?: string; state?: string }[] }
  return new Map(tabs.map((one) => [one.path ?? '', one.state ?? '']))
}

export const test = base.extend<{ isFreshTabs: void }>({
  isFreshTabs: [
    async ({ request }, use) => {
      await forgetOpenTabs(request)
      await use()
    },
    { auto: true },
  ],
})
