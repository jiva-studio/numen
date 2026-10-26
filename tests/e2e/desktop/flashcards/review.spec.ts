import fs from 'node:fs'
import path from 'node:path'
import { expect, test } from 'playwright/test'
import { STAND_IN_REPLY } from '../scripts/stand-ins.mjs'

const vault = process.env.NUMEN_FLASHCARDS_VAULT ?? ''
const answers = path.join(vault, '.numen', 'md.numen.views.flashcards.studio')

/** The lines of review answers the plugin wrote into its own folder of the vault. */
function readAnswers(): string[] {
  if (!fs.existsSync(answers)) return []
  return fs
    .readdirSync(answers)
    .filter((name) => name.endsWith('.jsonl'))
    .flatMap((name) => fs.readFileSync(path.join(answers, name), 'utf8').split('\n'))
    .filter(Boolean)
}

test('the window loads the flashcards plugin, shows what is due and writes an answer', async ({
  page,
}) => {
  await page.goto('/')

  const row = page.getByRole('button', { name: new RegExp(path.basename(vault)) })
  await expect(row).toBeVisible()
  await row.click()

  await expect(page.getByRole('heading', { name: path.basename(vault) })).toBeVisible()
  const review = page.getByRole('button', { name: /Review/ })
  await expect(review).toBeEnabled()
  await review.click()

  await page.getByRole('button', { name: /Show the answer/ }).click()
  await page.getByRole('button', { name: /Good/ }).click()

  await expect.poll(readAnswers).toHaveLength(1)
  expect(JSON.parse(readAnswers()[0] ?? '{}')).toMatchObject({ rating: expect.anything() })
})

/** The JSON message of the one frame a Connect streaming request carries. */
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
  // The card goes beside the question, and the question is only what was written.
  expect(requests).toHaveLength(1)
  expect(requests[0]).toMatchObject({
    prompt: 'Where does it live?',
    focus: expect.stringMatching(/\.md$/),
  })
  expect(requests[0]?.['mark']).toEqual(expect.any(String))
  expect(requests[0]?.['face']).toEqual(expect.any(String))
})
