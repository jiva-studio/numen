import { test, expect, type Locator, type Page } from 'playwright/test'
import { openVaultFile, runPaletteCommand } from '../../support/workspace'
import { STAND_IN_REPLY } from '../../../scripts/stand-ins.mjs'

/** The two halves of `--numen-surface` in the default theme, as computed colours. */
const SURFACE = { light: 'rgb(250, 244, 232)', dark: 'rgb(30, 25, 22)' } as const

/** The two halves of `--numen-answer-fg`, the colour prose is set in, as computed colours. */
const PROSE = { light: 'rgb(58, 50, 42)', dark: 'rgb(236, 224, 205)' } as const

type Half = keyof typeof SURFACE

const MODE_NAMES = { light: 'Light', dark: 'Dark', system: 'System' } as const

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

const getColour = (locator: Locator) => locator.evaluate((node) => getComputedStyle(node).color)

/**
 * The colour of the text of three plugin views: the body and the heading of a book in the EPUB
 * reader, which is drawn in a shadow root, a face of a stencil in the flashcards studio, and a
 * reply in the agent chat.
 */
const getProseColours = async (page: Page) => {
  await openBook(page)
  const book = page.locator('[data-plugin-id="md.numen.readers.epub"]')
  const reader = {
    body: await getColour(book.getByText('Hello reflowable epub reader').first()),
    heading: await getColour(book.getByRole('heading', { name: 'Chapter 1' }).first()),
  }

  const stencil = await openVaultFile(page, 'cards', 'Animal.md')
  const face = stencil.getByRole('group', { name: 'Preview: Recognise Front' })
  const studio = await getColour(face.getByText('Name', { exact: true }))

  await page.getByRole('tab', { name: /^Agent/ }).click()
  await page.getByRole('textbox', { name: 'Ask about the vault' }).first().fill('Say something')
  await page.getByRole('button', { name: 'Send' }).first().click()
  const reply = page.getByText(STAND_IN_REPLY).first()
  await expect(reply).toBeVisible()
  const agent = await getColour(reply)

  return { reader, studio, agent }
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

        // The mode the page arrives with is written by the server, which stores the mode after
        // the window has applied it.
        await expect
          .poll(async () => {
            await openWindow(page)
            return getSurfaces(page)
          })
          .toMatchObject({ window: SURFACE[forced] })
      } finally {
        await setMode(page, 'system')
      }
    })

    test(`sets the text of the plugin views ${forced} when the mode is ${forced}`, async ({
      page,
    }) => {
      await openWindow(page)
      await setMode(page, forced)
      try {
        const colour = PROSE[forced]
        expect(await getProseColours(page)).toStrictEqual({
          reader: { body: colour, heading: colour },
          studio: colour,
          agent: colour,
        })
      } finally {
        await setMode(page, 'system')
      }
    })
  })
}
