import { feedTask } from '../../support/task-rows.js'
import { test, expect } from '../../support/test.js'
import { waitForWindow } from '../../support/workspace.js'

/** The id and title the core lists its vector pass under. */
const TASK_ID = 'making the vectors'

/** How often the list of tasks is reported again, as the pass reports every source it embeds. */
const UPDATE_MS = 50

test.describe('A task row that is reported again', () => {
  test.describe.configure({ timeout: 90_000 })

  test('shows each new source, and Cancel stops the task while it is reported', async ({
    page,
  }) => {
    // The window's own stream of tasks is replaced by one the test feeds, so the list is reported
    // again every UPDATE_MS the way a pass over 124 thousand chunks reports it.
    await feedTask(page, TASK_ID, UPDATE_MS)
    const cancelled: string[] = []
    await page.route('**/numen.v1.WindowService/CancelTask', async (route) => {
      cancelled.push(String(route.request().postData()))
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    await page.goto('/')
    await waitForWindow(page)
    await page.locator('[data-tasks-status] button').click()
    const row = page.locator(`.task-item[data-task-id="${TASK_ID}"]`)
    await expect(row).toBeVisible()

    // Every report names the next part, and the row shows it.
    const detail = row.locator('.task-item__detail')
    await expect(detail).toContainText(/part \d+\.md/)
    const first = await detail.textContent()
    await expect(detail).not.toHaveText(first ?? '')
    await expect(detail).toContainText('Books/Mahabharata')

    await row.hover()
    const cancel = row.getByRole('button', { name: 'Cancel task' })
    await expect(cancel).toHaveCSS('opacity', '1')
    await cancel.click()
    await expect.poll(() => cancelled.length).toBe(1)
    expect(cancelled[0]).toContain(TASK_ID)
  })
})
