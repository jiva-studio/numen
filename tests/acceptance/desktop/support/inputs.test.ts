import { chromium, type Browser, type Page } from 'playwright'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { recordInputs, type RecordedInput } from './inputs.js'

const SECRET = 'sk-invented-secret-value'
const PAGE = '<input id="plain" type="text" aria-label="Name"><input id="secret" type="password" aria-label="Key">'

describe('recordInputs', () => {
  let browser: Browser
  let page: Page
  let inputs: RecordedInput[]

  beforeAll(async () => {
    browser = await chromium.launch()
  })

  afterAll(async () => {
    await browser.close()
  })

  const start = async (): Promise<void> => {
    inputs = []
    page = await browser.newPage()
    await recordInputs(page, 0, inputs)
    await page.goto(`data:text/html,${encodeURIComponent(PAGE)}`)
  }

  it('records the text of a field of text', async () => {
    await start()
    await page.getByLabel('Name').fill('Milk')
    await vi.waitFor(() => expect(inputs.some((one) => one.kind === 'type' && one.text === 'Milk')).toBe(true))
  })

  it('records dots in place of what fills a password field', async () => {
    await start()
    await page.getByLabel('Key').fill(SECRET)
    await vi.waitFor(() => expect(inputs.some((one) => one.kind === 'type')).toBe(true))
    const typed = inputs.filter((one) => one.kind === 'type')
    expect(typed.every((one) => one.text === '••••')).toBe(true)
    expect(JSON.stringify(inputs)).not.toContain(SECRET)
  })

  it('records no character pressed in a password field', async () => {
    await start()
    await page.getByLabel('Key').click()
    await page.keyboard.type('abc')
    await page.keyboard.press('Enter')
    const getKeys = () => inputs.filter((one) => one.kind === 'key').map((one) => one.key)
    await vi.waitFor(() => expect(getKeys()).toHaveLength(4))
    expect(getKeys()).toEqual(['•', '•', '•', 'Enter'])
    expect(JSON.stringify(inputs)).not.toMatch(/"(text|key)":"[abc]"/)
  })
})
