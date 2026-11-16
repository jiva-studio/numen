/** The person at the composer of the agent: the message and the Send and Stop buttons. */
import { expect, type Locator } from 'playwright/test'
import type { Scenario } from './scenario.js'

const COMPOSER = 'Ask about the vault'
/** The stand-in model answers within this time once it is not held. */
const ANSWER_LIMIT_MS = 20_000
/** A real model answers within this time. */
const LIVE_ANSWER_LIMIT_MS = 120_000
/** What the agent says in place of an answer. */
const NON_ANSWERS = ['The agent returned no answer.', 'Choose a model.']

export class AgentDriver {
  constructor(private readonly scenario: Scenario) {}

  private getField(): Locator {
    return this.scenario.page.getByRole('textbox', { name: COMPOSER }).first()
  }

  private getSend(): Locator {
    return this.scenario.page.getByRole('button', { name: 'Send' }).first()
  }

  private getStop(): Locator {
    return this.scenario.page.getByRole('button', { name: 'Stop the agent' }).first()
  }

  async send(message: string): Promise<void> {
    const field = this.getField()
    await expect(field, 'the composer of the agent is not shown: open the Agent tab first').toBeVisible()
    await field.fill(message)
    await expect(this.getSend()).toBeEnabled()
    await this.getSend().click()
  }

  async stop(): Promise<void> {
    await expect(this.getStop(), 'the agent is not working, so there is nothing to stop').toBeEnabled({ timeout: ANSWER_LIMIT_MS })
    await this.getStop().click()
    await expect(this.scenario.page.getByRole('button', { name: 'Stop the agent' })).toHaveCount(0)
  }

  async expectEmpty(): Promise<void> {
    await expect(this.getField()).toHaveValue('')
  }

  async expectCannotSend(): Promise<void> {
    await expect(this.getSend()).toBeDisabled()
  }

  async expectText(text: string): Promise<void> {
    await expect(this.getField()).toHaveValue(text)
  }

  /** The thread has a settled answer with words; a turn that failed or says the agent has nothing to answer with ends the wait at once. */
  async expectAnswered(): Promise<void> {
    const turns = this.scenario.page.locator('[data-voice="response"]')
    const failed = turns.and(this.scenario.page.locator('[data-state="failed"]'))
    const going = turns.and(this.scenario.page.locator(':not([data-state="settled"]):not([data-state="failed"])'))
    const readOutcome = async (): Promise<{ isDone: boolean; text: string }> => {
      if ((await failed.count()) > 0) return { isDone: true, text: `the agent failed: ${(await failed.first().innerText()).trim()}` }
      if ((await going.count()) > 0) return { isDone: false, text: 'the agent is still answering' }
      const texts = (await turns.allInnerTexts()).map((one) => one.trim())
      const nothing = texts.find((one) => NON_ANSWERS.includes(one))
      if (nothing !== undefined) return { isDone: true, text: `the agent replied "${nothing}" in place of an answer` }
      const hasAnswer = texts.some((one) => one !== '')
      return { isDone: hasAnswer, text: hasAnswer ? '' : 'the agent has not answered yet' }
    }
    let outcome = { isDone: false, text: '' }
    await expect
      .poll(
        async () => {
          outcome = await readOutcome()
          return outcome.isDone
        },
        { timeout: LIVE_ANSWER_LIMIT_MS, message: 'the agent has not answered' },
      )
      .toBe(true)
    expect(outcome.text, 'the agent has not answered').toBe('')
  }
}
