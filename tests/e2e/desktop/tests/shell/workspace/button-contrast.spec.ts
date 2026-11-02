import { test, expect, type Locator, type Page } from '../../support/test.js'
import { NEW_RELEASE } from '../../../scripts/stand-ins.mjs'
import { getGraph, getPaletteField, openWorkspace, runCommand } from '../../support/workspace.js'
import { getStandInsOrigin } from '../../support/media.js'
import {
  createNoteWithBody,
  destroyNotes,
  getTab,
  showInFiles,
  getRow,
} from '../../support/layout.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  openNote,
  writeNote,
  writeSetting,
} from '../../support/vault.js'
import { expectReadable, measureContrasts, listShortfalls } from '../../support/contrast.js'

/**
 * Every page that draws a solid button on the accent, in both halves of the theme: the text of each
 * button must be readable on the ground it is drawn on. The accent is yellow in the dark half and
 * brown in the light one, and the text on it is the accent's own ink; a button that keeps the
 * colour of the text around it is cream on yellow, which cannot be read.
 */

/** Has the stand-in release feed name a release newer than any build. */
const publishNewRelease = async (page: Page, origin: string): Promise<void> => {
  const answer = await page.request.put(`${origin}/stand-in/release`, { data: NEW_RELEASE })
  expect(answer.ok()).toBe(true)
}

/** Has the stand-in release feed name the old release again. */
const withdrawNewRelease = async (page: Page, origin: string): Promise<void> => {
  const answer = await page.request.delete(`${origin}/stand-in/release`)
  expect(answer.ok()).toBe(true)
}

/**
 * The buttons of a page, native or by role, and the parts of a native one that hold text. A plugin
 * that is off is a row drawn at half strength, which WCAG leaves out as an inactive control. A switch
 * holds no text; its thumb on its track is measured in the kit's contrast stories.
 */
const BUTTONS =
  'button:not([role="switch"]), [role="button"]:not(.plugin-row--disabled), button span'

/** What a control is: a button, an option of a list or a row of a menu. */
const CONTROLS = 'button, [role="button"], [role="option"], [role="menuitem"]'

/** Reads the control and then reads it again with the pointer over it. */
const expectReadableAtRestAndHovered = async (control: Locator): Promise<void> => {
  await expectReadable(control, CONTROLS, 1)
  await control.hover()
  await expectReadable(control, CONTROLS, 1)
}

