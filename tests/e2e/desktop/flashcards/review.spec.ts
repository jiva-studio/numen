import fs from 'node:fs'
import path from 'node:path'
import { expect, test, type Page } from '../tests/support/certificate.js'
import { STAND_IN_REPLY } from '../scripts/stand-ins.mjs'
import type { JsonObject } from '../tests/support/json.js'

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

test('a deck started on its own reviews only its cards', async ({ page }) => {
  await openDecks(page)
  const deck = page.getByRole('button', { name: /crlf-deck/ })
  await expect(deck).toBeEnabled()
  const figure = deck.locator('.due-count')
  await expect(figure).toHaveText(/^\d+ to review$/)
  const due = (await figure.innerText()).split(' ')[0]

  await deck.click()

  await expect(page.locator('.session__deck')).toHaveText('crlf-deck')
  await expect(page.locator('.session__left')).toHaveText(`${due} left`)
  const count = readReviewLog().length
  await page.getByRole('button', { name: /Show the answer/ }).click()
  await page.getByRole('button', { name: /Easy/ }).click()
  // The one card of cards/crlf-deck.md, by the mark its heading carries.
  expect(await waitForReviewLine(count)).toMatchObject({ card: '9r4tn6wqhd', rating: RATING.Easy })
})

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

test('writes Again, Hard and Easy from their buttons to the review log', async ({ page }) => {
  await startReview(page)

  for (const answer of ['Again', 'Hard', 'Easy'] as const) {
    const count = readReviewLog().length
    await page.getByRole('button', { name: /Show the answer/ }).click()
    await page.getByRole('button', { name: new RegExp(answer) }).click()

    expect(await waitForReviewLine(count)).toMatchObject({ rating: RATING[answer] })
  }
})

test('shows the answer on Space and writes the answer of each number key', async ({ page }) => {
  await startReview(page)

  for (const [key, rating] of [
    ['1', RATING.Again],
    ['2', RATING.Hard],
    ['3', RATING.Good],
    ['4', RATING.Easy],
  ] as const) {
    const count = readReviewLog().length
    await expect(page.getByRole('button', { name: /Show the answer/ })).toBeVisible()
    await page.keyboard.press('Space')
    await expect(page.getByRole('button', { name: /Good/ })).toBeVisible()
    await page.keyboard.press(key)

    expect(await waitForReviewLine(count)).toMatchObject({ rating })
  }
})

test('Undo writes a line taking the last answer back, and shows its card again', async ({
  page,
}) => {
  await startReview(page)
  const where = await page.locator('.session__where').textContent()
  const remaining = await page.locator('.session__left').innerText()
  const count = readReviewLog().length
  await page.getByRole('button', { name: /Show the answer/ }).click()
  await page.getByRole('button', { name: /Easy/ }).click()
  const answer = await waitForReviewLine(count)
  await expect(page.locator('.session__left')).not.toHaveText(remaining)

  await page.getByRole('button', { name: /Undo/ }).click()

  const undone = await waitForReviewLine(count + 1)
  expect(undone).toMatchObject({ v: 1, undo: answer.id })
  expect(undone.rating).toBeUndefined()
  await expect(page.locator('.session__left')).toHaveText(remaining)
  await expect(page.locator('.session__where')).toHaveText(where ?? '')
  await expect(page.getByRole('button', { name: /Again/ })).toBeVisible()
})

function readFrame(body: Buffer): JsonObject {
  return JSON.parse(body.subarray(5, 5 + body.readUInt32BE(1)).toString('utf8'))
}

// The window's agent is numen's own, over the stand-in chat API the server script starts.
test('the review mounts the conversation of the agent chat plugin in its agent panel', async ({
  page,
}) => {
  const requests: JsonObject[] = []
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
