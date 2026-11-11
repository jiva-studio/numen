/** The Wails runtime module as the pages use it. */

/** The part of the Wails runtime module the pages use, under the names Wails gives it. */
export interface WailsRuntime {
  readonly Window?: {
    readonly Focus?: () => Promise<void>
    readonly Minimise?: () => Promise<void>
    readonly ToggleMaximise?: () => Promise<void>
    readonly Close?: () => Promise<void>
    readonly IsMaximised?: () => Promise<boolean>
    readonly IsFullscreen?: () => Promise<boolean>
  }
  readonly System?: {
    readonly IsMac?: () => boolean
    readonly IsWindows?: () => boolean
    readonly IsLinux?: () => boolean
    readonly Environment?: () => Promise<{ readonly OSInfo?: { readonly Version?: string } }>
  }
  readonly Browser?: { readonly OpenURL?: (href: string) => void | Promise<void> }
}

/** Loads the runtime module; resolves to undefined where the page is not served by Wails. */
export type WailsRuntimeLoader = () => Promise<WailsRuntime | undefined>
