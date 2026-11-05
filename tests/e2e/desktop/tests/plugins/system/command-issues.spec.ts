import { observeFor } from '../../support/browser.js'
import { test, expect } from '../../support/test.js'
import { openTaskList, openWorkspace } from '../../support/workspace.js'

/** The words a plugin that declares a command and registers no handler for it is reported in. */
const MISSING_HANDLER = /declared in the manifest and has no handler/i

test.describe('the commands of the stock plugins after the window starts', () => {
  test('are not listed as failed background tasks for a missing handler', async ({ page }) => {
    await openWorkspace(page)
    // The plugins register their commands as they start, a moment after the window is drawn.
    await observeFor(page, 1_000)

    const tasks = await openTaskList(page)

    await expect(tasks).not.toContainText(MISSING_HANDLER)
  })
})
