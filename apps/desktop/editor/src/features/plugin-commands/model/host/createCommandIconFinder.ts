import type { Component } from 'vue'
import type { PluginHost } from '@numen/plugin-loader'
import { getNamedIcon } from '@/shared/ui/icons'
import type { DeclaredCommands } from '../declaredCommands'
import { getPaletteId } from '../declaredCommands'

/** The icon a command of a plugin is drawn with: the one it registered, else the one it declares. */
export const createCommandIconFinder =
  (
    pluginHost: Pick<PluginHost, 'commands'>,
    declaredCommands: Pick<DeclaredCommands, 'commands'>,
  ) =>
  (id: string): Component | undefined => {
    const name =
      pluginHost.commands.getPaletteCommands().find((one) => one.id === id)?.icon ??
      declaredCommands.commands.value?.find((one) => getPaletteId(one) === id)?.icon
    return name ? getNamedIcon(name) : undefined
  }
