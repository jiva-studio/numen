/** The part of the runtime of the desktop shell the pages use, under the names the shell gives it. */
export interface FrameRuntime {
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

/** Loads the runtime module; resolves to undefined where the page is not served by the desktop shell. */
export type FrameRuntimeLoader = () => Promise<FrameRuntime | undefined>
