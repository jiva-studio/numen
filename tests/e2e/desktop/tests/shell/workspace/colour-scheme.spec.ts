import { test, expect, type Page } from 'playwright/test'
import { runPaletteCommand } from '../../support/workspace'

/** The two halves of `--numen-surface` in the default theme, as computed colours. */
const SURFACE = { light: 'rgb(250, 244, 232)', dark: 'rgb(30, 25, 22)' } as const

type Half = keyof typeof SURFACE

const MODE_NAMES = { light: 'Light', dark: 'Dark', system: 'Follow the system' } as const

/**
 * The colour `--numen-surface` resolves to in the window, in the files view, which is drawn in the
 * window's document, and in the EPUB reader, which is drawn in a shadow root.
 */
const getSurfaces = (page: Page) =>
  page.evaluate(() => {
    const resolve = (parent: Element | ShadowRoot | null | undefined) => {
      if (!parent) return null
      const probe = document.createElement('div')
      probe.style.color = 'var(--numen-surface)'
      parent.append(probe)
      const colour = getComputedStyle(probe).color
      probe.remove()
      return colour
    }
    const reader = document.querySelector('[data-plugin-id="md.numen.readers.epub"]')
    return {
      window: resolve(document.body),
      files: resolve(document.querySelector('[data-tree-row]')),
      reader: resolve(reader?.shadowRoot),
    }
  })

const setMode = async (page: Page, mode: keyof typeof MODE_NAMES) => {
  await runPaletteCommand(page, 'Light or dark', 'Light or dark')
  await page.getByRole('option', { name: MODE_NAMES[mode] }).first().click()
}

/** Loads the window and waits for its files view. */
const openWindow = async (page: Page) => {
  await page.goto('/')
  await expect(page.getByRole('tab', { name: /^Files/ })).toBeVisible()
}

const openBook = async (page: Page) => {
  await page.getByRole('tab', { name: /^Files/ }).click()
  await page.locator('[data-tree-row="assets"]').press('Enter')
  await page.locator('[data-tree-row="assets/sample.epub"]').press('Enter')
  await expect(page.locator('text=Hello reflowable epub reader').first()).toBeVisible()
}

for (const system of ['light', 'dark'] as const) {
  const forced: Half = system === 'light' ? 'dark' : 'light'

  test.describe(`a ${system} system`, () => {
    test.use({ colorScheme: system })

    test(`draws the window and the plugin views ${forced} when the mode is ${forced}`, async ({
      page,
    }) => {
      await openWindow(page)
      // The mode is kept in the settings, which a test before this one may have left set.
      await setMode(page, 'system')
      await openBook(page)
      const getExpected = (half: Half) => ({
        window: SURFACE[half],
        files: SURFACE[half],
        reader: SURFACE[half],
      })
      await expect.poll(() => getSurfaces(page)).toStrictEqual(getExpected(system))

      try {
        await setMode(page, forced)
        await expect.poll(() => getSurfaces(page)).toStrictEqual(getExpected(forced))

        // The mode the page arrives with is written by the server.
        await openWindow(page)
        await expect.poll(() => getSurfaces(page)).toMatchObject({ window: SURFACE[forced] })
      } finally {
        await setMode(page, 'system')
      }
    })
  })
}