for (const mode of ['light', 'dark'] as const) {
  test.describe(`in the ${mode} half of the theme`, () => {
    test.beforeEach(async ({ request }) => {
      await writeSetting(request, ['appearance', 'mode'], mode)
    })

    test.afterEach(async ({ request }) => {
      await writeSetting(request, ['appearance', 'mode'], 'system')
    })

    test('reads the text of every button of the update popover, the Download button on the accent among them', async ({
      page,
      request,
    }) => {
      const origin = await getStandInsOrigin(request)
      await publishNewRelease(page, origin)
      try {
        await openWorkspace(page)
        const indicator = page.getByRole('button', { name: 'Application update available' })
        await expectReadableAtRestAndHovered(indicator)
        await indicator.click()
        const popover = page.getByRole('dialog', { name: 'Update available' })
        await expect(popover).toBeVisible()
        await expectReadable(popover, BUTTONS, 4)
        await expectReadableAtRestAndHovered(
          popover.getByRole('button', { name: 'Download', exact: true }),
        )
      } finally {
        await withdrawNewRelease(page, origin)
      }
    })

    test('reads the text of the Done button of the link inspector, at rest and hovered', async ({
      page,
    }) => {
      await openWorkspace(page)
      await page.locator('.plex-frame:not([data-moving])').waitFor({ state: 'attached' })
      const edgeHit = getGraph(page).locator('.plex__edge-hit').first()
      await expect(edgeHit).toBeAttached()
      await edgeHit.dispatchEvent('click', { clientX: 300, clientY: 300 })
      const doneButton = page.locator('.plex-link-popover__btn--done')
      await expect(doneButton).toBeVisible()
      await expectReadable(page.locator('.plex-link-popover'), BUTTONS, 2)
      await expectReadableAtRestAndHovered(doneButton)
    })

    test.describe('the quick link of a note in the plex', () => {
      let quay = ''
      let jetty = ''

      test.beforeEach(async ({ request }) => {
        quay = await createNoteWithBody(
          request,
          'Quay',
          'notes',
          '# Quay\n\nWhere the barges unload.\n',
        )
        jetty = await createNoteWithBody(
          request,
          'Jetty',
          'notes',
          '# Jetty\n\nWhere the ferry ties up.\n',
        )
      })

      test.afterEach(async ({ request }) => {
        await destroyNotes(request, [quay, jetty])
      })

      test('reads the text of the highlighted row, which is filled with the accent, and of the others', async ({
        page,
      }) => {
        await openWorkspace(page)
        await showInFiles(page, ['notes'])
        await getRow(page, quay).click({ button: 'right' })
        await page.getByRole('menuitem', { name: 'Show in plex', exact: true }).click()
        await expect(getTab(page, 'Quay')).toBeVisible()
        await getGraph(page).getByRole('img', { name: 'Quay, focus' }).hover()
        const handle = page.locator('.plex__handle').first()
        await handle.hover()
        await handle.click()
        const dialog = page.getByRole('dialog', { name: 'New thought' })
        await dialog.getByRole('combobox').fill('Jett')
        await expect(
          dialog.getByRole('option', { name: 'Jetty', exact: true }).first(),
        ).toBeVisible()
        const frame = dialog.locator('[data-numen-searchable-list]')
        await frame.evaluate((el) =>
          Promise.all(el.getAnimations({ subtree: true }).map((animation) => animation.finished)),
        )
        const highlighted = dialog.getByRole('option', { selected: true })
        await expect(highlighted).toHaveCount(1)
        await expectReadable(dialog, '[role="option"], [role="option"] span', 2)
        await expectReadableAtRestAndHovered(highlighted)
      })
    })

    test('reads the text of every button of the settings page and of a plugin page with its Choose buttons', async ({
      page,
    }) => {
      await openWorkspace(page)
      await runCommand(page, 'Settings', /^Settings$/)
      const settings = page.getByRole('tabpanel', { name: /Settings/ })
      await expect(settings).toBeVisible()
      await expectReadable(settings, BUTTONS, 3)
      await settings.getByRole('button', { name: /Antigravity Agent/ }).click()
      await expect(settings.locator('.plugin-detail__title')).toContainText('Antigravity Agent')
      const choose = settings.getByRole('button', { name: /Choose/ })
      await expect(choose.first()).toBeVisible()
      await expectReadable(settings, BUTTONS, 2)
      await expectReadableAtRestAndHovered(choose.first())
    })

    test('reads the text of the Destroy row of the palette that asks for a note’s title, the confirmation of removing it', async ({
      page,
      request,
    }) => {
      const title = createUniqueTitle('Ballast')
      const path = await createNote(request, title)
      try {
        await writeNote(request, path, 'Stones laid in the hold.\n')
        await openWorkspace(page)
        await openNote(page, path)
        await page.keyboard.press('Control+p')
        await getPaletteField(page).fill('Remove note')
        const remove = page.getByRole('option', { selected: true })
        await expect(remove).toContainText('Remove note')
        expect(
          listShortfalls(await measureContrasts(page.locator('body'), '[role="option"]')),
        ).toEqual([])
        await page.keyboard.press('Shift+Enter')
        await getPaletteField(page).fill(title)
        const destroy = page.getByRole('option', { name: /^Destroy “/ })
        await expect(destroy).not.toHaveAttribute('aria-disabled', 'true')
        await expectReadable(page.locator('body'), '[role="option"]', 1)
        await expectReadableAtRestAndHovered(destroy)
      } finally {
        await destroyNote(request, path)
      }
    })
  })
}
