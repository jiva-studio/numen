import { test, expect, type Locator, type Page } from '../../support/test.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'
import { readOwner, writeSetting } from '../../support/vault.js'

/** `--numen-surface` in the Dracula theme of the preset themes plugin. */
const DRACULA_SURFACE = '#21222c'

/** How often the value of a control is read, and for how long after a change. */
const STEP_MS = 50
const WATCH_MS = 3_000

/** The text of the style element the window carries its theme in. */
const getThemeStyle = (page: Page): Promise<string> =>
  page.evaluate(
    () => document.head.querySelector('style[data-appearance="theme"]')?.textContent ?? '',
  )

/** Opens the page System of the settings tab, and gives back the tab. */
const openSystemSettings = async (page: Page): Promise<Locator> => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: /^System/ }).click()
  return settings
}

/** What the control of a list or a slider shows, read in the page. */
type Control = { readonly list: string } | { readonly slider: string }

/**
 * Reads the control every STEP_MS in the page itself while `change` runs and for WATCH_MS after it,
 * so no value that is drawn between two steps of the test is missed. Gives every value read, in
 * order, with the first read after `change` started marked.
 */
const watchControl = async (
  page: Page,
  control: Control,
  change: () => Promise<void>,
): Promise<{ readonly before: string; readonly samples: readonly string[] }> => {
  const sampler = await page.evaluateHandle(
    ({ target, step }) => {
      const read = (): string => {
        if ('list' in target) {
          const holder = document.querySelector(`[data-testid="setting-${target.list}"]`)
          return holder?.querySelector('button')?.textContent?.trim() ?? ''
        }
        const handle = [...document.querySelectorAll('[role="slider"]')].find((one) => {
          const named = one.getAttribute('aria-labelledby')
          return (
            named !== null && document.getElementById(named)?.textContent?.trim() === target.slider
          )
        })
        return handle?.getAttribute('aria-valuenow') ?? ''
      }
      const samples = [read()]
      const timer = window.setInterval(() => samples.push(read()), step)
      return {
        stop: () => {
          window.clearInterval(timer)
          return samples
        },
      }
    },
    { target: control, step: STEP_MS },
  )
  await change()
  await page.evaluate((ms) => new Promise((resolve) => setTimeout(resolve, ms)), WATCH_MS)
  const samples = await sampler.evaluate((one) => one.stop())
  return { before: samples[0] ?? '', samples }
}

/** The values in the order they were first samples, a value that returns is counted again. */
const getRuns = (samples: readonly string[]): string[] =>
  samples.filter((one, index) => index === 0 || one !== samples[index - 1])

test.describe('A control of the page System after a change', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['app', 'appearance', 'theme'], 'preset:numen')
    await writeSetting(request, ['app', 'appearance', 'mode'], 'system')
    await writeSetting(request, ['app', 'appearance', 'interfaceScale'], 1)
    await writeSetting(request, ['app', 'performance', 'profile'], 'balanced')
  })

  test('shows the theme of a plugin that was chosen and never the old one again, and the window draws it', async ({
    page,
  }) => {
    await openWorkspace(page)
    const settings = await openSystemSettings(page)

    const { before, samples } = await watchControl(page, { list: 'appearance.theme' }, async () => {
      await settings.getByRole('button', { name: 'Theme', exact: true }).click()
      await page.getByRole('menuitemradio', { name: /^Dracula/ }).click()
    })

    expect(before).not.toContain('Dracula')
    const runs = getRuns(samples)
    expect(runs[runs.length - 1]).toContain('Dracula')
    expect(runs.filter((one) => one === before)).toHaveLength(1)
    await expect.poll(() => getThemeStyle(page)).toContain(DRACULA_SURFACE)
  })

  test('shows light or dark as chosen and never the old mode again', async ({ page }) => {
    await openWorkspace(page)
    const settings = await openSystemSettings(page)

    const { before, samples } = await watchControl(page, { list: 'appearance.mode' }, async () => {
      await settings.getByRole('button', { name: 'Light or dark', exact: true }).click()
      await page.getByRole('menuitemradio', { name: /^Dark/ }).click()
    })

    const runs = getRuns(samples)
    expect(runs).toHaveLength(2)
    expect(runs[0]).toBe(before)
    expect(runs[1]).toContain('Dark')
  })

  test('shows the performance profile as chosen and never the old one again', async ({ page }) => {
    await openWorkspace(page)
    const settings = await openSystemSettings(page)

    const { before, samples } = await watchControl(
      page,
      { list: 'performance.profile' },
      async () => {
        await settings.getByRole('button', { name: 'Performance profile', exact: true }).click()
        await page.getByRole('menuitemradio', { name: /^Eco/ }).click()
      },
    )

    const runs = getRuns(samples)
    expect(runs).toHaveLength(2)
    expect(runs[0]).toBe(before)
    expect(runs[1]).toContain('Eco')
  })

  test('moves the interface size forward with the keys and never back', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    const settings = await openSystemSettings(page)
    const slider = settings.getByRole('slider', { name: 'Interface size' })

    const { samples } = await watchControl(page, { slider: 'Interface size' }, async () => {
      await slider.focus()
      await slider.press('ArrowRight')
      await slider.press('ArrowRight')
      await slider.press('ArrowRight')
    })

    expect(getRuns(samples).map(Number)).toStrictEqual([1, 1.1, 1.2, 1.3])
    await expect
      .poll(async () => (await readOwner(request, 'app'))['appearance'])
      .toMatchObject({ interfaceScale: 1.3 })
  })

  test('keeps the interface size at the place the handle is dropped on', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    const settings = await openSystemSettings(page)
    const slider = settings.getByRole('slider', { name: 'Interface size' })
    const box = await slider.boundingBox()
    expect(box).not.toBeNull()
    const startX = (box?.x ?? 0) + (box?.width ?? 0) / 2
    const y = (box?.y ?? 0) + (box?.height ?? 0) / 2

    const { samples } = await watchControl(page, { slider: 'Interface size' }, async () => {
      await page.mouse.move(startX, y)
      await page.mouse.down()
      await page.mouse.move(startX + 60, y, { steps: 12 })
      await page.mouse.move(startX + 120, y, { steps: 12 })
      await page.mouse.up()
    })

    const values = getRuns(samples).map(Number)
    expect(values.length).toBeGreaterThan(1)
    expect(values).toStrictEqual([...values].sort((one, other) => one - other))
    const last = values[values.length - 1]
    expect(last).toBeGreaterThan(1)
    await expect
      .poll(async () => (await readOwner(request, 'app'))['appearance'])
      .toMatchObject({ interfaceScale: last })
  })
})
