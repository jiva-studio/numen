import {
  test,
  expect,
  type APIRequestContext,
  type Locator,
  type Page,
} from '../../support/test.js'
import { NEW_RELEASE } from '../../../scripts/stand-ins.mjs'
import { openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import { getStandInsOrigin, writePluginSetting } from '../../support/media.js'

const UPDATER = 'md.numen.tools.updater'

const publishNewRelease = async (request: APIRequestContext, origin: string): Promise<void> => {
  const answer = await request.put(`${origin}/stand-in/release`, { data: NEW_RELEASE })
  expect(answer.ok()).toBe(true)
}

const withdrawNewRelease = async (request: APIRequestContext, origin: string): Promise<void> => {
  const answer = await request.delete(`${origin}/stand-in/release`)
  expect(answer.ok()).toBe(true)
}

const countManifestRequests = async (
  request: APIRequestContext,
  origin: string,
): Promise<number> => {
  const counts = (await (await request.get(`${origin}/stand-in/requests`)).json()) as Record<
    string,
    number
  >
  return counts['/latest/latest.json'] ?? 0
}

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

const getViewport = (page: Page): { width: number; height: number } => {
  const viewport = page.viewportSize()
  if (!viewport) throw new Error('the page has no viewport')
  return viewport
}

const isApart = (first: Box, second: Box): boolean =>
  second.y >= first.y + first.height || second.x >= first.x + first.width

const getGaps = (row: readonly Box[]): number[] =>
  row.slice(1).map((one, index) => one.x - ((row[index]?.x ?? 0) + (row[index]?.width ?? 0)))

const openUpdaterSettings = async (page: Page): Promise<void> => {
  await runPaletteCommand(page, 'Open settings', 'Open settings')
  await page
    .getByRole('button', { name: /Update Notifier/i })
    .first()
    .click()
  await expect(page.getByTestId('setting-channel')).toBeVisible({ timeout: 10_000 })
}

test.describe('the update indicator', () => {
  let origin: string
  let endpoint: string

  test.beforeEach(async ({ request }) => {
    origin = await getStandInsOrigin(request)
    endpoint = `${origin}/latest/latest.json`
  })

  test.afterEach(async ({ request }) => {
    await withdrawNewRelease(request, origin)
    await writePluginSetting(request, UPDATER, 'channel', 'custom')
    await writePluginSetting(request, UPDATER, 'endpoint', endpoint)
  })

  test('announces a newer release the feed names when the window opens', async ({
    page,
    request,
  }) => {
    await publishNewRelease(request, origin)

    await openWorkspace(page)

    const indicator = page.getByRole('button', { name: /^Update available: v/ })
    await expect(indicator).toHaveText(`v${NEW_RELEASE.version}`)
    await indicator.click()
    await expect(page.getByText('Update available')).toBeVisible()
    await expect(
      page
        .getByRole('dialog', { name: 'Update available' })
        .getByRole('button', { name: 'Download' }),
    ).toBeVisible()
  })

  test('opens a popover above the indicator, with its details laid out, and closes it on Escape', async ({
    page,
    request,
  }) => {
    await publishNewRelease(request, origin)
    await openWorkspace(page)
    const indicator = page.getByRole('button', { name: /^Update available: v/ })
    await indicator.click()

    const popover = page.getByRole('dialog', { name: 'Update available' })
    await expect(popover).toBeVisible()
    const box = await getBox(popover)
    const viewport = getViewport(page)
    expect(box.x).toBeGreaterThanOrEqual(0)
    expect(box.y).toBeGreaterThanOrEqual(0)
    expect(box.x + box.width).toBeLessThanOrEqual(viewport.width)
    expect(box.y + box.height).toBeLessThanOrEqual(viewport.height)

    const bar = await getBox(page.locator('[data-title-bar]'))
    expect(box.y).toBeGreaterThanOrEqual(bar.y + bar.height)
    const trigger = await getBox(indicator)
    expect(box.x).toBeLessThan(trigger.x + trigger.width)
    expect(box.x + box.width).toBeGreaterThan(trigger.x)

    const header = popover.locator('header')
    await expect(header).toContainText('Update available')
    await expect(header).not.toContainText(`${NEW_RELEASE.version}`)
    const release = popover.getByText(`v${NEW_RELEASE.version} · Released on`)
    await expect(release).toBeVisible()
    const titleBox = await getBox(popover.getByText('Update available', { exact: true }))
    expect(isApart(titleBox, await getBox(release))).toBe(true)

    const names = ['Download', 'Release notes', 'Dismiss']
    const boxes = []
    for (const name of names) {
      const button = popover.getByRole('button', { name, exact: true })
      await expect(button).toBeVisible()
      expect(await button.evaluate((node) => node.tagName)).toBe('BUTTON')
      const one = await getBox(button)
      expect(one.x).toBeGreaterThanOrEqual(box.x)
      expect(one.x + one.width).toBeLessThanOrEqual(box.x + box.width)
      expect(one.y + one.height).toBeLessThanOrEqual(box.y + box.height)
      boxes.push(one)
    }
    expect(
      Math.max(...boxes.map((one) => one.y)) - Math.min(...boxes.map((one) => one.y)),
    ).toBeLessThan(4)
    expect(getGaps(boxes).every((gap) => gap > 0)).toBe(true)
    await expect(popover.getByRole('button', { name: /^Close/ })).toBeVisible()

    await page.keyboard.press('Escape')

    await expect(popover).toHaveCount(0)
  })

  test('checks the feed again when the release feed address changes, and announces what it finds', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await expect.poll(() => countManifestRequests(request, origin)).toBeGreaterThan(0)
    const indicator = page.getByRole('button', { name: /^Update available: v/ })
    await expect(indicator).toHaveCount(0)
    const requestCount = await countManifestRequests(request, origin)
    await publishNewRelease(request, origin)

    await writePluginSetting(request, UPDATER, 'endpoint', `${endpoint}?again=1`)

    await expect(indicator).toHaveText(`v${NEW_RELEASE.version}`)
    expect(await countManifestRequests(request, origin)).toBeGreaterThan(requestCount)

    await withdrawNewRelease(request, origin)
    await writePluginSetting(request, UPDATER, 'endpoint', endpoint)

    await expect(indicator).toHaveCount(0)
  })

  test('shows the release feed address for the Custom channel only, and keeps what was typed', async ({
    page,
  }) => {
    await openWorkspace(page)
    await openUpdaterSettings(page)
    const channel = page.getByTestId('setting-channel').locator('.settings__choice')
    const address = page.getByRole('textbox', { name: 'Release feed address' })
    const newAddress = 'http://127.0.0.1:9/latest.json'

    await expect(channel).toHaveText('Custom')
    await expect(address).toBeVisible()
    await address.fill(newAddress)
    await address.press('Tab')

    for (const name of ['Stable', 'Beta']) {
      await channel.click()
      await page.getByRole('menuitemradio', { name: new RegExp(`^${name}`) }).click()
      await expect(channel).toHaveText(name)
      await expect(page.getByText('Release feed address', { exact: true })).toHaveCount(0)
      await expect(page.getByTestId('setting-endpoint')).toHaveCount(0)
    }

    await channel.click()
    await page.getByRole('menuitemradio', { name: /^Custom/ }).click()
    await expect(address).toHaveValue(newAddress)
  })
})
