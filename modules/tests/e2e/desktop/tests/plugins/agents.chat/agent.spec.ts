import { test, expect } from 'playwright/test'

test.describe('Agent Plugin and Interface', () => {
  test('opens agent panel and views prompt conversation interface', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Agent tab
    await page.getByText('Agent').first().click()

    // Verify Agent panel interface
    const input = page.getByRole('textbox', { name: 'Ask about the vault' })
    await expect(input).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Nothing said yet')).toBeVisible()
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
      await expect(page.locator('.agent__composer')).toBeVisible({ timeout: 10_000 })
    }
  })

  test('inputs query into agent textbox and triggers send submission', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Agent tab
    await page.getByText('Agent').first().click()

    const input = page.getByRole('textbox', { name: 'Ask about the vault' })
    await expect(input).toBeVisible({ timeout: 10_000 })

    // Send button starts disabled when input is empty
    const sendBtn = page.getByRole('button', { name: 'Send' })
    await expect(sendBtn).toBeDisabled()

    // Fill query
    await input.fill('Summarize thermodynamics principles')
    await expect(sendBtn).toBeEnabled()

    // Click send
    await sendBtn.click()

    // Verify query is submitted into the conversation thread
    await expect(page.getByText('Summarize thermodynamics principles')).toBeVisible({
      timeout: 10_000,
    })
  })
})
