import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'

const PLUGIN = 'Index stats'
const PLUGIN_ID = 'md.numen.tools.index.stats'

/** How long the core takes to answer the write of the switch, in milliseconds. */
const WRITE_MS = 2_000

/** How long the backend of the plugin answers late, as it does while it starts, in milliseconds. */
const BACKEND_LATE_MS = 6_000

/** How long after the switch moves the status item may take to appear, in milliseconds. */
const APPEARS_WITHIN_MS = 1_000

/** How long after the switch moves the status item may take to leave, in milliseconds. */
const LEAVES_WITHIN_MS = 500

/** What one frame showed of the switch and of the status item. */
interface Frame {
  readonly elapsed: number
  readonly checked: string
  readonly isDisabled: boolean
  readonly hasItem: boolean
}

const writeEnabled = async (page: Page, isOn: boolean): Promise<void> => {
  const answer = await page.request.post('/numen.v1.SettingsService/WriteSettings', {
    data: { settings: [{ at: ['plugins', PLUGIN_ID, 'enabled'], value: JSON.stringify(isOn) }] },
    headers: { 'Content-Type': 'application/json' },
  })
  expect(answer.ok()).toBe(true)
}

/** Clicks the switch and samples it and the status item on every frame for `span` milliseconds. */
const clickAndSample = (page: Page, span: number): Promise<Frame[]> =>
  page.evaluate(
    (milliseconds) =>
      new Promise<Frame[]>((resolve) => {
        const frames: Frame[] = []
        const toggle = document.querySelector('[role="switch"][aria-label^="Enabled"]')
        const startedAt = performance.now()
        const sample = (): void => {
          const elapsed = performance.now() - startedAt
          frames.push({
            elapsed,
            checked: toggle?.getAttribute('aria-checked') ?? '',
            isDisabled: toggle?.hasAttribute('disabled') ?? false,
            hasItem: document.querySelector('.index-stats') !== null,
          })
          if (elapsed < milliseconds) requestAnimationFrame(sample)
          else resolve(frames)
        }
        ;(toggle as HTMLElement).click()
        requestAnimationFrame(sample)
      }),
    span,
  )

/** The time the switch first showed `value`, and the time the status item first matched `isShown`. */
const findMoments = (
  frames: readonly Frame[],
  value: string,
  isShown: boolean,
): { switchAt: number; itemAt: number } => {
  const switchAt = frames.find((one) => one.checked === value)?.elapsed ?? Infinity
  const itemAt = frames.find((one) => one.hasItem === isShown)?.elapsed ?? Infinity
  return { switchAt, itemAt }
}

test.describe('The switch of a plugin and its status item', () => {
  test.describe.configure({ timeout: 120_000 })

  test.afterEach(async ({ page }) => {
    await writeEnabled(page, true)
  })

  test('waits for the answer of the core, then shows the status item within a second and removes it within half of one', async ({
    page,
  }) => {
    await writeEnabled(page, true)
    await page.route('**/numen.v1.SettingsService/WriteSettings', async (route) => {
      const response = await route.fetch()
      await new Promise((resolve) => setTimeout(resolve, WRITE_MS))
      await route.fulfill({ response })
    })
    await page.route('**/*GetStats*', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, BACKEND_LATE_MS))
      await route.continue()
    })
    await openWorkspace(page)
    await runCommand(page, 'Settings', /^Settings$/)
    const settings = page.getByRole('tabpanel', { name: /Settings/ })
    await settings.getByRole('button', { name: new RegExp(PLUGIN) }).click()
    const toggle = settings.getByRole('switch', { name: new RegExp(`Enabled ${PLUGIN}`) })
    await expect(toggle).toHaveAttribute('aria-checked', 'true')
    await expect(page.locator('.index-stats')).toBeVisible()

    const off = await clickAndSample(page, 4_000)
    const pending = off.filter((one) => one.elapsed < WRITE_MS - 300)
    expect(pending.every((one) => one.checked === 'true' && one.isDisabled)).toBe(true)
    expect(pending.every((one) => one.hasItem)).toBe(true)
    expect(new Set(off.map((one) => one.checked))).toEqual(new Set(['true', 'false']))
    const turnedOff = findMoments(off, 'false', false)
    expect(turnedOff.switchAt).toBeGreaterThanOrEqual(WRITE_MS - 300)
    expect(turnedOff.itemAt - turnedOff.switchAt).toBeLessThan(LEAVES_WITHIN_MS)
    expect(off.at(-1)?.isDisabled).toBe(false)

    const on = await clickAndSample(page, 4_000)
    expect(new Set(on.map((one) => one.checked))).toEqual(new Set(['false', 'true']))
    const turnedOn = findMoments(on, 'true', true)
    expect(turnedOn.switchAt).toBeGreaterThanOrEqual(WRITE_MS - 300)
    expect(turnedOn.itemAt - turnedOn.switchAt).toBeLessThan(APPEARS_WITHIN_MS)
    expect(on.at(-1)?.isDisabled).toBe(false)
  })
})
