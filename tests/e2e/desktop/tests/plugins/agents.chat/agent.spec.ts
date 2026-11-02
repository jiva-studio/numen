import { test, expect, type Page } from '../../support/test.js'
import {
  DELAYED_PROMPT,
  STAND_IN_REPLY,
  TOOL_NOTE_PATH,
  TOOL_NOTE_TEXT,
  TOOL_PROMPT,
} from '../../../scripts/stand-ins.mjs'
import { observeFor } from '../../support/browser.js'
import { openVaultFile, openWorkspace, runCommand, waitForWindow } from '../../support/workspace.js'
import { hasVaultFile } from '../../support/media.js'
import { callRPC, readVaultFile } from '../../support/vault.js'

/** Opens the agent panel in a freshly loaded window, and gives back its composer. */
const openAgentPanel = async (page: Page) => {
  await openWorkspace(page)
  await page.getByText('Agent').first().click()
  const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
  await expect(input).toBeVisible()
  return input
}

test.describe('Agent Plugin and Interface', () => {
  test('Ask the agent on a note puts the note path in the composer', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'notes', 'Entropy.md')

    await runCommand(page, 'Ask the agent', /^Ask the agent/)

    const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
    await expect(input).toBeVisible()
    await expect(input).toHaveValue('notes/Entropy.md — ')
  })

  test('the model selector names the stand-in model the settings choose', async ({ page }) => {
    await openAgentPanel(page)

    const pill = page.locator('.model-selector__pill').first()
    await expect(pill).toHaveText(/ · Stand-in model$/)
    await pill.click()
    await expect(
      page
        .getByRole('menu', { name: 'Model' })
        .getByRole('menuitemradio', { name: 'Stand-in model' }),
    ).toHaveAttribute('aria-checked', 'true')
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
      await callRPC(request, 'FileService/RemoveFile', { path: TOOL_NOTE_PATH, destroy: true })
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
    discValues?: string[]
    toolCallTexts?: Set<string>
  }
}

/**
 * Starts recording in the page every value the send/stop disc takes, `<label>:<enabled|disabled>`,
 * on every mutation and every animation frame. A click is recorded as `click`, and the disc is
 * read again in the first frame after it.
 */
const recordDisc = (page: Page) =>
  page.evaluate(() => {
    const values: string[] = []
    window.discValues = values
    const read = () => {
      const buttons = [...document.querySelectorAll('.agent__composer .disc button')]
      const value =
        buttons
          .map((button) => {
            const state = button.hasAttribute('disabled') ? 'disabled' : 'enabled'
            return `${button.getAttribute('aria-label') ?? ''}:${state}`
          })
          .join('+') || 'none'
      if (values.at(-1) !== value) values.push(value)
    }
    read()
    new MutationObserver(read).observe(document.body, {
      subtree: true,
      childList: true,
      attributes: true,
    })
    const readEachFrame = () => {
      read()
      requestAnimationFrame(readEachFrame)
    }
    requestAnimationFrame(readEachFrame)
    document.addEventListener(
      'click',
      (event) => {
        if (!(event.target instanceof Element) || !event.target.closest('.disc')) return
        values.push('click')
        requestAnimationFrame(read)
      },
      { capture: true },
    )
  })

const readDiscValues = (page: Page) => page.evaluate(() => window.discValues ?? [])

/** Longer than any transition of the disc. */
const TRANSITIONS_MS = 500

const openAgent = async (page: Page) => {
  await page.goto('/')
  await waitForWindow(page)
  await page.getByText('Agent').first().click()
  const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
  await expect(input).toBeVisible({ timeout: 10_000 })
  return input
}

test.describe('the send/stop disc', () => {
  test('shows send, then stop until the answer settles, then send', async ({ page }) => {
    const input = await openAgent(page)
    await input.fill('Which pear trees grow in the east orchard?')
    await recordDisc(page)

    await page.getByRole('button', { name: 'Send' }).first().click()
    await expect(page.getByText(STAND_IN_REPLY).first()).toBeVisible({ timeout: 20_000 })
    await expect(page.getByRole('button', { name: 'Send' }).first()).toBeDisabled()
    // The disc is recorded for as long as its transitions last, and the recording is the result.
    await observeFor(page, TRANSITIONS_MS)

    expect(await readDiscValues(page)).toEqual([
      'Send:enabled',
      'click',
      'Stop the agent:enabled',
      'Send:disabled',
    ])
  })

  test('shows send once stop is pressed in the middle of the answer', async ({ page }) => {
    const input = await openAgent(page)
    await input.fill(`${DELAYED_PROMPT}: count the apple trees by the gate`)
    await recordDisc(page)

    await page.getByRole('button', { name: 'Send' }).first().click()
    const stop = page.getByRole('button', { name: 'Stop the agent' }).first()
    await expect(stop).toBeEnabled()
    await stop.click()
    await expect(page.getByRole('button', { name: 'Send' }).first()).toBeDisabled()
    // The disc is recorded for as long as its transitions last, and the recording is the result.
    await observeFor(page, TRANSITIONS_MS)

    expect(await readDiscValues(page)).toEqual([
      'Send:enabled',
      'click',
      'Stop the agent:enabled',
      'click',
      'Send:disabled',
    ])
  })
})
