import { observeFor, waitForFrames } from '../../support/browser.js'
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
  test.describe.configure({ timeout: 180_000 })

  test('the row holds still under the pointer, and Cancel stops the vector pass', async ({
    page,
    request,
  }) => {
    // The vector pass is one task that names the source it last embedded, and it embeds the pending
    // chunks source by source. The row of the pass is found by the path of the held note, so the
    // start waits for the vault to be indexed and for no task to be running. The failed task of
    // corrupted.pdf is not waited for: a plugin that stops takes its tasks out of the list, so it is
    // there in some runs and not in others.
    await page.goto('/')
    const trigger = page.locator('.status-bar__trigger')
    await expect(trigger).toBeVisible()
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

      // The ring and the button are stacked in one slot, in the same box. The button is transparent
      // until the pointer reaches the row, then it takes the place of the ring and moves nothing.
      const readGeometry = () =>
        row.evaluate((element) =>
          [
            element,
            ...element.querySelectorAll('.task-item__slot, .task-item__indicator, .task-item__btn'),
          ]
            .map((part) => {
              const box = part.getBoundingClientRect()
              return `${box.x},${box.y},${box.width},${box.height}`
            })
            .join(';'),
        )
      const ring = row.locator('.task-item__indicator')
      await trigger.hover()
      await expect(cancel).toHaveCSS('opacity', '0')
      await expect(ring).toHaveCSS('opacity', '1')
      // The popover is placed on its first frames, and the boxes of the row are empty until then:
      // the baseline is the geometry two reads a few frames apart agree on.
      let geometry = ''
      await expect
        .poll(async () => {
          const before = await readGeometry()
          await waitForFrames(page, 3)
          geometry = await readGeometry()
          return before === geometry && !geometry.startsWith('0,0,0,0')
        })
        .toBe(true)
      const boxes = await row.evaluate((element) => {
        const read = (selector: string) => {
          const box = element.querySelector(selector)?.getBoundingClientRect()
          return box ? [box.x, box.y, box.width, box.height] : []
        }
        return [read('.task-item__indicator'), read('.task-item__btn')]
      })
      expect(boxes[0]).toHaveLength(4)
      expect(boxes[1]).toEqual(boxes[0])

      await row.hover()
      await expect(cancel).toHaveCSS('opacity', '1')
      await expect(ring).toHaveCSS('opacity', '0')
      expect(await readGeometry()).toBe(geometry)
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
            element.querySelector('.task-item__btn'),
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
