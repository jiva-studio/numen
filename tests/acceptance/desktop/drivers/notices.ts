/** What a computer or a reader tells the person: the warnings of Numen Sync and the words the window shows. */
import { expect } from 'playwright/test'
import type { ComputerDriver } from './computer.js'
import type { SharedReaderDriver } from './shared-reader.js'
import type { SharingDriver } from './sharing.js'
import { readSyncWarnings } from './sync.js'

/** A notice shows within a few passes of the plugin on both computers. */
const TOLD_LIMIT_MS = 20_000

const escapePattern = (text: string): string => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

export class NoticeDriver {
  constructor(
    private readonly computers: ComputerDriver,
    private readonly sharing: SharingDriver,
    private readonly readers: SharedReaderDriver,
  ) {}

  /** Expects the teller to be told the words: by a warning of Numen Sync, or by any text the window shows, whatever its case. */
  async expectTold(teller: string, text: string): Promise<void> {
    if (teller === 'the reader') return this.readers.expectTold('reader', text)
    if (teller === 'the second reader') return this.readers.expectTold('second reader', text)
    if (teller === 'the person' && this.sharing.isRefusedWith(text)) return
    const scenario = this.computers.getScenario(teller === 'the person' ? 'first' : 'second')
    const words = new RegExp(escapePattern(text), 'i')
    await expect
      .poll(async () => (await readSyncWarnings(scenario)).includes(text) || scenario.page.getByText(words).first().isVisible(), {
        message: `${teller} is told "${text}"`,
        timeout: TOLD_LIMIT_MS,
      })
      .toBe(true)
  }
}
