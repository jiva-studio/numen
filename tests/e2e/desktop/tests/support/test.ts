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
    data: { tabs: [], front: '' },
    headers: { 'Content-Type': 'application/json' },
  })
  if (!answer.ok()) throw new Error(`clearing the open tabs answered ${answer.status()}`)
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
