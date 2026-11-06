/**
 * The modules a window serves to plugin frontends, each at its path on the window's origin.
 *
 * A plugin bundle imports these by path and carries no copy of them, so every plugin runs on the
 * window's own instance: one Vue, one `@numen/ui`, one SDK.
 */
export const HOST_MODULES = {
  vue: '/sdk/vue.js',
  '@numen/plugin-sdk': '/sdk/plugin-sdk.js',
  '@numen/ui': '/sdk/ui.js',
} as const

export type HostModuleName = keyof typeof HOST_MODULES
