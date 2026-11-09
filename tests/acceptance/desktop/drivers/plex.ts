/** The person at the plex: the notes drawn around the note in focus, and the links made from there. */
import { expect, type Locator } from 'playwright/test'
import { getGraph } from '../../../e2e/desktop/tests/support/workspace.js'
import { getEscaped, getStem, parseReference } from './rules.js'
import type { Scenario } from './scenario.js'

export class PlexDriver {
  constructor(private readonly scenario: Scenario) {}

  private getTitle(reference: string, name: string): string {
    return parseReference(reference, name).reference === 'at' ? getStem(name) : name
  }

  private getNote(title: string, relation: string): Locator {
    const role = relation === 'focus' ? 'img' : 'button'
    return getGraph(this.scenario.page).getByRole(role, { name: `${title}, ${relation}`, exact: true })
  }

  async expectHasNote(reference: string, name: string, relation: string): Promise<void> {
    await expect(this.getNote(this.getTitle(reference, name), relation)).toBeAttached()
  }

  async choose(reference: string, name: string): Promise<void> {
    const title = getEscaped(this.getTitle(reference, name))
    await getGraph(this.scenario.page)
      .getByRole('button', { name: new RegExp(`^${title}, `) })
      .click()
  }

  async link(reference: string, name: string, toReference: string, toName: string): Promise<void> {
    const { page } = this.scenario
    const focus = this.getNote(this.getTitle(reference, name), 'focus')
    await focus.hover()
    const handle = page.locator('.plex__handle').first()
    await handle.hover()
    await handle.click()
    const field = page.getByRole('dialog', { name: 'New thought' }).getByRole('combobox')
    const toTitle = this.getTitle(toReference, toName)
    await field.fill(toTitle)
    await page.getByRole('option', { name: toTitle, exact: true }).first().click()
  }
}
