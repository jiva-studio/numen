import { presets } from '@/entities/deck'
import { WORDS as words } from '@/shared/words'
import type { MessageWriter } from '@/shared/notices/messages'
import type { VaultCore } from './vault'

export interface DeckPresetHandlers {
  openPreset: (path: string, title?: string) => Promise<void> | void
  writeMessage: MessageWriter
}

export async function resolveDeckPreset(
  vaultCore: VaultCore,
  path: string,
  handlers: DeckPresetHandlers,
): Promise<void> {
  const kind = (await vaultCore.fileKinds([path])).get(path)
  if (kind?.type !== 'deck') {
    await handlers.openPreset(path)
    return
  }
  const answer = await presets.getDeckPreset(path)
  if (!answer.ok) return handlers.writeMessage(words.errors[answer.error], 'error')
  const read = answer.value.preset
  if (!read?.path) return handlers.writeMessage(words.noPreset, 'caution')
  await handlers.openPreset(read.path, read.title)
}
