import {
  test,
  expect,
  type APIRequestContext,
  type Locator,
  type Page,
} from '../../support/test.js'
import { NEW_RELEASE } from '../../../scripts/stand-ins.mjs'
import { getGraph, openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import { getStandInsOrigin, writePluginSetting } from '../../support/media.js'

const UPDATER = 'md.numen.tools.updater'

/** Has the stand-in release feed name a release newer than any build. */
const publishNewRelease = async (request: APIRequestContext, origin: string): Promise<void> => {
  const answer = await request.put(`${origin}/stand-in/release`, { data: NEW_RELEASE })
  expect(answer.ok()).toBe(true)
}

/** Has the stand-in release feed name the old release again. */
const withdrawNewRelease = async (request: APIRequestContext, origin: string): Promise<void> => {
  const answer = await request.delete(`${origin}/stand-in/release`)
  expect(answer.ok()).toBe(true)
}

/** How many times the updater asked the stand-in for the release manifest. */
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

/** The box of an element on the page; the test fails where it is not drawn. */
const getBox = async (element: Locator): Promise<Box> => {
  const box = await element.boundingBox()
  if (!box) throw new Error('the element has no box')
  return box
}

/** The size of the window the page is drawn in. */
const getViewport = (page: Page): { width: number; height: number } => {
  const viewport = page.viewportSize()
  if (!viewport) throw new Error('the page has no viewport')
  return viewport
}

/** Whether the boxes do not overlap: one is under the other, or beside it. */
const isApart = (first: Box, second: Box): boolean =>
  second.y >= first.y + first.height || second.x >= first.x + first.width

/** The space between each box and the next one of a row, from left to right. */
const getGaps = (row: readonly Box[]): number[] =>
  row.slice(1).map((one, index) => one.x - ((row[index]?.x ?? 0) + (row[index]?.width ?? 0)))

/** One computed style of an element. */
const getStyle = (element: Locator, property: string): Promise<string> =>
  element.evaluate((node, name) => getComputedStyle(node).getPropertyValue(name), property)

/** The text colour of a button and the colour it is drawn on, as the page resolves them to red, green and blue. */
const getButtonColours = (button: Locator) =>
  button.evaluate((node) => {
    const style = getComputedStyle(node)
    const toRgb = (colour: string): [number, number, number] => {
      const canvas = document.createElement('canvas')
      canvas.width = 1
      canvas.height = 1
      const context = canvas.getContext('2d', { willReadFrequently: true })!
      context.fillStyle = colour
      context.fillRect(0, 0, 1, 1)
      const [red, green, blue] = context.getImageData(0, 0, 1, 1).data
      return [red!, green!, blue!]
    }
    return { text: toRgb(style.color), ground: toRgb(style.backgroundColor) }
  })

/** The WCAG contrast ratio of the text of a button against its own background. */
const getContrast = async (button: Locator): Promise<number> => {
  const { text, ground } = await getButtonColours(button)
  const getLuminance = (rgb: readonly number[]): number => {
    const [red, green, blue] = rgb.map((channel) => {
      const value = channel / 255
      return value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
    }) as [number, number, number]
    return 0.2126 * red + 0.7152 * green + 0.0722 * blue
  }
  const [lighter, darker] = [getLuminance(text), getLuminance(ground)].sort((a, b) => b - a) as [
    number,
    number,
  ]
  return (lighter + 0.05) / (darker + 0.05)
}

/** The settings page of the updater plugin, opened from the palette. */
const openUpdaterSettings = async (page: Page): Promise<void> => {
  await runPaletteCommand(page, 'Settings', 'Settings')
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

    const indicator = page.getByRole('button', { name: 'Application update available' })
    await expect(indicator).toHaveText(`v${NEW_RELEASE.version}`)
    await indicator.click()
    await expect(page.getByText('Update available')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Download' })).toBeVisible()
  })

  test('opens a framed popover above the indicator, with its details laid out, and closes it on Escape', async ({
    page,
    request,
  }) => {
    await publishNewRelease(request, origin)
    await openWorkspace(page)
    const indicator = page.getByRole('button', { name: 'Application update available' })
    await indicator.click()

    const popover = page.getByRole('dialog', { name: 'Update available' })
    await expect(popover).toBeVisible()
    const box = await getBox(popover)
    const viewport = getViewport(page)
    expect(box.x).toBeGreaterThanOrEqual(0)
    expect(box.y).toBeGreaterThanOrEqual(0)
    expect(box.x + box.width).toBeLessThanOrEqual(viewport.width)
    expect(box.y + box.height).toBeLessThanOrEqual(viewport.height)

    const bar = await getBox(page.getByRole('region', { name: 'Status Bar' }))
    expect(box.y + box.height).toBeLessThanOrEqual(bar.y)
    const trigger = await getBox(indicator)
    expect(box.x).toBeLessThan(trigger.x + trigger.width)
    expect(box.x + box.width).toBeGreaterThan(trigger.x)

    expect(await getStyle(popover, 'background-color')).not.toBe('rgba(0, 0, 0, 0)')
    expect(await getStyle(popover, 'border-top-style')).toBe('solid')
    expect(await getStyle(popover, 'border-top-width')).toBe('1px')

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
      expect(parseFloat(await getStyle(button, 'padding-left'))).toBeGreaterThan(0)
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

  test('draws the Download button in the ink of the accent, as readable as the primary button of the link inspector', async ({
    page,
    request,
  }) => {
    await publishNewRelease(request, origin)
    await openWorkspace(page)
    await page.getByRole('button', { name: 'Application update available' }).click()
    const popover = page.getByRole('dialog', { name: 'Update available' })
    const download = popover.getByRole('button', { name: 'Download', exact: true })
    await expect(download).toBeVisible()
    const downloadContrast = await getContrast(download)
    expect(downloadContrast).toBeGreaterThanOrEqual(4.5)
    await page.keyboard.press('Escape')
    await expect(popover).toHaveCount(0)

    await page.locator('.plex-frame:not([data-moving])').waitFor({ state: 'attached' })
    const edgeHit = getGraph(page).locator('.plex__edge-hit').first()
    await expect(edgeHit).toBeAttached()
    await edgeHit.dispatchEvent('click', { clientX: 300, clientY: 300 })
    const applyButton = page.locator('.plex-link-popover__btn--done')
    await expect(applyButton).toBeVisible()
    const applyContrast = await getContrast(applyButton)

    expect(applyContrast).toBeGreaterThanOrEqual(4.5)
    expect(Math.abs(downloadContrast - applyContrast)).toBeLessThanOrEqual(0.5)
  })

  test('checks the feed again when the release feed address changes, and announces what it finds', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await expect.poll(() => countManifestRequests(request, origin)).toBeGreaterThan(0)
    const indicator = page.getByRole('button', { name: 'Application update available' })
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
