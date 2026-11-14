/** The person at the text of a file or a link: making it, proofreading it, downloading it, deleting it. */
import { expect } from 'playwright/test'
import { getTab } from '../../../e2e/desktop/tests/support/layout.js'
import type { ApplicationDriver } from './application.js'
import type { WindowDriver } from './window.js'
import { getAddressTitle, getLinkNotePath, getCurrentVersionLabel, getVersionTitle, type TabKindChoice, type TextKind } from './rules.js'
import { TabPanel } from './components/TabPanel.js'
import type { Scenario } from './scenario.js'
import type { TabDriver } from './tabs.js'
import { hasMadeText, hasVersions, listVersions } from './text-versions.js'

const CREATE_TEXT = 'Create text'
const RECOGNIZE_TEXT = 'Recognize text'
const PROOFREAD_TEXT = 'Proofread text'
const TRANSCRIBE = 'Transcribe'
const IMPORT_LINK = 'Import link'
const DOWNLOAD_CAPTIONS = 'Download captions'
const DOWNLOAD_MEDIA = 'Download media'
const VIDEO = 'Video'
const DELETE_TEXT = 'Delete text'
const DELETE_MEDIA = 'Delete media'
/** The call of the host that stores an edit of a text. */
const CREATE_TEXT_EDIT = '/numen.v1.TextService/CreateTextEdit'

export class TextDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly application: ApplicationDriver,
    private readonly tab: TabDriver,
    private readonly window: WindowDriver,
  ) {}

  async recognize(filePath: string): Promise<void> {
    await this.tab.showFile(filePath)
    await this.application.runCommand(CREATE_TEXT)
    await this.application.pickItem(RECOGNIZE_TEXT)
    await this.expectMadeText(filePath)
  }

  async transcribe(filePath: string): Promise<void> {
    await this.tab.showFile(filePath)
    await this.application.runCommand(TRANSCRIBE)
    await this.expectMadeText(filePath)
  }

  async proofread(filePath: string, kind: TextKind): Promise<void> {
    await this.tab.showFile(filePath)
    await this.application.runCommand(PROOFREAD_TEXT)
    await this.application.pickItem(getCurrentVersionLabel(kind))
  }

  /** The file has a version of the kind of text, which the product lists with the title of the kind. */
  async expectVersion(filePath: string, kind: TextKind): Promise<void> {
    const { request } = this.scenario.page
    await expect
      .poll(async () => (await listVersions(request, filePath)).some((one) => one.title.startsWith(getVersionTitle(kind))))
      .toBe(true)
  }

  async expectNoVersion(filePath: string, kind: TextKind): Promise<void> {
    const { request } = this.scenario.page
    await expect
      .poll(async () => (await listVersions(request, filePath)).some((one) => one.title.startsWith(getVersionTitle(kind))))
      .toBe(false)
  }

  async deleteVersion(filePath: string, kind: TextKind): Promise<void> {
    await this.tab.showFile(filePath)
    await this.application.runCommand(DELETE_TEXT)
    await this.application.pickItem(getCurrentVersionLabel(kind))
  }

  /** The person clicks the transcript, goes to its end, types and saves with the key; the save answers once the edit is written. */
  async writeTranscript(words: string, kind: TabKindChoice, reference: string, name: string): Promise<void> {
    const { page } = this.scenario
    const panel = await this.tab.getTabPanel(kind, reference, name)
    await new TabPanel(panel).transcript().click()
    await page.keyboard.press('ControlOrMeta+End')
    await page.keyboard.type(` ${words}`)
    const saved = page.waitForResponse((answer) => answer.url().includes(CREATE_TEXT_EDIT))
    await page.keyboard.press('ControlOrMeta+s')
    expect((await saved).ok()).toBe(true)
  }

  async importLink(address: string): Promise<void> {
    await this.application.runCommand(IMPORT_LINK)
    await this.application.enterText(address)
    await expect(getTab(this.scenario.page, getAddressTitle(address)).first()).toBeVisible()
  }

  async downloadText(address: string): Promise<void> {
    const notePath = await this.showLink(address)
    await this.application.runCommand(DOWNLOAD_CAPTIONS)
    await this.expectEnded(notePath)
  }

  async downloadVideo(address: string): Promise<void> {
    await this.showLink(address)
    await this.application.runCommand(DOWNLOAD_MEDIA)
    await this.application.pickItem(VIDEO)
    // The copy runs in the background; a later delete or download of the same link waits for its end.
    await this.tab.expectHasRecording('Media', 'for the web address', address, true)
  }

  async deleteVideo(address: string): Promise<void> {
    await this.showLink(address)
    await this.application.runCommand(DELETE_MEDIA)
    await this.tab.expectHasRecording('Media', 'for the web address', address, false)
  }

  async deleteText(address: string): Promise<void> {
    await this.showLink(address)
    await this.application.runCommand(DELETE_TEXT)
    await this.application.pickItem(getCurrentVersionLabel('article'))
  }

  async openLink(address: string): Promise<void> {
    await this.tab.openFile(getLinkNotePath(address))
  }

  private async showLink(address: string): Promise<string> {
    const notePath = getLinkNotePath(address)
    await this.tab.showFile(notePath)
    return notePath
  }

  private async expectMadeText(filePath: string): Promise<void> {
    const { request } = this.scenario.page
    await expect.poll(() => hasMadeText(request, filePath)).toBe(true)
  }

  private async expectEnded(notePath: string): Promise<void> {
    const { request } = this.scenario.page
    await expect.poll(async () => (await hasVersions(request, notePath)) || (await this.window.hasFailedTask())).toBe(true)
  }
}
