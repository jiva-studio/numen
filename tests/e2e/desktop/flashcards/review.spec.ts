import fs from 'node:fs'
import path from 'node:path'
import { expect, test } from 'playwright/test'

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
