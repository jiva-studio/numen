/** What the listed commands are read from: the host of the plugins. */
interface CommandHost {
  readonly commands: { getPaletteCommands(): readonly { readonly id: string }[] }
}

/** The ids of the commands the plugins list in the palette now. */
export const getListedCommandIds = (pluginHost: CommandHost): string[] =>
  pluginHost.commands.getPaletteCommands().map((one) => one.id)

/** Whether the plugins list a command of this id in the palette now. */
export const isCommandListed = (pluginHost: CommandHost, id: string): boolean =>
  pluginHost.commands.getPaletteCommands().some((one) => one.id === id)
