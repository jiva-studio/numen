/** The person at a tab, whatever it holds: closing it, finding the one in front, its position, what it shows. */
import { expect, type Locator } from 'playwright/test'
import { readActivePath } from '../../../e2e/desktop/tests/support/test.js'
import { getNoteTab, getTabName, revealTreeRow } from '../../../e2e/desktop/tests/support/vault.js'
import {
  ANY_TAB,
  TAB_KINDS,
  assertSingleTab,
  getAddressTitle,
  getEscaped,
  getStem,
  getTitlePattern,
  parseReference,
  pickKindTab,
  type TabKindChoice,
} from './rules.js'
import type { Scenario } from './scenario.js'
import { expectTabHighlight } from './tab-highlight.js'
import { readOpenTabs, type OpenTab } from '../../../e2e/desktop/tests/support/layout.js'
import { expectTabAt, openTabAt } from './tab-position.js'

/** The tab of the file tree: it shows each file and folder as a row named by its path. */
const FILES_TITLE = 'Files'

export class TabDriver {
  constructor(private readonly scenario: Scenario) {}

  private async getTab(kind: TabKindChoice, reference: string, name: string): Promise<Locator> {
    const { page } = this.scenario
    if (kind !== ANY_TAB) {
      const one = await this.requireKind(kind, reference, name)
      return page.getByRole('tab').and(page.locator(`[data-workspace-tab="${one.id}"]`))
    }
    if (parseReference(reference, name).reference === 'at') {
      if (name.endsWith('.md')) return getNoteTab(page, name)
      return page.getByRole('tab', { name: new RegExp(`^${getEscaped(getStem(name))}`, 'i') })
    }
    const tabs = page.getByRole('tab', { name: getTitlePattern(this.getTitle(reference, name)) })
    await expect(tabs.first()).toBeVisible()
    assertSingleTab(name, await tabs.count())
    return tabs
  }

  private async listCandidates(reference: string, name: string): Promise<OpenTab[]> {
    const { tabs } = await readOpenTabs(this.scenario.page.request)
    if (parseReference(reference, name).reference === 'at') return tabs.filter((one) => one.path === name)
    const title = this.getTitle(reference, name)
    return tabs.filter((one) => one.title === title)
  }

  private async requireKind(kind: TabKindChoice, reference: string, name: string): Promise<OpenTab> {
    if (kind === ANY_TAB) throw new Error('no kind is named')
    let found: OpenTab | undefined
    let problem: unknown
    try {
      await expect
        .poll(async () => {
          try {
            found = pickKindTab(kind, await this.listCandidates(reference, name), reference, name)
            problem = undefined
          } catch (error) {
            found = undefined
            problem = error
          }
          return found
        })
        .toBeDefined()
    } catch {
      throw problem ?? new Error(`the tab ${reference} "${name}" is not open`)
    }
    return found as OpenTab
  }

  private async requireKindIfNamed(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    if (kind !== ANY_TAB) await this.requireKind(kind, reference, name)
  }

  private getTitle(reference: string, name: string): string {
    const kind = parseReference(reference, name).reference
    if (kind === 'titled') return name
    if (kind === 'for the web address') return getAddressTitle(name)
    return name.endsWith('.md') ? getTabName(name) : getStem(name)
  }

  private getPanel(reference: string, name: string): Locator {
    const title = this.getTitle(reference, name)
    // The panel of a file that is not a note is named by the file name, with its extension.
    const isFile = parseReference(reference, name).reference === 'at' && !name.endsWith('.md')
    const pattern = isFile ? new RegExp(`^(Linked tab )?${getEscaped(title)}(\\.\\w+)?( Close |$)`) : getTitlePattern(title)
    return this.scenario.page.getByRole('tabpanel', { name: pattern })
  }

  async close(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    const tab = await this.getTab(kind, reference, name)
    await tab.getByRole('button', { name: /^Close / }).click()
    await expect(tab).toHaveCount(0)
  }

  private isNote(reference: string, name: string): boolean {
    return parseReference(reference, name).reference === 'at' && name.endsWith('.md')
  }

