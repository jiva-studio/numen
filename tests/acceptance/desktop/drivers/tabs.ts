/** The person at a tab, whatever it holds: closing it, finding the one in front, its position, what it shows. */
import { expect, type Locator } from 'playwright/test'
import { readActivePath } from '../../../e2e/desktop/tests/support/test.js'
import { getTabName, revealTreeRow } from '../../../e2e/desktop/tests/support/vault.js'
import {
  ANY_TAB,
  TAB_KINDS,
  assertSingleTab,
  getAddressTitle,
  getStem,
  parseTabRows,
  parseReference,
  pickKindTab,
  type Place,
  type TabKindChoice,
} from './rules.js'
import { TabPanel, TabPanels } from './components/TabPanel.js'
import { TabStrip } from './components/TabStrip.js'
import type { Scenario } from './scenario.js'
import { expectTabHighlight } from './tab-highlight.js'
import { readOpenTabs, type OpenTab } from '../../../e2e/desktop/tests/support/layout.js'
import { expectTabAt, openTabAt } from './tab-position.js'

/** The tab of the file tree: it shows each file and folder as a row named by its path. */
const FILES_TITLE = 'Files'
/** The id the window gives its one Settings tab: the kind and the identity of the kind. */
const SETTINGS_TAB_ID = 'settings:settings'
const isTitledByContent = (kind: TabKindChoice): boolean => kind === 'Agent'

/** The files that have a tab type: a file of any other type is not opened, so nothing shows that the open was handled. */
const TAB_TYPE_FILES = /\.(md|pdf|epub|mp3|m4a|wav|ogg|flac|mp4|webm)$/i
const SETTLE_MS = 1_500

export class TabDriver {
  private readonly strip: TabStrip
  private readonly panels: TabPanels

  constructor(private readonly scenario: Scenario) {
    this.strip = new TabStrip(scenario)
    this.panels = new TabPanels(scenario)
  }

  async findTab(kind: TabKindChoice, reference: string, name: string): Promise<Locator> {
    return this.getTab(kind, reference, name)
  }

  private async getTab(kind: TabKindChoice, reference: string, name: string, preferNote = false): Promise<Locator> {
    if (kind !== ANY_TAB) {
      const one = await this.requireKind(kind, reference, name)
      return this.strip.tabById(one.id)
    }
    if (parseReference(reference, name).reference === 'at') {
      if (name.endsWith('.md')) return this.strip.noteTab(getTabName(name))
      return this.strip.tabOfFile(name)
    }
    const tabs = this.strip.tabTitled(this.getTitle(reference, name))
    await expect(tabs.first()).toBeVisible()
    const count = await tabs.count()
    // A note and its plex carry one title: a command run on the tab of that title is meant for the note.
    if (preferNote && count > 1) {
      const notes = tabs.and(this.strip.noteTabs())
      if ((await notes.count()) === 1) return notes
    }
    assertSingleTab(name, count)
    return tabs
  }

  private async listCandidates(kind: TabKindChoice, reference: string, name: string): Promise<OpenTab[]> {
    const { tabs } = await readOpenTabs(this.scenario.page.request)
    if (isTitledByContent(kind) && parseReference(reference, name).reference === 'titled') {
      return tabs.filter((one) => one.kind === TAB_KINDS.Agent.id)
    }
    if (parseReference(reference, name).reference === 'at') return tabs.filter((one) => one.path === name)
    const title = this.getTitle(reference, name)
    return tabs.filter((one) => one.title === title)
  }

