import { observeFor } from '../../support/browser.js'
import { test, expect, type APIRequestContext } from '../../support/test.js'
import type { JsonObject } from '../../support/json.js'

/** The word that makes the mock embedder hold a batch until it is cancelled (engine.MockHoldWord). */
const HOLD_WORD = 'numenmockhold'

/** The id the core lists its vector pass under. */
const VECTOR_TASK_ID = 'making the vectors'

const BODY = Array.from(
  { length: 12 },
  (_, line) =>
    `The keeper of the tide clock winds the brass drum at the change of watch ${line + 1}.`,
).join(' ')

const callRPC = async (
  request: APIRequestContext,
  method: string,
  data: JsonObject,
): Promise<{ status: number; body: JsonObject }> => {
  const answer = await request.post(`/numen.v1.${method}`, {
    data,
    headers: { 'Content-Type': 'application/json' },
  })
  const text = await answer.text()
  return {
    status: answer.status(),
    body: text ? (JSON.parse(text) as JsonObject) : {},
  }
}

test.describe('Cancelling a background task', () => {
  test.describe.configure({ timeout: 180_000 })

  test('Cancel on the row of the vector pass stops the pass', async ({ page, request }) => {
    // The vector pass names the source it embeds, and the mock embedder holds the batch of the held
    // note until it is cancelled: the row of the pass is found by the path of that note.
    await page.goto('/')
    const trigger = page.locator('.tasks-status__trigger')
    await expect(trigger).toBeVisible()
    await expect(page.locator('.tasks-status__summary')).not.toHaveText(/^\d+ of \d+$/)

    const created = await callRPC(request, 'NoteService/CreateNote', {
      title: 'Tide clock',
      path: 'notes',
    })
    expect(created.status).toBe(200)
    const path = String(created.body.path)

    try {
      const writeAnswer = await callRPC(request, 'NoteService/WriteNote', {
        path,
        body: `# Tide clock\n\n${BODY} ${HOLD_WORD}\n`,
      })
      expect(writeAnswer.status).toBe(200)

      await trigger.click()
      const row = page.locator('.task-item').filter({ hasText: path })
      await expect(row).toBeVisible({ timeout: 45_000 })

      // The button shows once the pointer is on the row.
      const cancel = row.getByRole('button', { name: 'Cancel task' })
      await row.hover()
      await expect(cancel).toHaveCSS('opacity', '1')

      await cancel.click()
      await expect(row).toHaveCount(0, { timeout: 15_000 })

      // The core no longer lists the pass, and nothing reports it again.
      const again = await callRPC(request, 'WindowService/CancelTask', {
        window: 'editor',
        id: VECTOR_TASK_ID,
      })
      expect(again.status).toBe(404)
      await observeFor(page, 3_000)
      await expect(row).toHaveCount(0)
    } finally {
      await callRPC(request, 'NoteService/WriteNote', { path, body: `# Tide clock\n\n${BODY}\n` })
    }
  })
})
