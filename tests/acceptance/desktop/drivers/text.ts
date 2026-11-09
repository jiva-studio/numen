/** The person at the text of a file or a link: making it, proofreading it, downloading it, deleting it. */
import { expect } from 'playwright/test'
import type { ApplicationDriver } from './application.js'
import { getAddressTitle, getCurrentVersionLabel, type TextKind } from './rules.js'
import type { Scenario } from './scenario.js'
import type { TabDriver } from './tabs.js'
import { hasMadeText, hasVersions } from './text-versions.js'

const CREATE_TEXT = 'Create text'
const RECOGNIZE_TEXT = 'Recognize text'
const PROOFREAD_TEXT = 'Proofread text'
const TRANSCRIBE = 'Transcribe'
const IMPORT_LINK = 'Import link'
const DOWNLOAD_CAPTIONS = 'Download captions'
const DOWNLOAD_MEDIA = 'Download media'
const VIDEO = 'Video'
const DELETE_TEXT = 'Delete text'

export class TextDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly application: ApplicationDriver,
    private readonly tab: TabDriver,
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

  async importLink(address: string): Promise<void> {
    await this.application.runCommand(IMPORT_LINK)
    await this.application.enterText(address)
  }

  async downloadText(address: string): Promise<void> {
    const notePath = await this.showLink(address)
    await this.application.runCommand(DOWNLOAD_CAPTIONS)
    await this.expectVersions(notePath)
  }

  async downloadVideo(address: string): Promise<void> {
    await this.showLink(address)
    await this.application.runCommand(DOWNLOAD_MEDIA)
    await this.application.pickItem(VIDEO)
  }

  async deleteText(address: string): Promise<void> {
    await this.showLink(address)
    await this.application.runCommand(DELETE_TEXT)
    await this.application.pickItem(getCurrentVersionLabel('article'))
  }

  private async showLink(address: string): Promise<string> {
    const notePath = `${getAddressTitle(address)}.md`
    await this.tab.showFile(notePath)
    return notePath
  }

  private async expectMadeText(filePath: string): Promise<void> {
    const { request } = this.scenario.page
    await expect.poll(() => hasMadeText(request, filePath)).toBe(true)
  }

  private async expectVersions(notePath: string): Promise<void> {
    const { request } = this.scenario.page
    await expect.poll(() => hasVersions(request, notePath)).toBe(true)
  }
}
