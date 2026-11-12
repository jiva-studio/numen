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

  async expectHasNote(title: string, relation: string, has: boolean): Promise<void> {
    await expect(this.getNote(title, relation)).toHaveCount(has ? 1 : 0)
  }

  /** The node of a note, whatever its relation to the note in focus. */
  private getNode(title: string): Locator {
    return getGraph(this.scenario.page).locator(`[aria-label^="${title}, "]`)
  }

  async hoverOver(title: string): Promise<void> {
    const node = this.getNode(title)
    await expect(node, `the plex draws no note titled "${title}"`).toHaveCount(1)
    await node.hover()
  }

  async expectHeading(heading: string, title: string, has: boolean): Promise<void> {
    const headings = this.getNode(title).locator('.plex__heading-text').filter({ hasText: heading })
    await expect(headings).toHaveCount(has ? 1 : 0)
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
