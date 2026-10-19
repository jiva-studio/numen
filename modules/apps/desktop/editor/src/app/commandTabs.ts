import { WORDS as words } from '@/shared/words'
import type { MessageWriter } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'
import type { VaultCore } from './vault'

/** Opens the tab a command over a file leads to, as the kinds of the window say where that is. */
export async function openCommandTab(
  deps: {
    readonly vaultCore: VaultCore
    readonly held: Pick<ReturnType<typeof useWindowTabs>, 'resolveCommand' | 'openTabOfKind'>
    readonly writeMessage: MessageWriter
  },
  path: string,
  command: string,
): Promise<void> {
  const { vaultCore, held, writeMessage } = deps
  const type = (await vaultCore.fileKinds([path])).get(path)?.type ?? 'note'
  const resolved = await held.resolveCommand(path, command, type)
  if (resolved === null) return writeMessage(words.notOffered, 'error')
  if (!resolved.ok) return writeMessage(resolved.text, resolved.severity)
  await held.openTabOfKind(resolved.kind, resolved.at)
}
