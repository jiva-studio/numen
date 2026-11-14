import { test, expect, type Page } from '../../support/test.js'
import {
  TOOL_NOTE_PATH,
  TOOL_NOTE_TEXT,
  TOOL_PROMPT,
} from '../../../scripts/stand-ins.mjs'
import { openWorkspace } from '../../support/workspace.js'
import { hasVaultFile } from '../../support/media.js'
import { destroyNote, readVaultFile } from '../../support/vault.js'

const openAgentPanel = async (page: Page) => {
  await openWorkspace(page)
  await page.getByText('Agent').first().click()
  const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
  await expect(input).toBeVisible()
  return input
}

test.describe('Agent Plugin and Interface', () => {
  test('writes the note a tool call of the model asks for, and shows the call in the turn', async ({
    page,
    request,
  }) => {
    expect(hasVaultFile(TOOL_NOTE_PATH)).toBe(false)
    try {
      const input = await openAgentPanel(page)
      await input.fill(TOOL_PROMPT)
      // The step is taken down when the model is asked again, so every text it takes is recorded.
      await page.evaluate(() => {
        const texts = new Set<string>()
        window.toolCallTexts = texts
        const read = () =>
          document.querySelectorAll('.tool-call').forEach((one) => texts.add(one.textContent ?? ''))
        new MutationObserver(read).observe(document.body, {
          subtree: true,
          childList: true,
          characterData: true,
        })
      })
      await page.getByRole('button', { name: 'Send' }).first().click()

      await expect(page.getByText(/The tide table is written\./)).toBeVisible({ timeout: 20_000 })
      expect(await readVaultFile(request, TOOL_NOTE_PATH)).toBe(TOOL_NOTE_TEXT)
      const texts = await page.evaluate(() => [...(window.toolCallTexts ?? [])])
      expect(texts).toContain(`Write Note${TOOL_NOTE_PATH}`)
    } finally {
      await destroyNote(request, TOOL_NOTE_PATH)
    }
  })
})

declare global {
  interface Window {
    toolCallTexts?: Set<string>
  }
}
