/**
 * The build settings every desktop window shares.
 */
import { fileURLToPath, URL } from 'node:url'
import { Features } from 'lightningcss'
import type { BuildOptions } from 'vite'

/**
 * The Lightning CSS options of every stylesheet a window is built into. `light-dark()` stays as
 * written: the window chooses a half with `color-scheme`, which the lowered form does not read.
 */
export const LIGHTNING_CSS = { exclude: Features.LightDark } as const

/** The packages every window imports, by their path from the window's folder. */
const WINDOW_PACKAGES = {
  '@numen/core-client': '../../../libs/core-client/src/index.ts',
  '@numen/plugin-host': '../../../libs/plugin-host-ts/src/index.ts',
  '@numen/ui/styles.css': '../../../libs/ui/core/dist/numen-ui.css',
  '@numen/ui': '../../../libs/ui/core/dist/index.js',
  '@numen/shell/styles.css': '../../../libs/ui/shell/dist/numen-shell.css',
  '@numen/shell': '../../../libs/ui/shell/dist/index.js',
} as const

/** The host module entries of `@numen/plugin-host/vite` a window is built with. */
export interface HostModuleBuild {
  readonly inputs: Readonly<Record<string, string>>
  readonly getEntryFileName: (chunk: { readonly name: string }) => string
}

/**
 * The aliases of a window, resolved against `root`, the URL of its config file. `@/` is the
 * window's own source; `@numen/ui` and `@numen/shell` are read from their builds. `own` is matched
 * before the shared packages, so a subpath such as `@numen/core-client/testing` is listed there.
 */
export function createWindowAliases(
  root: string,
  own: Readonly<Record<string, string>> = {},
): Record<string, string> {
  const resolve = (path: string) => fileURLToPath(new URL(path, root))
  const packages = Object.entries({ ...own, ...WINDOW_PACKAGES })
  return { '@': resolve('./src'), ...Object.fromEntries(packages.map(([name, path]) => [name, resolve(path)])) }
}

/**
 * The build of a window into `outDir`, the Go package that embeds it. `assets` is where the
 * vault's own files are requested, so the built files go to `built`.
 */
export function createWindowBuild(root: string, outDir: string, host: HostModuleBuild): BuildOptions {
  return {
    outDir,
    emptyOutDir: true,
    assetsDir: 'built',
    rollupOptions: {
      input: { index: fileURLToPath(new URL('./index.html', root)), ...host.inputs },
      // A host module keeps every export, and the page and the plugins share its one instance.
      preserveEntrySignatures: 'strict',
      output: { entryFileNames: host.getEntryFileName },
    },
  }
}
