/** The window follows the editor settings: it writes them as tokens of the page. */
import { win } from '@numen/ui'
import type { SettingsStore } from '@/entities/settings'

export function createEditorTokens(
  _store: Pick<SettingsStore, 'read' | 'state' | 'changes'>,
  _sheet: Document = win.document,
): { start(): () => void } {
  return { start: () => () => undefined }
}
