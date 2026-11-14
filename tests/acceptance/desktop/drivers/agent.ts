/** The person at the composer of the agent: the message and the Send and Stop buttons. */
import { expect, type Locator } from 'playwright/test'
import type { Scenario } from './scenario.js'

const COMPOSER = 'Ask about the vault'
/** The stand-in model answers within this time once it is not held. */
const ANSWER_LIMIT_MS = 20_000

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
}
