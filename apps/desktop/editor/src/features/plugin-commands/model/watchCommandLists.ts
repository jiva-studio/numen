/**
 * Loads the commands the plugins declare as the window starts and each time the palette opens.
 * They are read again when a plugin starts or stops, so the shortcuts of its commands follow it.
 */
import { watch, type Ref } from 'vue'
import { formatError, type Caught } from '@/shared/api/errors'
import type { DeclaredCommands } from './declaredCommands'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { JsonValue } from '@/shared/api/json'

interface CommandListSources {
  readonly declaredCommands: DeclaredCommands
  readonly writeMessage: MessageWriter
  readonly plugins: Ref<readonly { readonly id: string; readonly isActive: boolean }[]>
  readonly getCommittedSetting: (path: readonly string[]) => JsonValue | undefined
}

export function watchCommandLists(sources: CommandListSources, isOpen: Ref<boolean>): void {
  const { declaredCommands, writeMessage, plugins, getCommittedSetting } = sources
  const loadDeclaredCommands = () => {
    declaredCommands.load().catch((error: Caught) => writeMessage(formatError(error), 'error'))
  }
  loadDeclaredCommands()
  watch(isOpen, (open) => {
    if (open) loadDeclaredCommands()
  })
  watch(
    () =>
      plugins.value
        .map(
          (one) =>
            `${one.id}:${one.isActive}:${getCommittedSetting([one.id, 'enabled']) !== false}`,
        )
        .join(),
    loadDeclaredCommands,
  )
}
