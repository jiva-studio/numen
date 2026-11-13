import { mkdir, writeFile } from 'node:fs/promises'
import { dirname } from 'node:path'
import { test, expect, type Page } from '../../support/test.js'
import { observeFor } from '../../support/browser.js'
import type { JsonValue } from '../../support/json.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'

const FIRST = { id: 'md.numen.engines.ocr', name: 'Document OCR' }
const SECOND = { id: 'md.numen.agents.acp', name: 'ACP Agents' }

const createModelFile = async (): Promise<string> => {
  const path = test.info().outputPath('ocr', 'recognition.onnx')
  await mkdir(dirname(path), { recursive: true })
  await writeFile(path, 'model')
  return path
}

/** The slowest first render of a plugin page the scenario accepts, in milliseconds. */
const FIRST_RENDER_LIMIT_MS = 1_000

declare global {
  interface Window {
    shownPaths?: string[]
    firstRenderMs?: number
  }
}

const writeSetting = async (page: Page, path: string[], value: JsonValue): Promise<void> => {
  const answer = await page.request.post('/numen.v1.SettingsService/WriteSettings', {
    data: { settings: [{ at: path, value: JSON.stringify(value) }] },
    headers: { 'Content-Type': 'application/json' },
  })
  expect(answer.ok()).toBe(true)
}

const recordPage = async (page: Page, pluginName: string): Promise<void> => {
  await page.evaluate((name) => {
    const startedAt = performance.now()
    window.shownPaths = []
    delete window.firstRenderMs
    const record = () => {
      for (const one of document.querySelectorAll('[data-testid="setting-path"]')) {
        window.shownPaths?.push(one.getAttribute('title') ?? '')
      }
      const titles = Array.from(document.querySelectorAll('h2'), (one) => one.textContent ?? '')
      if (window.firstRenderMs === undefined && titles.includes(name)) {
        window.firstRenderMs = performance.now() - startedAt
      }
    }
    new MutationObserver(record).observe(document.body, {
      subtree: true,
      childList: true,
      characterData: true,
      attributes: true,
      attributeFilter: ['title'],
    })
  }, pluginName)
}

test.describe('The page of a plugin', () => {
  test.afterEach(async ({ page }) => {
    await writeSetting(page, [FIRST.id, 'customModelPath'], '')
    await writeSetting(page, [FIRST.id, 'model'], 'pp-ocrv6-tiny')
  })

  test('shows none of the values of the plugin page opened before it', async ({ page }) => {
    const firstPath = await createModelFile()
    await writeSetting(page, [FIRST.id, 'model'], 'custom')
    await writeSetting(page, [FIRST.id, 'customModelPath'], firstPath)
    await openWorkspace(page)
    await runCommand(page, 'Open settings', /^Open settings$/)
    const settings = page.getByRole('tabpanel', { name: /^(Linked tab )?Settings Close Settings$/ })

    await settings.getByRole('button', { name: new RegExp(FIRST.name) }).click()
    await expect(settings.getByTestId('setting-path')).toHaveAttribute('title', firstPath)
    await settings.getByRole('button', { name: 'Back', exact: true }).click()

    await recordPage(page, SECOND.name)
    await settings.getByRole('button', { name: new RegExp(SECOND.name) }).click()
    await expect(settings.getByRole('heading', { level: 2, name: SECOND.name })).toBeVisible()
    await expect(settings.getByText('Request timeout')).toBeVisible()
    // The settings are read again after the page opens.
    await observeFor(page, 2_000)

    const shownPaths = await page.evaluate(() => window.shownPaths ?? [])
    const firstRenderMs = await page.evaluate(() => window.firstRenderMs ?? Infinity)
    test.info().annotations.push({
      type: 'first render',
      description: `${SECOND.name} page: ${Math.round(firstRenderMs)} ms`,
    })
    expect(shownPaths).toEqual([])
    expect(firstRenderMs).toBeLessThan(FIRST_RENDER_LIMIT_MS)
  })
})
