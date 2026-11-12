import { test, expect } from '../../support/test.js'
import { openVaultFile, openWorkspace } from '../../support/workspace.js'
import { getRow, readWidth, showInFiles } from '../../support/layout.js'
import { writeSetting } from '../../support/vault.js'

const NOTES_ID = 'md.numen.notes'
const PDF_ID = 'md.numen.readers.pdf'

test.describe('Plugin settings take effect', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, [NOTES_ID, 'lineNumbers'], false)
    await writeSetting(request, [NOTES_ID, 'softWrap'], true)
    await writeSetting(request, [PDF_ID, 'defaultZoom'], 'page')
  })

  test('draws line numbers and no soft wrap in a note once the notes settings say so', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, 'notes/Entropy.md').press('Enter')
    const editor = page.getByRole('tabpanel', { name: /Entropy/ }).locator('.cm-editor')
    await expect(editor).toBeVisible()
    await expect(editor.locator('.cm-lineNumbers')).toHaveCount(0)
    await expect(editor.locator('.cm-lineWrapping')).toHaveCount(1)

    await writeSetting(request, [NOTES_ID, 'lineNumbers'], true)
    await writeSetting(request, [NOTES_ID, 'softWrap'], false)

    await expect(editor.locator('.cm-lineNumbers')).toHaveCount(1)
    await expect(editor.locator('.cm-lineWrapping')).toHaveCount(0)
  })

  for (const [fit, getExpected] of [
    ['width', (viewport: number) => viewport - 32],
    ['100%', () => (612 * 96) / 72],
  ] as const) {
    test(`draws a new PDF at the ${fit} default zoom`, async ({ page, request }) => {
      await writeSetting(request, [PDF_ID, 'defaultZoom'], fit)
      await openWorkspace(page)
      await openVaultFile(page, 'assets', 'pages.pdf')
      const first = page.locator('.reader__page[data-page="0"]')
      await expect(first).toBeAttached({ timeout: 20_000 })
      const viewport = page.locator('.reader__viewport')

      await expect
        .poll(async () =>
          Math.abs((await readWidth(first)) - getExpected(await readWidth(viewport))),
        )
        .toBeLessThan(24)
    })
  }
})
