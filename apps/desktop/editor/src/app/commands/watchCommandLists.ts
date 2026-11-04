/**
 * Loads the commands the plugins declare and the menu of the file in front, as the
 * window starts and each time the palette opens. The declared commands are read again when a
 * plugin starts or stops, so the shortcuts of its commands follow it.
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
  readonly plugins?: Ref<readonly { readonly id: string; readonly isActive: boolean }[]> | undefined
}

export function watchCommandLists(
  sources: CommandListSources,
  isOpen: Ref<boolean>,
  getTarget: () => CommandTarget,
): void {
  const { declaredCommands, fileMenuCommands, writeMessage, plugins } = sources
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
      watch(() => plugins.value.map((one) => `${one.id}:${one.isActive}`).join(), loadDeclared)
    }
  }

  // Stryker disable next-line ConditionalExpression: fileMenuCommands is optional in unit test setups
  if (fileMenuCommands) {
    watch(isOpen, (open) => {
      if (!open) return
      fileMenuCommands
        .load(getTarget().file)
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    })
  }
}
