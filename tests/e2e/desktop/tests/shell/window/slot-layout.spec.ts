import { test, expect, type APIRequestContext, type Locator, type Page } from '../../support/test.js'
import { NEW_RELEASE } from '../../../scripts/stand-ins.mjs'
import { openWorkspace } from '../../support/workspace.js'
import { getStandInsOrigin } from '../../support/media.js'
import { readOwner, writeSetting } from '../../support/vault.js'

/**
 * The bar entries moved by Cmd/Ctrl+drag: the order and the bar the user gives them are kept in
 * `app.layout.slots` of numen.json and drawn again after the window starts anew.
 */
const TASKS_KEY = 'md.numen.views.tasks/status'
const STATS_KEY = 'md.numen.tools.index.stats/stats'
const UPDATER_KEY = 'md.numen.tools.updater/update'

/** The window takes Cmd on macOS and Ctrl elsewhere; the core runs on the machine of the test. */
const MODIFIER = process.platform === 'darwin' ? 'Meta' : 'Control'

const getTasks = (page: Page): Locator => page.locator('.tasks-status__trigger')
const getStats = (page: Page): Locator => page.locator('.index-stats__trigger')
const getUpdater = (page: Page): Locator => page.getByRole('button', { name: /^Update available: v/ })
const getTitleBar = (page: Page): Locator => page.locator('[data-title-bar]')
const getStatusEnd = (page: Page): Locator => page.locator('[data-numen-slot="statusBar.right"]')

interface Box {
  x: number
  y: number
  width: number
  height: number
}

const getBox = async (element: Locator): Promise<Box> => {
  const box = await element.boundingBox()
  if (!box) throw new Error('the element has no box')
  return box
}

const getCenter = (box: Box) => ({ x: box.x + box.width / 2, y: box.y + box.height / 2 })

/** Holds the modifier, presses the entry, moves past the threshold and then to the target, and releases. */
const dragWithModifier = async (
  page: Page,
  entry: Locator,
  getTarget: () => Promise<{ x: number; y: number }>,
): Promise<void> => {
  const from = getCenter(await getBox(entry))
  await page.keyboard.down(MODIFIER)
  await page.mouse.move(from.x, from.y)
  await page.mouse.down()
  await page.mouse.move(from.x + 12, from.y + 12, { steps: 4 })
  const to = await getTarget()
  await page.mouse.move(to.x, to.y, { steps: 8 })
  await page.mouse.up()
  await page.keyboard.up(MODIFIER)
}

const publishNewRelease = async (request: APIRequestContext): Promise<void> => {
  const origin = await getStandInsOrigin(request)
  const answer = await request.put(`${origin}/stand-in/release`, { data: NEW_RELEASE })
  expect(answer.ok()).toBe(true)
}

const withdrawNewRelease = async (request: APIRequestContext): Promise<void> => {
  const origin = await getStandInsOrigin(request)
  const answer = await request.delete(`${origin}/stand-in/release`)
  expect(answer.ok()).toBe(true)
}

const readLayoutSlots = async (request: APIRequestContext): Promise<unknown> =>
  ((await readOwner(request, 'app'))['layout'] as { slots?: unknown } | undefined)?.slots

test.describe('the layout of the bar entries', () => {
  test.describe.configure({ timeout: 120_000 })

  test.beforeEach(async ({ request }) => {
    await writeSetting(request, ['app', 'layout', 'slots'], {})
    await publishNewRelease(request)
  })

  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['app', 'layout', 'slots'], {})
    await withdrawNewRelease(request)
  })

  test('keeps a chip reordered in the title bar and a chip moved into the status bar after a restart', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await expect(getTasks(page)).toBeVisible()
    await expect(getStats(page)).toBeVisible()
    await expect(getUpdater(page)).toBeVisible()
    expect((await getBox(getStats(page))).x).toBeLessThan((await getBox(getTasks(page))).x)

    await dragWithModifier(page, getTasks(page), async () => {
      const stats = await getBox(getStats(page))
      return { x: stats.x + stats.width / 4, y: stats.y + stats.height / 2 }
    })
    await expect
      .poll(async () => (await getBox(getTasks(page))).x < (await getBox(getStats(page))).x)
      .toBe(true)

    await dragWithModifier(page, getUpdater(page), async () => getCenter(await getBox(getStatusEnd(page))))
    await expect(getStatusEnd(page).getByRole('button', { name: /^Update available: v/ })).toBeVisible()

    await openWorkspace(page)

    await expect(getTasks(page)).toBeVisible()
    await expect(getStats(page)).toBeVisible()
    expect((await getBox(getTasks(page))).x).toBeLessThan((await getBox(getStats(page))).x)
    const updater = getStatusEnd(page).getByRole('button', { name: /^Update available: v/ })
    await expect(updater).toBeVisible()
    await expect(getTitleBar(page).getByRole('button', { name: /^Update available: v/ })).toHaveCount(0)
    expect(await readLayoutSlots(request)).toStrictEqual({
      'titleBar.status': [TASKS_KEY, STATS_KEY],
      'statusBar.right': [UPDATER_KEY],
    })

    await updater.click()
    const popover = page.getByRole('dialog', { name: 'Update available' })
    await expect(popover).toBeVisible()
    const popoverBox = await getBox(popover)
    expect(popoverBox.y + popoverBox.height).toBeLessThanOrEqual((await getBox(updater)).y)
  })
})
