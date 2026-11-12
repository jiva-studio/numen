/** The plugins whose frontend the window loads: a plugin that declares none has nothing to fetch. */
export const getFrontendPluginIds = (
  plugins: readonly { readonly id: string; readonly hasFrontend: boolean }[],
): string[] => plugins.filter((plugin) => plugin.hasFrontend).map((plugin) => plugin.id)
