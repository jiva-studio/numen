import { observeFor } from '../../support/browser.js'
import { test, expect, type APIRequestContext } from '../../support/test.js'

/** The word that makes the mock embedder hold a batch until it is cancelled (engine.MockHoldWord). */
const HOLD_WORD = 'numenmockhold'

/** The id the core lists its vector pass under. */
const VECTOR_TASK_ID = 'making the vectors'

const BODY = Array.from(
  { length: 12 },
  (_, line) =>
    `The keeper of the tide clock winds the brass drum at the change of watch ${line + 1}.`,
).join(' ')

/** Calls a unary RPC of the debug server with a JSON body and returns its answer. */
const callRPC = async (
  request: APIRequestContext,
  method: string,
  data: Record<string, unknown>,
): Promise<{ status: number; body: Record<string, unknown> }> => {
  const answer = await request.post(`/numen.v1.${method}`, {
    data,
    headers: { 'Content-Type': 'application/json' },
  })
  const text = await answer.text()
  return {
    status: answer.status(),
    body: text ? (JSON.parse(text) as Record<string, unknown>) : {},
  }
}

/** Whether the vault is read and every chunk it holds has its vector, as the core reports it. */
const isIndexed = async (request: APIRequestContext): Promise<boolean> => {
  const { body } = await callRPC(request, 'VaultService/GetVaultState', {})
  const scan = body['scan'] as { isReady?: boolean } | undefined
  const coverage = body['coverage'] as { chunkCount?: string; embeddedCount?: string } | undefined
  return scan?.isReady === true && coverage?.chunkCount === coverage?.embeddedCount
}

test.describe('Cancelling a background task', () => {
  test.describe.configure({ timeout: 240_000 })

  test('the row holds still under the pointer, and Cancel stops the vector pass', async ({
    page,
    request,
  }) => {
    // The vector pass is one task that names the source it last embedded, and it embeds the pending
    // chunks source by source. The row of the held note is the row of the pass only once the
    // fixtures are indexed, the text of their PDFs included: the core reports the PDF that cannot
    // be read as failed once its text pass has got there, and the chunks of the others follow.
    await page.goto('/')
    const trigger = page.locator('.status-bar__trigger')
    await expect(trigger).toBeVisible()
    await trigger.click()
    await expect(
      page.locator('.task-item--failed').filter({ hasText: 'corrupted.pdf' }),
    ).toBeVisible({ timeout: 120_000 })
    await page.keyboard.press('Escape')
    await expect.poll(() => isIndexed(request), { timeout: 120_000 }).toBe(true)
    await expect(page.locator('.status-bar__summary')).not.toHaveText(/^[1-9]\d* active$/)

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
      const cancel = row.getByRole('button', { name: 'Cancel task' })
      await expect(cancel).toBeVisible()

      // The button is transparent in its slot until the pointer reaches the row, and showing it
      // moves nothing.
      const readGeometry = () =>
        row.evaluate((element) =>
          [element, ...element.querySelectorAll('.task-item__indicator, .task-item__action button')]
            .map((part) => {
              const box = part.getBoundingClientRect()
              return `${box.x},${box.y},${box.width},${box.height}`
            })
            .join(';'),
        )
      await trigger.hover()
      await expect(cancel).toHaveCSS('opacity', '0')
      const geometry = await readGeometry()

      await row.hover()
      await expect(cancel).toHaveCSS('opacity', '1')
      expect(await readGeometry()).toBe(geometry)
      await expect(row.locator('.task-item__indicator')).toHaveCSS('opacity', '1')
      // Records every DOM change in the row and the visibility of its parts on each frame, for a
      // second while the pointer rests on it.
      const record = await row.evaluate(async (element) => {
        const changes: string[] = []
        const observer = new MutationObserver((list) => {
          for (const one of list) {
            if (one.type === 'childList') changes.push(`children of ${one.target.nodeName}`)
            if (one.type === 'attributes' && one.attributeName !== 'stroke-dashoffset') {
              changes.push(`${one.attributeName} of ${one.target.nodeName}`)
            }
          }
        })
        observer.observe(element, { subtree: true, childList: true, attributes: true })
        const frames = new Set<string>()
        const end = performance.now() + 1_000
        while (performance.now() < end) {
          const parts = [
            element.querySelector('.task-item__indicator'),
            element.querySelector('.task-item__action button'),
          ]
          frames.add(
            parts
              .map((part) => {
                if (!part) return 'missing'
                const style = getComputedStyle(part)
                const box = part.getBoundingClientRect()
                return `${style.display} ${style.opacity} ${style.visibility} ${box.x} ${box.y}`
              })
              .join(' | '),
          )
          await new Promise((resolve) => requestAnimationFrame(resolve))
        }
        observer.disconnect()
        return { changes, frames: [...frames], isConnected: element.isConnected }
      })
      expect(record.changes).toEqual([])
      expect(record.frames).toHaveLength(1)
      expect(record.frames[0]).not.toContain('missing')
      expect(record.isConnected).toBe(true)

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
