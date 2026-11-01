/**
 * Loads the commands the plugins declare and the menu of the file in front, as the
 * window starts and each time the palette opens.
 */
import { watch, type Ref } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import type { CommandTarget } from '@/features/command-palette'
import type { DeclaredCommands } from './declaredCommands'
import type { FileMenuCommands } from './fileMenuCommands'
import type { MessageWriter } from '@/shared/notices/messages'

interface CommandListSources {
  readonly declaredCommands?: DeclaredCommands | undefined
  readonly fileMenuCommands?: FileMenuCommands | undefined
  readonly writeMessage: MessageWriter
}

export function watchCommandLists(
  sources: CommandListSources,
  isOpen: Ref<boolean>,
  getTarget: () => CommandTarget,
): void {
  const { declaredCommands, fileMenuCommands, writeMessage } = sources
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
  }

  if (fileMenuCommands) {
    watch(isOpen, (open) => {
      if (!open) return
      fileMenuCommands
        .load(getTarget().file)
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    })
  }
}