  private async requireKind(kind: TabKindChoice, reference: string, name: string): Promise<OpenTab> {
    if (kind === ANY_TAB) throw new Error('no kind is named')
    // The core keeps the tabs of the vault; the Settings tab belongs to the window alone.
    if (kind === 'Settings') {
      await expect(this.strip.tabById(SETTINGS_TAB_ID)).toBeVisible()
      return { id: SETTINGS_TAB_ID, kind: TAB_KINDS.Settings.id, path: '', title: TAB_KINDS.Settings.title }
    }
    let found: OpenTab | undefined
    let problem: Error | undefined
    try {
      await expect
        .poll(async () => {
          try {
            found = pickKindTab(kind, await this.listCandidates(kind, reference, name), reference, name)
            problem = undefined
          } catch (error) {
            found = undefined
            problem = error instanceof Error ? error : new Error(String(error))
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

  private async getShownPanel(kind: TabKindChoice, reference: string, name: string): Promise<Locator> {
    await this.requireKindIfNamed(kind, reference, name)
    await this.scenario.closePalette()
    return this.panels.shownFor(await this.getTab(kind, reference, name))
  }

  /** The panel of the tab, once the tab is open as the kind the step names. */
  async getTabPanel(kind: TabKindChoice, reference: string, name: string): Promise<Locator> {
    return this.getShownPanel(kind, reference, name)
  }

  async expectTabs(rows: readonly (readonly string[])[]): Promise<void> {
    const wanted = parseTabRows(rows)
    await expect
      .poll(async () => {
        const { tabs } = await readOpenTabs(this.scenario.page.request)
        return wanted.map((one) => tabs.filter((tab) => tab.kind === TAB_KINDS[one.kind].id && tab.path === one.path).length)
      })
      .toStrictEqual(wanted.map(() => 1))
  }

  async clickLink(kind: TabKindChoice, reference: string, name: string, link: string): Promise<void> {
    const panel = await this.getTabPanel(kind, reference, name)
    await new TabPanel(panel).link(link).click()
  }

  async expectHasRecording(kind: TabKindChoice, reference: string, name: string, has: boolean): Promise<void> {
    const panel = await this.getTabPanel(kind, reference, name)
    await expect(new TabPanel(panel).playButton()).toHaveCount(has ? 1 : 0)
  }

  async close(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    const tab = await this.getTab(kind, reference, name)
    await this.strip.closeButton(tab).click()
    await expect(tab).toHaveCount(0)
  }

  private isNote(reference: string, name: string): boolean {
    return parseReference(reference, name).reference === 'at' && name.endsWith('.md')
  }

  private async isFileOpened(filePath: string): Promise<boolean> {
    const { tabs, activeId } = await readOpenTabs(this.scenario.page.request)
    const active = tabs.find((one) => one.id === activeId)
    if (active?.path === filePath) return true
    // A file opened from a file tree that is linked to another tab shows in that tab, and the tree stays in front.
    return active?.kind === TAB_KINDS.Files.id && tabs.some((one) => one.path === filePath)
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
          const tabs = await this.listCandidates(kind, reference, name)
          return tabs.some((one) => one.kind === TAB_KINDS[kind].id)
        })
        .toBe(false)
    } else if (this.isNote(reference, name)) {
      await expect.poll(() => this.listNotePaths()).not.toContain(name)
    } else {
      await expect(this.strip.tabTitled(this.getTitle(reference, name))).toHaveCount(0)
    }
  }

  async activate(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await this.scenario.closePalette()
    await (await this.getTab(kind, reference, name, true)).click()
  }

  async expectHasText(kind: TabKindChoice, reference: string, name: string, text: string): Promise<void> {
    const panel = await this.getShownPanel(kind, reference, name)
    if (this.getTitle(reference, name) === FILES_TITLE) {
      await expect(new TabPanel(panel).treeRow(text)).toBeVisible()
      return
    }
    await expect(panel).toContainText(text)
  }

  async expectDoesNotHaveText(kind: TabKindChoice, reference: string, name: string, text: string): Promise<void> {
    const panel = await this.getShownPanel(kind, reference, name)
    await expect(panel).not.toContainText(text)
  }

  async openFile(filePath: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await this.scenario.closePalette()
    await this.scenario.watchVersions(filePath)
    await (await revealTreeRow(this.scenario.page, filePath)).press('Enter')
    if (!TAB_TYPE_FILES.test(filePath)) {
      await this.scenario.page.waitForTimeout(SETTLE_MS)
      return
    }
    await expect.poll(() => this.isFileOpened(filePath)).toBe(true)
  }

  async showFile(filePath: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    if (!(await this.isFileOpened(filePath))) await this.openFile(filePath)
  }

  async playRecording(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    const panel = await this.getShownPanel(kind, reference, name)
    await new TabPanel(panel).playButton().click()
  }

  async expectPlaying(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    const panel = await this.getShownPanel(kind, reference, name)
    await expect(new TabPanel(panel).pauseButton()).toBeVisible()
    await expect(panel).toContainText(/0:0[1-9]/)
  }

  private async getShownNotePath(notePath: string): Promise<string> {
    const active = await readActivePath(this.scenario.page.request)
    if (active === notePath) return active
    const { tabs, activeId } = await readOpenTabs(this.scenario.page.request)
    if (tabs.find((one) => one.id === activeId)?.kind !== TAB_KINDS.Files.id) return active
    const tab = await this.getTab(ANY_TAB, 'at', notePath)
    if ((await tab.count()) === 0) return active
    return (await this.strip.isSelected(tab.first())) ? notePath : active
  }

  async expectActive(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    if (kind === 'Agent' || kind === 'Settings') {
      // The core leaves a conversation and the Settings tab out of the active tab, so the tab in front of its pane is meant.
      await expect(await this.getTab(kind, reference, name)).toHaveAttribute('aria-selected', 'true')
      return
    }
    if (kind !== ANY_TAB) {
      const one = await this.requireKind(kind, reference, name)
      await expect.poll(async () => (await readOpenTabs(this.scenario.page.request)).activeId).toBe(one.id)
      return
    }
    if (parseReference(reference, name).reference === 'at' && name.endsWith('.md')) {
      // A tab linked to another tab keeps its own id and shows the note it was sent to.
      await expect.poll(() => this.getShownNotePath(name)).toBe(name)
      return
    }
    await expect(await this.getTab(kind, reference, name)).toHaveAttribute('aria-selected', 'true')
  }

  async openAt(kind: TabKindChoice, reference: string, name: string, place: Place): Promise<void> {
    await this.scenario.ensureWorkspace()
    await openTabAt(this.scenario, await this.scenario.getPath(reference, name), place.position, place.where)
    await this.requireKindIfNamed(kind, reference, name)
  }

  async expectAt(kind: TabKindChoice, reference: string, name: string, place: Place): Promise<void> {
    await this.requireKindIfNamed(kind, reference, name)
    await expectTabAt(this.scenario, await this.scenario.getPath(reference, name), place.position, place.where)
  }

  async expectHighlight(kind: TabKindChoice, reference: string, name: string, text?: string): Promise<void> {
    await this.requireKindIfNamed(kind, reference, name)
    await expectTabHighlight(this.scenario, await this.scenario.getPath(reference, name), text)
  }
}
