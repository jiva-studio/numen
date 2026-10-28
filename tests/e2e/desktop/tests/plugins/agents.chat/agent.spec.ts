import { test, expect } from '../../support/test.js'

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
