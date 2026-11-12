import { test, expect, type Page } from '../../support/test.js'
import {
  HELD_PROMPT,
  STAND_IN_REPLY,
  TOOL_NOTE_PATH,
  TOOL_NOTE_TEXT,
  TOOL_PROMPT,
} from '../../../scripts/stand-ins.mjs'
import { openWorkspace, waitForWindow } from '../../support/workspace.js'
import { getStandInsOrigin, hasVaultFile } from '../../support/media.js'
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
      expect(texts).toContain('Write Note')
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

const openAgent = async (page: Page) => {
  await page.goto('/')
  await waitForWindow(page)
  await page.getByText('Agent').first().click()
  const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
  await expect(input).toBeVisible({ timeout: 10_000 })
  return input
}

test.describe('the send/stop disc', () => {
  test('shows send, then stop until the answer settles, then send', async ({ page, request }) => {
    const origin = await getStandInsOrigin(request)
    expect((await request.put(`${origin}/stand-in/hold`)).ok()).toBe(true)
    try {
      const input = await openAgent(page)
      await input.fill(`${HELD_PROMPT}: which pear trees grow in the east orchard?`)
      const send = page.getByRole('button', { name: 'Send' }).first()
      await expect(send).toBeEnabled()

      await send.click()
      await expect(page.getByRole('button', { name: 'Stop the agent' }).first()).toBeEnabled()
      await expect(page.getByText(STAND_IN_REPLY)).toHaveCount(0)

      expect((await request.delete(`${origin}/stand-in/hold`)).ok()).toBe(true)
      await expect(page.getByText(STAND_IN_REPLY).first()).toBeVisible({ timeout: 20_000 })
      await expect(page.getByRole('button', { name: 'Stop the agent' })).toHaveCount(0)
      await expect(send).toBeDisabled()
    } finally {
      await request.delete(`${origin}/stand-in/hold`)
    }
  })
})
