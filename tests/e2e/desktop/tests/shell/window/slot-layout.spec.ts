import {
  test,
  expect,
  type APIRequestContext,
  type Locator,
  type Page,
} from '../../support/test.js'
import { NEW_RELEASE } from '../../../scripts/stand-ins.mjs'
import { openWorkspace } from '../../support/workspace.js'
import { getStandInsOrigin } from '../../support/media.js'
import { readOwner, writeSetting } from '../../support/vault.js'

/**
 * The slot entries moved from their right-click menu: the order and the slot the user gives them are
 * kept in `app.layout.slots` of numen.json and drawn again after the window starts anew.
 */
const TASKS_KEY = 'titleBar.status|md.numen.views.tasks/status'
const STATS_KEY = 'titleBar.status|md.numen.tools.index.stats/stats'
const UPDATER_KEY = 'titleBar.status|md.numen.tools.updater/update'

const getTasks = (page: Page): Locator => page.locator('.tasks-status__trigger')
const getStats = (page: Page): Locator => page.locator('.index-stats__trigger')
const getUpdater = (page: Page): Locator =>
  page.getByRole('button', { name: /^Update available: v/ })
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

/** Opens the menu of the entry with a right click and chooses Move to, the slot and the item. */
const moveFromMenu = async (
  page: Page,
  entry: Locator,
  path: readonly [slot: string, item: string],
): Promise<void> => {
  await entry.click({ button: 'right' })
  await page.getByRole('menuitem', { name: 'Move to', exact: true }).click()
  await page.getByRole('menuitem', { name: path[0], exact: true }).click()
  await page.getByRole('menuitem', { name: path[1], exact: true }).click()
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

test.describe('the layout of the slot entries', () => {
  test.describe.configure({ timeout: 120_000 })

  test.beforeEach(async ({ request }) => {
    await writeSetting(request, ['app', 'layout', 'slots'], {})
    await publishNewRelease(request)
  })

  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['app', 'layout', 'slots'], {})
    await withdrawNewRelease(request)
  })

  test('keeps a chip reordered in the title bar and a chip moved into the status bar from its menu after a restart', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await expect(getTasks(page)).toBeVisible()
    await expect(getStats(page)).toBeVisible()
    await expect(getUpdater(page)).toBeVisible()
    expect((await getBox(getStats(page))).x).toBeLessThan((await getBox(getTasks(page))).x)

    await moveFromMenu(page, getTasks(page), ['Title bar status', 'Index statistics'])
    await expect
      .poll(async () => (await getBox(getTasks(page))).x < (await getBox(getStats(page))).x)
      .toBe(true)

    await moveFromMenu(page, getUpdater(page), ['Status bar end', 'At the end'])
    await expect(
      getStatusEnd(page).getByRole('button', { name: /^Update available: v/ }),
    ).toBeVisible()

    await openWorkspace(page)

    await expect(getTasks(page)).toBeVisible()
    await expect(getStats(page)).toBeVisible()
    expect((await getBox(getTasks(page))).x).toBeLessThan((await getBox(getStats(page))).x)
    const updater = getStatusEnd(page).getByRole('button', { name: /^Update available: v/ })
    await expect(updater).toBeVisible()
    await expect(
      getTitleBar(page).getByRole('button', { name: /^Update available: v/ }),
    ).toHaveCount(0)
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
