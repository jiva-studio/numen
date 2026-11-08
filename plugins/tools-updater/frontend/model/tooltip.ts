/**
 * What the tooltip of the version chip says: the update that waits and the version that runs.
 */
import type { Update } from '../api/updater'
import { WORDS as words } from '../words'

export function getTooltip(version: string, update: Update): string {
  return `${words.available}: v${update.version}. ${words.running} v${version}. ${words.clickToDownload}`
}
