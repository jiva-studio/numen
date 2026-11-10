/**
 * Loads the commands the plugins declare as the window starts and each time the palette opens.
 * They are read again when a plugin starts or stops, so the shortcuts of its commands follow it.
 */
import { watch, type Ref } from 'vue'
import { formatError } from '@/shared/api/errors'
import type { DeclaredCommands } from './declaredCommands'
import type { MessageWriter } from '@/shared/model/notices/messages'

interface CommandListSources {
  readonly declaredCommands?: DeclaredCommands | undefined
  readonly writeMessage: MessageWriter
  readonly plugins?: Ref<readonly { readonly id: string; readonly isActive: boolean }[]> | undefined
  readonly rest?:
    | {
        readonly read?:
          | { readonly getCommittedSetting?: ((path: readonly string[]) => unknown) | undefined }
          | undefined
      }
    | undefined
}

export function watchCommandLists(sources: CommandListSources, isOpen: Ref<boolean>): void {
  const { declaredCommands, writeMessage, plugins } = sources
  const getCommittedSetting = sources.rest?.read?.getCommittedSetting
  if (declaredCommands) {
    const loadDeclared = () => {
      declaredCommands.load().catch((error: unknown) => writeMessage(formatError(error), 'error'))
    }
    loadDeclared()
    watch(isOpen, (open) => {
      if (open) loadDeclared()
    })
    if (plugins) {
      watch(
        () =>
          plugins.value
            .map((one) => `${one.id}:${one.isActive}:${isSwitchedOn(one.id, getCommittedSetting)}`)
            .join(),
        loadDeclared,
      )
    }
  }
}

const isSwitchedOn = (
  pluginId: string,
  getCommittedSetting: ((path: readonly string[]) => unknown) | undefined,
): boolean => getCommittedSetting?.([pluginId, 'enabled']) !== false