  private async listNotePaths(): Promise<string[]> {
    const { tabs } = await readOpenTabs(this.scenario.page.request)
    return tabs.filter((one) => one.kind === 'note').map((one) => one.path)
  }

  async expectOpen(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    if (kind === ANY_TAB && this.isNote(reference, name)) await expect.poll(() => this.listNotePaths()).toContain(name)
    await expect(await this.getTab(kind, reference, name)).toBeVisible()
  }

  async expectNotOpen(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    if (kind !== ANY_TAB) {
      await expect
        .poll(async () => {
          const tabs = await this.listCandidates(reference, name)
          return tabs.some((one) => one.kind === TAB_KINDS[kind].id)
        })
        .toBe(false)
    } else if (this.isNote(reference, name)) {
      await expect.poll(() => this.listNotePaths()).not.toContain(name)
    } else {
      await expect(this.scenario.page.getByRole('tab', { name: getTitlePattern(this.getTitle(reference, name)) })).toHaveCount(0)
    }
  }

  async activate(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await (await this.getTab(kind, reference, name)).click()
  }

  async expectHasText(kind: TabKindChoice, reference: string, name: string, text: string): Promise<void> {
    await this.requireKindIfNamed(kind, reference, name)
    const panel = this.getPanel(reference, name)
    await expect(panel).toBeVisible()
    if (this.getTitle(reference, name) === FILES_TITLE) {
      await expect(panel.locator(`[data-tree-row="${text}"]`)).toBeVisible()
      return
    }
    await expect(panel).toContainText(text)
  }

  async expectDoesNotHaveText(kind: TabKindChoice, reference: string, name: string, text: string): Promise<void> {
    await this.requireKindIfNamed(kind, reference, name)
    const panel = this.getPanel(reference, name)
    await expect(panel).toBeVisible()
    await expect(panel).not.toContainText(text)
  }

  async openFile(filePath: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await this.scenario.watchVersions(filePath)
    await (await revealTreeRow(this.scenario.page, filePath)).press('Enter')
    await expect.poll(() => readActivePath(this.scenario.page.request)).toBe(filePath)
  }

  async playRecording(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await this.requireKindIfNamed(kind, reference, name)
    await this.getPanel(reference, name).getByRole('button', { name: /^Play/ }).click()
  }

  async expectPlaying(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await this.requireKindIfNamed(kind, reference, name)
    const panel = this.getPanel(reference, name)
    await expect(panel.getByRole('button', { name: /^Pause/ })).toBeVisible()
    await expect(panel).toContainText(/0:0[1-9]/)
  }

  async expectActive(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    if (kind !== ANY_TAB) {
      const one = await this.requireKind(kind, reference, name)
      await expect.poll(async () => (await readOpenTabs(this.scenario.page.request)).activeId).toBe(one.id)
      return
    }
    if (parseReference(reference, name).reference === 'at' && name.endsWith('.md')) {
      // A tab linked to another tab keeps its own id and shows the note it was sent to.
      await expect.poll(() => readActivePath(this.scenario.page.request)).toBe(name)
      return
    }
    await expect(await this.getTab(kind, reference, name)).toHaveAttribute('aria-selected', 'true')
  }

  async openAt(kind: TabKindChoice, reference: string, name: string, place: { position: string; where: string }): Promise<void> {
    await this.scenario.ensureWorkspace()
    await this.requireKindIfNamed(kind, reference, name)
    await openTabAt(this.scenario.page, await this.scenario.getPath(reference, name), place.position, place.where)
  }

  async expectAt(kind: TabKindChoice, reference: string, name: string, place: { position: string; where: string }): Promise<void> {
    await this.requireKindIfNamed(kind, reference, name)
    await expectTabAt(this.scenario.page, await this.scenario.getPath(reference, name), place.position, place.where)
  }

  async expectHighlight(kind: TabKindChoice, reference: string, name: string, text?: string): Promise<void> {
    await this.requireKindIfNamed(kind, reference, name)
    await expectTabHighlight(this.scenario.page, await this.scenario.getPath(reference, name), text)
  }
}
