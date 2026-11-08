import fs from 'node:fs'
import path from 'node:path'
import { expect, test, type Page } from '../tests/support/certificate.js'
import { STAND_IN_REPLY } from '../scripts/stand-ins.mjs'

const vault = process.env.NUMEN_FLASHCARDS_VAULT ?? ''
const answers = path.join(vault, '.numen', 'md.numen.views.flashcards.studio')

/** One line of the review log: an answer of a card, or the taking back of one. */
interface ReviewLine {
  readonly v: number
  readonly id: string
  readonly card?: string
  readonly template?: string
  readonly rating?: number
  readonly undo?: string
}

/** The ratings the review log writes for the four answers. */
const RATING = { Again: 1, Hard: 2, Good: 3, Easy: 4 } as const

function readReviewLog(): ReviewLine[] {
  if (!fs.existsSync(answers)) return []
  return fs
    .readdirSync(answers)
    .filter((name) => name.endsWith('.jsonl'))
    .sort()
    .flatMap((name) => fs.readFileSync(path.join(answers, name), 'utf8').split('\n'))
    .filter(Boolean)
    .map((line) => JSON.parse(line) as ReviewLine)
}

async function waitForReviewLine(count: number): Promise<ReviewLine> {
  await expect.poll(() => readReviewLog().length).toBe(count + 1)
  return readReviewLog().at(-1) as ReviewLine
}

async function openDecks(page: Page): Promise<void> {
  await page.goto('/')
  await page.getByRole('button', { name: new RegExp(path.basename(vault)) }).click()
  await expect(page.getByRole('heading', { name: path.basename(vault) })).toBeVisible()
}

async function startReview(page: Page): Promise<void> {
  await openDecks(page)
  const review = page.getByRole('button', { name: /Review/ })
  await expect(review).toBeEnabled()
  await review.click()
  await expect(page.getByRole('button', { name: /Show the answer/ })).toBeVisible()
}

test('writes a Good answer of the card shown to the review log', async ({ page }) => {
  const count = readReviewLog().length
  await startReview(page)

  await page.getByRole('button', { name: /Show the answer/ }).click()
  await page.getByRole('button', { name: /Good/ }).click()

  const line = await waitForReviewLine(count)
  expect(line).toMatchObject({ v: 1, rating: RATING.Good })
  expect(line.card).toMatch(/^[0-9a-hjkmnp-tv-z]{10}$/)
  expect(line.template).toBeTruthy()
})

function readFrame(body: Buffer): Record<string, unknown> {
  return JSON.parse(body.subarray(5, 5 + body.readUInt32BE(1)).toString('utf8'))
}

// The window's agent is numen's own, over the stand-in chat API the server script starts.
test('the review mounts the conversation of the agent chat plugin in its agent panel', async ({
  page,
}) => {
  const requests: Record<string, unknown>[] = []
  page.on('request', (request) => {
    if (request.url().endsWith('/numen.v1.AgentService/AskAgent')) {
      requests.push(readFrame(request.postDataBuffer() ?? Buffer.alloc(5)))
    }
  })
  await page.goto('/')

  await page.getByRole('button', { name: new RegExp(path.basename(vault)) }).click()
  await page.getByRole('button', { name: /Review/ }).click()
  await page.getByRole('button', { name: /Ask/ }).click()

  const panel = page.getByRole('region', { name: 'Ask about this card' })
  const field = panel.getByRole('textbox', { name: 'Ask about this card' })
  await expect(field).toBeVisible()
  await field.fill('Where does it live?')
  await panel.getByRole('button', { name: 'Send' }).click()

  await expect(panel.getByText('Where does it live?')).toBeVisible()
  await expect(panel).toContainText(STAND_IN_REPLY)
  // The card goes in the context of the question, and the question is only what was written.
  expect(requests).toHaveLength(1)
  expect(requests[0]).toMatchObject({
    prompt: 'Where does it live?',
    focus: expect.stringMatching(/\.md$/),
  })
  expect(requests[0]?.['context']).toEqual({
    'md.numen.views.flashcards.studio/deck': expect.stringMatching(/\.md$/),
    'md.numen.views.flashcards.studio/card': expect.any(String),
    'md.numen.views.flashcards.studio/template': expect.any(String),
  })
})
