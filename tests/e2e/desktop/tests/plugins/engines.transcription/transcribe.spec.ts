import { test, expect, type Locator, type Page } from '../../support/test.js'
import {
  findPassages,
  openVaultFile,
  openWorkspace,
  runCommand,
  runCommandOverItem,
  waitForPassage,
} from '../../support/workspace.js'
import { copyVaultFile, deleteVaultFiles } from '../../support/media.js'
import { createUniqueTitle } from '../../support/vault.js'

/** What the mock transcriber of `-debug-mock-ml` returns for every recording. */
const MOCK_TRANSCRIPT = 'Mock transcribed text for automated test.'

/** Words of the text the mock recogniser of `-debug-mock-ml` reads on every page. */
const MOCK_RECOGNISED_WORDS = 'recovers kinetic energy'

/** What the mock proofreader of `-debug-mock-ml` adds to the end of every line it puts right. */
const PROOFREAD_MARK = '[proofread]'

/**
 * Waits until the search for the words lists a passage of `present` and none of `gone`. The
 * passage of `present` shows the list is drawn for the words, so the missing one is missing from
 * an answer and not from a list still loading.
 */
const expectPassageGone = async (
  page: Page,
  query: string,
  present: string,
  gone: string,
): Promise<void> => {
  await expect(async () => {
    await expect((await findPassages(page, query, present)).first()).toBeVisible({
      timeout: 3_000,
    })
    await expect(await findPassages(page, query, gone)).toHaveCount(0)
  }).toPass({ timeout: 90_000 })
  await page.keyboard.press('Escape')
}

/**
 * Waits until no task recognises the file. The plugin recognises a scan on its own when it opens,
 * and the Recognise command starts a second run: the words of the first run are found while the
 * second still runs. Deleting the recognition then takes away the file the second run is about to
 * report, and that run ends as a failed task that stays in the status bar.
 */
const expectNoRecognitionRunning = async (page: Page, name: string): Promise<void> => {
  await expect(page.locator('.status-bar__summary')).not.toHaveText(/^[1-9]\d* active$/)
  await page.locator('.status-bar__trigger').click()
  await expect(page.locator('.task-item').filter({ hasText: name })).toHaveCount(0)
  await page.keyboard.press('Escape')
}

/** Opens a recording of the vault and has it transcribed through the palette. */
const transcribe = async (page: Page, name: string): Promise<Locator> => {
  await openWorkspace(page)
  const tab = await openVaultFile(page, 'assets', name)
  await runCommand(page, 'Transcribe', /^Transcribe/)
  await expect(tab.getByText(MOCK_TRANSCRIPT)).toBeVisible({ timeout: 30_000 })
  return tab
}

/** Types words after the transcript of the open recording and saves them as an edit of it. */
const appendToTranscript = async (page: Page, tab: Locator, words: string): Promise<void> => {
  await tab.getByRole('textbox', { name: 'Transcript' }).click()
  await page.keyboard.press('Control+End')
  await page.keyboard.type(words)
  const saved = page.waitForResponse((response) =>
    response.url().includes('/numen.v1.TextService/CreateTextEdit'),
  )
  await page.keyboard.press('Control+s')
  expect((await saved).ok()).toBe(true)
}

test.describe('the text versions of a file', () => {
  test.describe.configure({ timeout: 240_000 })

  const copies: string[] = []

  test.afterEach(() => {
    deleteVaultFiles(...copies.splice(0).map((name) => `assets/${name}`))
  })

  /** Copies a fixture of `assets/` under a name of this test alone, and returns the name. */
  const copyAsset = (fixture: string, stem: string): string => {
    const name = `${createUniqueTitle(stem).replace(/ /g, '-')}${fixture.slice(fixture.lastIndexOf('.'))}`
    copyVaultFile(`assets/${fixture}`, `assets/${name}`)
    copies.push(name)
    return name
  }

  test('Delete transcript takes the transcript away, and the search stops finding it', async ({
    page,
  }) => {
    const control = copyAsset('lecture.wav', 'harbour-control')
    const deleted = copyAsset('lecture.wav', 'harbour-deleted')
    const controlTab = await transcribe(page, control)
    // The search lists a passage once for words the files share, so the control has words of its own.
    await appendToTranscript(page, controlTab, ' The harbour master logged the tide.')
    let tab = await transcribe(page, deleted)
    await waitForPassage(page, 'Mock transcribed', `assets/${control}`)
    await page.keyboard.press('Escape')
    await waitForPassage(page, 'Mock transcribed', `assets/${deleted}`)
    await page.keyboard.press('Escape')

    await runCommandOverItem(page, 'Delete transcript', /^Delete transcript$/, /^transcript/)

    await expect(tab.getByRole('button', { name: 'Transcribe' })).toBeVisible()
    await expect(tab.getByText(MOCK_TRANSCRIPT)).toHaveCount(0)
    await expectPassageGone(page, 'Mock transcribed', `assets/${control}`, `assets/${deleted}`)

    await openWorkspace(page)
    tab = await openVaultFile(page, 'assets', deleted)
    await expect(tab.getByRole('button', { name: 'Transcribe' })).toBeVisible()
    await expect(tab.getByText(MOCK_TRANSCRIPT)).toHaveCount(0)
  })

  test('Delete recognition takes the text of a scan away, and the search stops finding it', async ({
    page,
  }) => {
    const control = copyAsset('regenerative-braking.pdf', 'braking-control')
    const deleted = copyAsset('regenerative-braking.pdf', 'braking-deleted')
    for (const name of [control, deleted]) {
      await openWorkspace(page)
      await openVaultFile(page, 'assets', name)
      await runCommand(page, 'Recognise', /^Recognise/)
      await waitForPassage(page, MOCK_RECOGNISED_WORDS, `assets/${name}`)
      await page.keyboard.press('Escape')
      await expectNoRecognitionRunning(page, name)
    }

    await runCommandOverItem(page, 'Delete recognition', /^Delete recognition$/, /^recognition/)

    await expectPassageGone(page, MOCK_RECOGNISED_WORDS, `assets/${control}`, `assets/${deleted}`)
  })

  test('Delete proofreading takes the proofread version away and keeps the transcript', async ({
    page,
  }) => {
    const name = copyAsset('lecture.wav', 'harbour-proofread')
    let tab = await transcribe(page, name)
    await runCommandOverItem(page, 'Proofread', /^Proofread/, /\(live\)$/)
    await expect(tab.getByText(/Mock transcribed text.*\[proofread\]/)).toBeVisible({
      timeout: 90_000,
    })

    // The proofread version is the live one once it is made.
    await runCommandOverItem(page, 'Delete proofreading', /^Delete proofreading$/, /\(live\)$/)

    await expect(tab.getByText(MOCK_TRANSCRIPT)).toBeVisible()
    await expect(tab.getByText(PROOFREAD_MARK)).toHaveCount(0)
    await expect(async () => {
      const passages = await findPassages(page, 'Mock transcribed', `assets/${name}`)
      await expect(passages.first()).toBeVisible({ timeout: 10_000 })
      await expect(passages.filter({ hasText: PROOFREAD_MARK })).toHaveCount(0)
    }).toPass({ timeout: 90_000 })
    await page.keyboard.press('Escape')

    await openWorkspace(page)
    tab = await openVaultFile(page, 'assets', name)
    await expect(tab.getByText(MOCK_TRANSCRIPT)).toBeVisible()
    await expect(tab.getByText(PROOFREAD_MARK)).toHaveCount(0)
  })
})
