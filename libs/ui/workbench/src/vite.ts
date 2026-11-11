/**
 * The build settings every desktop window shares.
 */
import { fileURLToPath, URL } from 'node:url'
import { Features } from 'lightningcss'
import type { PluginOption, UserConfig } from 'vite'

/**
 * The Lightning CSS options of every stylesheet a window is built into. `light-dark()` stays as
 * written: the window chooses a half with `color-scheme`, which the lowered form does not read.
 */
export const LIGHTNING_CSS = { exclude: Features.LightDark } as const

const CSS_OPTIONS = { lightningcss: LIGHTNING_CSS }

/** The packages every window imports, by their path from the window's folder. */
const WINDOW_PACKAGES = {
  '@numen/core-client': '../../../libs/core/client/src/index.ts',
  '@numen/plugin-loader': '../../../libs/plugin/loader/src/index.ts',
  '@numen/ui/styles.css': '../../../libs/ui/components/dist/numen-ui.css',
  '@numen/ui': '../../../libs/ui/components/dist/index.js',
  '@numen/workbench/styles.css': '../../../libs/ui/workbench/dist/numen-shell.css',
  '@numen/workbench': '../../../libs/ui/workbench/dist/index.js',
} as const

/** The host module build of `@numen/plugin-loader/vite` a window is built with. */
export interface HostModuleBuild {
  readonly createHostModuleEntries: () => PluginOption
  readonly hostModuleInputs: Readonly<Record<string, string>>
  readonly getEntryFileName: (chunk: { readonly name: string }) => string
}

/** What one window adds to the config every window shares. */
export interface WindowOptions {
  readonly plugins: readonly PluginOption[]
  readonly hostModules: HostModuleBuild
  readonly outDir: string
  readonly own?: Readonly<Record<string, string>>
}

/**
 * The aliases of a window, resolved against `root`, the URL of its config file. `@/` is the
 * window's own source; `@numen/ui` and `@numen/workbench` are read from their builds. `own` is matched
 * before the shared packages, so a subpath such as `@numen/core-client/testing` is listed there.
 */
export function createWindowAliases(
  root: string,
  own: Readonly<Record<string, string>> = {},
): Record<string, string> {
  const resolve = (path: string) => fileURLToPath(new URL(path, root))
  const packages = Object.entries({ ...own, ...WINDOW_PACKAGES })
  const aliases: Record<string, string> = { '@': resolve('./src') }
  for (const [name, path] of packages) aliases[name] = resolve(path)
  return aliases
}

/**
 * The Vite config of a window, resolved against `root`, the URL of its config file. The build goes
 * into the Go package that embeds it; `assets` is where the vault's own files are requested, so the
 * built files go to `built`.
 */
export function createWindowConfig(root: string, options: WindowOptions): UserConfig {
  const { hostModules } = options
  const resolve = { alias: createWindowAliases(root, options.own) }
  return {
    plugins: [...options.plugins, hostModules.createHostModuleEntries()],
    css: CSS_OPTIONS,
    resolve,
    build: {
      outDir: options.outDir,
      emptyOutDir: true,
      assetsDir: 'built',
      rollupOptions: {
        input: {
          index: fileURLToPath(new URL('./index.html', root)),
          ...hostModules.hostModuleInputs,
        },
        // A host module keeps every export, and the page and the plugins share its one instance.
        preserveEntrySignatures: 'strict',
        output: { entryFileNames: hostModules.getEntryFileName },
      },
    },
  }
}
