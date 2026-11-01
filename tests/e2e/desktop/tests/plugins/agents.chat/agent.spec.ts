import { test, expect, type Page } from '../../support/test.js'
import { DELAYED_PROMPT, STAND_IN_REPLY } from '../../../scripts/stand-ins.mjs'

test.describe('Agent Plugin and Interface', () => {
  test('opens agent panel and views prompt conversation interface', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Agent tab
    await page.getByText('Agent').first().click()

    // Verify Agent panel interface
    const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
    await expect(input).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('No messages yet').first()).toBeVisible()
  })

  test('inspects agent model and provider selector button', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Agent tab
    await page.getByText('Agent').first().click()

    // Verify model selector if available, or verify composer layout
    const providerBtn = page.locator('.model-selector__pill')
    if ((await providerBtn.count()) > 0) {
      await expect(providerBtn.first()).toBeVisible({ timeout: 10_000 })
      await expect(providerBtn.first()).toBeEnabled()
    } else {
      await expect(page.locator('.agent__composer').first()).toBeVisible({ timeout: 10_000 })
    }
  })

  test('inputs query into agent textbox and triggers send submission', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Agent tab
    await page.getByText('Agent').first().click()

    const input = page.getByRole('textbox', { name: 'Ask about the vault' }).first()
    await expect(input).toBeVisible({ timeout: 10_000 })

    // Send button starts disabled when input is empty
    const sendBtn = page.getByRole('button', { name: 'Send' }).first()
    await expect(sendBtn).toBeDisabled()

    // Fill query
    await input.fill('Summarize thermodynamics principles')
    await expect(sendBtn).toBeEnabled()

    // Click send
    await sendBtn.click()

    // Verify query is submitted into the conversation thread
    await expect(page.getByText('Summarize thermodynamics principles').first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('opens note tab without 404 navigation when clicking a relative note link in agent reply', async ({
    page,
  }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()
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
  await expect(page.locator('text=Ready')).toBeVisible()
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
    await page.waitForTimeout(TRANSITIONS_MS)

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
    await page.waitForTimeout(TRANSITIONS_MS)

    expect(await readDiscValues(page)).toEqual([
      'Send:enabled',
      'click',
      'Stop the agent:enabled',
      'click',
      'Send:disabled',
    ])
  })
})
