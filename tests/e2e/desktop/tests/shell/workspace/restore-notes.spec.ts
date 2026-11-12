import { test, expect, readSavedStates, type Page } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  getNotePanel,
  getNoteTab,
  getNoteText,
  listFilePaths,
  openNote,
  writeNote,
} from '../../support/vault.js'

const NOTES_PLUGIN_MANIFEST = '**/md.numen.notes/manifest.json'
const WATCH_QUIT = '**/numen.v1.WindowService/WatchQuit'

const flushFrames = (): Buffer => {
  const frame = (flags: number, message: object): Buffer => {
    const json = Buffer.from(JSON.stringify(message))
    const header = Buffer.alloc(5)
    header.writeUInt8(flags, 0)
    header.writeUInt32BE(json.length, 1)
    return Buffer.concat([header, json])
  }
  return Buffer.concat([
    frame(0, { token: 'flush-test' }),
    frame(0, { token: 'flush-test', flush: true }),
    frame(2, {}),
  ])
}

const expectSavedPaths = async (page: Page, paths: readonly string[]): Promise<void> => {
  await expect
    .poll(async () => {
      const saved = [...(await readSavedStates(page.request)).keys()]
      return paths.every((one) => saved.includes(one)) && saved.every((one) => !one.includes(':'))
    })
    .toBe(true)
}

const expectRestored = async (page: Page, notes: readonly { path: string; text: string }[]) => {
  for (const { path, text } of notes) {
    await expect(getNoteTab(page, path)).toBeVisible()
    await getNoteTab(page, path).click()
    await expect(getNoteText(page, path)).toContainText(text)
    await expect(getNotePanel(page, path).getByRole('alert')).toHaveCount(0)
  }
}

const expectNoColonFile = async (page: Page, folders: readonly string[]): Promise<void> => {
  for (const folder of folders) {
    expect(
      (await listFilePaths(page.request, folder)).filter((one) => one.includes(':')),
    ).toStrictEqual([])
  }
}

test.describe('The note tabs of the window after a restart', () => {
  test.describe.configure({ timeout: 120_000 })

  test('keep the saved path of each note when the application asks the window to flush on quit', async ({
    page,
    request,
  }) => {
    const path = await createNote(request, createUniqueTitle('dd'))
    try {
      await writeNote(request, path, 'Text of the flushed note.\n')
      await openWorkspace(page)
      await openNote(page, path)
      await expectSavedPaths(page, [path])

      await page.route(WATCH_QUIT, (route) =>
        route.fulfill({ contentType: 'application/connect+json', body: flushFrames() }),
      )
      const reported = page.waitForRequest('**/numen.v1.WindowService/ReportFlush')
      await page.reload()
      await reported
      await expect(getNoteTab(page, path)).toBeVisible()
      // The window writes its tabs in the background after the store closed its notes.
      await new Promise((resolve) => setTimeout(resolve, 1_000))
      await page.unroute(WATCH_QUIT)
      await expectSavedPaths(page, [path])
      await page.reload()

      await expectRestored(page, [{ path, text: 'Text of the flushed note.' }])
      await expectNoColonFile(page, ['', 'notes'])
    } finally {
      await destroyNote(request, path)
    }
  })

  test('show the note when its plugin registers after the window opened the tabs', async ({
    page,
    request,
  }) => {
    const path = await createNote(request, createUniqueTitle('dd'))
    try {
      await writeNote(request, path, 'Text of the late note.\n')
      await openWorkspace(page)
      await openNote(page, path)
      await expectSavedPaths(page, [path])

      await page.route(NOTES_PLUGIN_MANIFEST, async (route) => {
        await new Promise((resolve) => setTimeout(resolve, 4_000))
        await route.continue()
      })
      await page.reload()

      await expectRestored(page, [{ path, text: 'Text of the late note.' }])
      await expectSavedPaths(page, [path])
      await expectNoColonFile(page, ['', 'notes'])
    } finally {
      await page.unroute(NOTES_PLUGIN_MANIFEST)
      await destroyNote(request, path)
    }
  })
})
