import type { Component } from 'vue'
import type { PluginHost } from '@numen/plugin-loader'
import { getNamedIcon } from '@/shared/ui/icons'
import { getPaletteId, type DeclaredCommand } from '@/entities/plugin'

/** The icon a command of a plugin is drawn with: the one it registered, else the one it declares. */
export const createCommandIconFinder =
  (
    pluginHost: Pick<PluginHost, 'commands'>,
    declaredCommands: { readonly commands: { readonly value: readonly DeclaredCommand[] | undefined } },
  ) =>
  (id: string): Component | undefined => {
    const name =
      pluginHost.commands.getPaletteCommands().find((one) => one.id === id)?.icon ??
      declaredCommands.commands.value?.find((one) => getPaletteId(one) === id)?.icon
    return name ? getNamedIcon(name) : undefined
  }
