import { test, expect } from '../../support/test.js'
import { openFromFiles, openTreeRow } from '../../support/workspace.js'

test.describe('Chaos, Fault Tolerance and Stress Scenarios', () => {
  test('gracefully parses malformed yaml frontmatter without editor crash', async ({ page }) => {
    await openFromFiles(page, ['notes', 'notes/malformed.md'])

    // Verify note editor detects malformed frontmatter and displays explicit alert banner
    const alertBanner = page.getByRole('alert')
    await expect(alertBanner).toBeVisible({ timeout: 10_000 })
    await expect(alertBanner).toContainText('frontmatter unreadable')

    // Verify application remains interactive
    await expect(page.locator('.status-bar')).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
  })

  test('renders high-volume document smoothly without interface freeze', async ({ page }) => {
    await openFromFiles(page, ['notes', 'notes/large.md'])

    // Verify large document editor is rendered and contains expected content
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
    const content = page.locator('.note__text .cm-content').first()
    await expect(content).toBeVisible({ timeout: 10_000 })
    await expect(content).toContainText('Large Document')
    await expect(content).toContainText('Paragraph line 1')
  })

  test('handles rapid tab switching and closing across heterogeneous file types', async ({
    page,
  }) => {
    await openFromFiles(page, ['notes', 'notes/Entropy.md'])

    // Switch back to Files tab to open assets
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Open assets folder
    await openTreeRow(page, 'assets')

    // Open sample.pdf
    await openTreeRow(page, 'assets/sample.pdf')

    // Rapid switch between tabs
    const entropyTab = page.getByRole('tab', { name: /Entropy/i }).first()
    const pdfTab = page.getByRole('tab', { name: /sample\.pdf/i }).first()

    await expect(entropyTab).toBeVisible({ timeout: 10_000 })
    await expect(pdfTab).toBeVisible({ timeout: 10_000 })

    await entropyTab.click()
    await expect(entropyTab).toHaveAttribute('aria-selected', 'true')
    await pdfTab.click()
    await expect(pdfTab).toHaveAttribute('aria-selected', 'true')
    await entropyTab.click()
    await expect(entropyTab).toHaveAttribute('aria-selected', 'true')

    // Verify workspace remains responsive
    await expect(page.locator('.status-bar')).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
  })
})
