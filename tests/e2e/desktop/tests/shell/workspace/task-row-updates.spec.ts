import { feedTask, recordRow } from '../../support/task-rows.js'
import { test, expect } from '../../support/test.js'
import { waitForWindow } from '../../support/workspace.js'

/** The id and title the core lists its vector pass under. */
const TASK_ID = 'making the vectors'

/** How often the list of tasks is reported again, as the pass reports every source it embeds. */
const UPDATE_MS = 50

/** How long the pointer rests on the row. */
const REST_MS = 5_000

test.describe('A task row that is reported again', () => {
  test.describe.configure({ timeout: 90_000 })

  test('keeps its nodes and the button under a pointer that rests on it, and Cancel stops the task', async ({
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
    await page.locator('.status-bar__trigger').click()
    const row = page.locator(`.task-item[data-task-id="${TASK_ID}"]`)
    await expect(row).toBeVisible()
    await expect(row.locator('.task-item__detail')).toContainText('Books/Mahabharata')

    // The real pointer, and then it holds still.
    const box = await row.boundingBox()
    expect(box).not.toBeNull()
    await page.mouse.move((box?.x ?? 0) + 40, (box?.y ?? 0) + (box?.height ?? 0) / 2)
    await expect(row.locator('.task-item__btn')).toHaveCSS('opacity', '1')

    const record = await recordRow(page, TASK_ID, REST_MS)

    expect.soft(record.updates, 'the task was reported while the pointer rested').toBeGreaterThan(
      REST_MS / UPDATE_MS / 2,
    )
    expect.soft(record.frames).toBeGreaterThan(30)
    expect.soft(record.changes).toEqual([])
    expect.soft(record.heights, 'the row keeps its height').toHaveLength(1)
    expect.soft(record.isSameRow).toBe(true)
    expect.soft(record.isSameSlot).toBe(true)
    expect.soft(record.isSameRing).toBe(true)
    expect.soft(record.isSameButton).toBe(true)
    // On every frame the button shows and the ring does not, with the row under the pointer.
    expect.soft(record.looks).toEqual(['ring 0, button 1, hover true'])

    await row.getByRole('button', { name: 'Cancel task' }).click()
    await expect.poll(() => cancelled.length).toBe(1)
    expect(cancelled[0]).toContain(TASK_ID)
  })
})
