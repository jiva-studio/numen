/**
 * What the tooltip of the version chip says: the version that runs, and the update when one waits.
 */
import type { State, Update } from '../api/updater'
import { WORDS as words } from '../words'

export function getTooltip(version: string, update: Update | null, state: State): string {
  if (update) {
    return `${words.available}: v${update.version}. ${words.running} v${version}. ${words.clickToDownload}`
  }
  const running = `${words.product} v${version}`
  return state === 'current' ? `${running}. ${words.latest}` : `${running}, ${words.versionYouRun}`
}
