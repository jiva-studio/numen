/** The panel of a tab, and what a reader or a player puts into it. */
import { expect, type Locator } from 'playwright/test'
import { escapeRegExp, getStem } from '../rules.js'
import type { Host } from './host.js'
import { TabStrip } from './TabStrip.js'

/** The content of one panel. */
export class TabPanel {
  constructor(readonly panel: Locator) {}

  link(name: string): Locator {
    return this.panel.getByRole('link', { name, exact: true })
  }

  playButton(): Locator {
    return this.panel.getByRole('button', { name: /^Play/ })
  }

  pauseButton(): Locator {
    return this.panel.getByRole('button', { name: /^Pause/ })
  }

  /** The text of a recording, which a person edits. */
  transcript(): Locator {
    return this.panel.getByRole('textbox', { name: 'Transcript' })
  }

  /** The row of the file tree named by its path. */
  treeRow(path: string): Locator {
    return this.panel.locator(`[data-tree-row="${path}"]`)
  }

  pageField(): Locator {
    return this.panel.getByRole('spinbutton', { name: 'Page' })
  }

  nextPageButton(): Locator {
    return this.panel.getByRole('button', { name: 'Next page' })
  }

  bookCount(): Locator {
    return this.panel.locator('.book__count')
  }

  book(): Locator {
    return this.panel.locator('.book-tab')
  }

  player(): Locator {
    return this.panel.getByRole('group', { name: 'Recording' })
  }

  /** The first highlight on the pages of a PDF. */
  pageHighlight(): Locator {
    return this.panel.locator('.reader__page .reader__highlight').first()
  }
}

/** Finding the panel of a tab. */
export class TabPanels {
  private readonly tabs: TabStrip

  constructor(private readonly host: Host) {
    this.tabs = new TabStrip(host)
  }

  /** The panel of the tab with the id: panels are found through their tab, as a note and its plex carry the same title. */
  byId(id: string | null): Locator {
    return this.host.page.locator(`[role="tabpanel"][aria-labelledby="${id}"]`)
  }

  /** The panel of the tab, once the tab is in front: only the panel of the tab in front is shown. */
  async shownFor(tab: Locator): Promise<Locator> {
    await this.tabs.show(tab)
    const panel = this.byId(await tab.getAttribute('id'))
    await expect(panel).toBeVisible()
    return panel
  }

  /** The panel of the file, named by the beginning of its file name. */
  ofFile(file: string): Locator {
    return this.host.page.getByRole('tabpanel', { name: new RegExp(escapeRegExp(getStem(file)), 'i') })
  }

  /** The panel of the note: a tab linked to the file tree is named with the words "Linked tab" before its title. */
  ofNote(title: string): Locator {
    const escaped = escapeRegExp(title)
    return this.host.page
      .getByRole('tabpanel', { name: new RegExp(`^(Linked tab )?${escaped} Close ${escaped}$`) })
      .filter({ has: this.host.page.locator('.note__text') })
  }
}
