/** The person at the flashcards window: choosing a vault, reviewing its cards, answering them. */
import fs from 'node:fs'
import path from 'node:path'
import { expect } from 'playwright/test'
import { escapeRegExp, getRatingNumber, parseReviewScope } from './rules.js'
import type { Scenario } from './scenario.js'

const getAnswerFolder = (vault: string): string => path.join(vault, '.numen', 'md.numen.views.flashcards.studio')

const readVaultAnswers = (vault: string): { rating?: number; undo?: string }[] => {
  const folder = getAnswerFolder(vault)
  if (!fs.existsSync(folder)) return []
  return fs
    .readdirSync(folder)
    .filter((name) => name.endsWith('.jsonl'))
    .sort()
    .flatMap((name) => fs.readFileSync(path.join(folder, name), 'utf8').split('\n'))
    .filter(Boolean)
    .map((line) => JSON.parse(line) as { rating?: number; undo?: string })
}

export class ReviewDriver {
  private answersBefore = 0
  private hasAnswered = false

  constructor(private readonly scenario: Scenario) {}

  private readAnswers(): { rating?: number; undo?: string }[] {
    const vaults = this.scenario.hasWorld ? this.scenario.getWorld().getVaults() : []
    const folder = vaults.length > 0 ? this.scenario.getWorld().getFolder(vaults[0]) : (process.env['NUMEN_FLASHCARDS_VAULT'] ?? '')
    return readVaultAnswers(folder)
  }

  async chooseVault(name: string): Promise<void> {
    const { page } = this.scenario
    await page.getByRole('button', { name: new RegExp(escapeRegExp(name)) }).click()
    await expect(page.getByRole('heading', { name })).toBeVisible()
  }

  async startReview(scope: string, name: string): Promise<void> {
    const { page } = this.scenario
    const button =
      parseReviewScope(scope) === 'vault'
        ? page.getByRole('button', { name: /Review/ })
        : page.getByRole('button', { name: new RegExp(escapeRegExp(name)) })
    await expect(button).toBeEnabled()
    await button.click()
    await expect(page.getByRole('button', { name: /Show the answer/ })).toBeVisible()
  }

  async showAnswer(): Promise<void> {
    await this.scenario.page.getByRole('button', { name: /Show the answer/ }).click()
  }

  async answer(answer: string): Promise<void> {
    getRatingNumber(answer)
    this.answersBefore = this.readAnswers().length
    this.hasAnswered = true
    await this.scenario.page.getByRole('button', { name: new RegExp(`\\b${answer}\\b`) }).click()
  }

  async expectAnswer(answer: string): Promise<void> {
    const rating = getRatingNumber(answer)
    await expect.poll(() => this.readAnswers().length).toBe(this.answersBefore + 1)
    expect(this.readAnswers().at(-1)?.rating).toBe(rating)
  }

  /** Takes back the answer of the scenario, so the card is due again for the next one. */
  async takeBackAnswer(): Promise<void> {
    if (!this.hasAnswered) return
    const { page } = this.scenario
    const undo = page.getByRole('button', { name: /Undo/ })
    const finished = page.getByRole('button', { name: /Back to the decks/ })
    await expect(undo.or(finished)).toBeVisible()
    if (await finished.isVisible()) return
    await expect(undo).toBeEnabled()
    const count = this.readAnswers().length
    await undo.click()
    await expect.poll(() => this.readAnswers().length).toBe(count + 1)
  }

  async expectSessionHas(text: string): Promise<void> {
    await expect(this.scenario.page.getByText(text, { exact: true }).first()).toBeVisible()
  }
}
