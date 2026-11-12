import { test, expect, type Page } from '../../support/test.js'
import {
  DELAYED_PROMPT,
  STAND_IN_MODEL,
  HELD_PROMPT,
  STAND_IN_REPLY,
  TOOL_NOTE_PATH,
  TOOL_NOTE_TEXT,
  TOOL_PROMPT,
} from '../../../scripts/stand-ins.mjs'
import { openVaultFile, openWorkspace, runCommand, waitForWindow } from '../../support/workspace.js'
import { getStandInsOrigin, hasVaultFile } from '../../support/media.js'
import { expectSearchableList } from '../../support/searchable-list.js'
import { destroyNote, readVaultFile } from '../../support/vault.js'

const openAgentPanel = async (page: Page) => {
  await openWorkspace(page)
  await page.getByText('Agent').first().click()
  const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
  await expect(input).toBeVisible()
  return input
}

test.describe('Agent Plugin and Interface', () => {
  test('Ask agent about note on a note puts the note path in the composer', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'notes', 'Entropy.md')

    await runCommand(page, 'Ask agent about note', /^Ask agent about note/)

    const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
    await expect(input).toBeVisible()
    await expect(input).toHaveValue('notes/Entropy.md — ')
  })

  test('the model selector names the stand-in model the settings choose', async ({ page }) => {
    await openAgentPanel(page)

    const pill = page.locator('.model-selector__pill').first()
    await expect(pill).toHaveText(/ · stand-in\/model$/)
    await pill.click()
    await expectSearchableList(page)
    await expect(
      page.getByRole('listbox', { name: 'Model' }).getByRole('option', { name: STAND_IN_MODEL }),
    ).toHaveAttribute('aria-current', 'true')
  })

  test('the model selector narrows its models by the words typed and chooses the one the keyboard is on', async ({
    page,
  }) => {
    await openAgentPanel(page)
    await page.locator('.model-selector__pill').first().click()
    const search = page.getByRole('combobox', { name: 'Search models' })
    const models = page.getByRole('listbox', { name: 'Model' })

    await search.fill('no such model')
    await expect(models).toHaveCount(0)
    await expect(page.locator('[data-numen-searchable-list]').getByRole('status')).toHaveText(
      'No models found',
    )

    await search.fill('stand-in')
    await expect(models.getByRole('option')).toHaveText([STAND_IN_MODEL])
    await search.press('Enter')
    await expect(search).toHaveCount(0)
    await expect(page.locator('.model-selector__pill').first()).toHaveText(/ · stand-in\/model$/)
  })

  test('sends the question and shows the answer of the chat API under it', async ({ page }) => {
    const input = await openAgentPanel(page)
    await expect(page.getByText('No messages yet').first()).toBeVisible()
    const send = page.getByRole('button', { name: 'Send' }).first()
    await expect(send).toBeDisabled()

    await input.fill('Summarize thermodynamics principles')
    await send.click()

    await expect(page.getByText('Summarize thermodynamics principles').first()).toBeVisible()
    await expect(page.getByText(STAND_IN_REPLY).first()).toBeVisible({ timeout: 20_000 })
    await expect(input).toHaveValue('')
    await expect(page.getByText('No messages yet')).toHaveCount(0)
  })

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

  test('opens note tab without 404 navigation when clicking a relative note link in agent reply', async ({
    page,
  }) => {
    await page.goto('/')

    await waitForWindow(page)
    await page.getByText('Agent').first().click()

    const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
    await expect(input).toBeVisible({ timeout: 10_000 })

    await input.fill('Link to Entropy')
    await page.getByRole('button', { name: 'Send' }).first().click()

    const noteLink = page.getByRole('link', { name: 'Entropy', exact: true }).first()
    await expect(noteLink).toBeVisible({ timeout: 10_000 })
    await noteLink.click()

    await expect(page.getByRole('tab', { name: /Entropy/i }).first()).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByText('404 page not found')).toHaveCount(0)
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

  test('shows send once stop is pressed in the middle of the answer', async ({ page }) => {
    const input = await openAgent(page)
    await input.fill(`${DELAYED_PROMPT}: count the apple trees by the gate`)

    await page.getByRole('button', { name: 'Send' }).first().click()
    const stop = page.getByRole('button', { name: 'Stop the agent' }).first()
    await expect(stop).toBeEnabled()
    await stop.click()
    await expect(page.getByRole('button', { name: 'Stop the agent' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Send' }).first()).toBeDisabled()
  })
})
