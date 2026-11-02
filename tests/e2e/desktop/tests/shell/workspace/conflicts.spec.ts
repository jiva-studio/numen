import { test, expect, type Page, type Route } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  getNotePanel,
  openNote,
  readNote,
  typeAtEnd,
  waitForSavedBody,
  writeNote,
} from '../../support/vault.js'

const WRITE_NOTE = '**/numen.v1.NoteService/WriteNote'

/**
 * Holds the writes the window makes until the returned function lets them go, so a change made on
 * disk lands between what the tab read and what it writes.
 */
const blockWrites = async (
  page: Page,
): Promise<{ pendingRoutes: Route[]; release: () => Promise<void> }> => {
  const pendingRoutes: Route[] = []
  let isHolding = true
  await page.route(WRITE_NOTE, async (route) => {
    if (isHolding) pendingRoutes.push(route)
    else await route.continue()
  })
  return {
    pendingRoutes,
    release: async () => {
      isHolding = false
      for (const route of pendingRoutes) await route.continue()
    },
  }
}

/** Opens a fresh note, types into it, and writes another body on disk before the typing lands. */
const raiseConflict = async (page: Page, notePath: string) => {
  await writeNote(page.request, notePath, 'The lock gate opens at high water.\n')
  await openWorkspace(page)
  const text = await openNote(page, notePath)
  await expect(text).toContainText('The lock gate opens at high water.')

  const writes = await blockWrites(page)
  await typeAtEnd(page, text, ' Typed in the window.')
  await expect.poll(() => writes.pendingRoutes.length).toBeGreaterThan(0)
  await writeNote(page.request, notePath, 'Written by another program.\n')
  await writes.release()

  const banner = getNotePanel(page, notePath).getByRole('alert')
  await expect(banner).toContainText('Saving paused: the file changed on disk.')
  return { text, banner }
}

test.describe('A note that changes on disk', () => {
  test('keeps the typed text over the disk version when the person keeps theirs', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Lock gate'))
    try {
      const { banner } = await raiseConflict(page, notePath)
      expect((await readNote(request, notePath)).body).toBe('Written by another program.\n')

      await banner.getByRole('button', { name: 'Keep my version' }).click()

      await waitForSavedBody(request, notePath, 'Typed in the window.')
      expect((await readNote(request, notePath)).body).not.toContain('Written by another program.')
      await expect(banner).toHaveCount(0)
    } finally {
      await destroyNote(request, notePath)
    }
  })

  test('shows the disk version and drops the typed text when the person takes it', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Weir'))
    try {
      const { text, banner } = await raiseConflict(page, notePath)

      await banner.getByRole('button', { name: 'Use disk version' }).click()

      await expect(text).toContainText('Written by another program.')
      await expect(text).not.toContainText('Typed in the window.')
      await expect(banner).toHaveCount(0)
      expect((await readNote(request, notePath)).body).toBe('Written by another program.\n')
    } finally {
      await destroyNote(request, notePath)
    }
  })

  test('shows the disk version at once when nothing was typed', async ({ page, request }) => {
    const notePath = await createNote(request, createUniqueTitle('Sluice'))
    try {
      await writeNote(request, notePath, 'The sluice drains the basin.\n')
      await openWorkspace(page)
      const text = await openNote(page, notePath)
      await expect(text).toContainText('The sluice drains the basin.')

      await writeNote(request, notePath, 'The sluice fills the basin.\n')

      await expect(text).toContainText('The sluice fills the basin.')
      await expect(getNotePanel(page, notePath).getByRole('alert')).toHaveCount(0)
      expect((await readNote(request, notePath)).body).toBe('The sluice fills the basin.\n')
    } finally {
      await destroyNote(request, notePath)
    }
  })
})

/** Turns any CSS colour the browser computes into red, green, blue and alpha. */
const readColour = (page: Page, css: string) =>
  page.evaluate((value) => {
    const pixel = document.createElement('canvas')
    pixel.width = pixel.height = 1
    const context = pixel.getContext('2d', { willReadFrequently: true })!
    context.clearRect(0, 0, 1, 1)
    context.fillStyle = value
    context.fillRect(0, 0, 1, 1)
    const [red = 0, green = 0, blue = 0, alpha = 0] = context.getImageData(0, 0, 1, 1).data
    return { red, green, blue, alpha: alpha / 255 }
  }, css)

const getLuminance = ({
  red,
  green,
  blue,
}: {
  red: number
  green: number
  blue: number
}): number => {
  const [r = 0, g = 0, b = 0] = [red, green, blue].map((one) => {
    const part = one / 255
    return part <= 0.03928 ? part / 12.92 : ((part + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

test.describe('The answers of a note banner', () => {
  test('are filled buttons with a readable text, and Recreate it writes the note at its own path', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Dyke'))
    try {
      await writeNote(request, notePath, 'The dyke holds the tide.\n')
      await openWorkspace(page)
      const text = await openNote(page, notePath)
      await expect(text).toContainText('The dyke holds the tide.')

      await destroyNote(request, notePath)

      const banner = getNotePanel(page, notePath).getByRole('alert')
      await expect(banner).toContainText('the note was removed from the vault')
      const recreate = banner.getByRole('button', { name: 'Recreate it' })
      const style = await recreate.evaluate((one) => {
        const computed = getComputedStyle(one)
        return { background: computed.backgroundColor, ink: computed.color }
      })
      const background = await readColour(page, style.background)
      const ink = await readColour(page, style.ink)
      expect(background.alpha).toBeGreaterThan(0)
      const [lighter, darker] = [getLuminance(background), getLuminance(ink)].sort((a, b) => b - a)
      expect(((lighter ?? 0) + 0.05) / ((darker ?? 0) + 0.05)).toBeGreaterThanOrEqual(4.5)

      await recreate.click()

      await waitForSavedBody(request, notePath, 'The dyke holds the tide.')
      await expect(banner).toHaveCount(0)
    } finally {
      await destroyNote(request, notePath)
    }
  })
})
