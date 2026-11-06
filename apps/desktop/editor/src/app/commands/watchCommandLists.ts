/**
 * Loads the commands the plugins declare as the window starts and each time the palette opens.
 * They are read again when a plugin starts or stops, so the shortcuts of its commands follow it.
 */
import { watch, type Ref } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import type { DeclaredCommands } from './declaredCommands'
import type { MessageWriter } from '@/shared/notices/messages'

interface CommandListSources {
  readonly declaredCommands?: DeclaredCommands | undefined
  readonly writeMessage: MessageWriter
  readonly plugins?: Ref<readonly { readonly id: string; readonly isActive: boolean }[]> | undefined
  readonly rest?:
    | {
        /** A setting as the core last confirmed it. A plugin with no backend is never active, so its switch is read here. */
        readonly getCommittedSetting?: ((path: readonly string[]) => unknown) | undefined
      }
    | undefined
}

export function watchCommandLists(sources: CommandListSources, isOpen: Ref<boolean>): void {
  const { declaredCommands, writeMessage, plugins } = sources
  const getCommittedSetting = sources.rest?.getCommittedSetting
  if (declaredCommands) {
    const loadDeclared = () => {
      declaredCommands
        .load()
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
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

/** Whether the setting of the plugin that switches it on reads true; absent counts as on. */
const isSwitchedOn = (
  pluginId: string,
  getCommittedSetting: ((path: readonly string[]) => unknown) | undefined,
): boolean => getCommittedSetting?.([pluginId, 'enabled']) !== false
