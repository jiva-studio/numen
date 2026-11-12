/**
 * What a command is carried out over: the vault and the target of the palette, and the invocation
 * that carries it out.
 */
import type { Selector } from '@numen/plugin-loader'

/** The vault a command is over: the identity the list gives it, and its name. */
export interface VaultRef {
  readonly id: string
  readonly name: string
}

/**
 * The runs this build cannot perform. The application reports them the first
 * time one is requested; this window omits it from then on.
 */
export interface RunSupport {
  /** Whether this build can do a run at all. A view drawing it follows the answer. */
  canRun(run: string): boolean
  /** A run the application answered it cannot do at all. */
  cannotRun(run: string): void
}

/**
 * The current context: the active tab and the note it resolves to. For an agent
 * tab, the note the plex is focused on.
 */
export interface CommandTarget {
  readonly tab: string
  readonly kind: string | null
  readonly path: string
  readonly title: string
  readonly file: string
  readonly line?: number
  readonly selector?: Selector
  readonly isFolder?: boolean
  readonly others?: readonly string[]
  readonly isTargetNamed?: boolean
  readonly vault: VaultRef
  readonly isReady: boolean
}

/** What a command was invoked over: the target of the palette without whether it is ready. */
export interface InvocationTarget {
  readonly path: string
  readonly vault: VaultRef
  readonly title: string
  readonly file: string
  readonly line?: number
  readonly selector?: Selector
  readonly isFolder?: boolean
  readonly isTargetNamed?: boolean
  readonly others: readonly string[]
  readonly kind: string | null
  readonly tab: string
}

/**
 * One command as it is carried out: what it is over, the tab the note is open in, and what was
 * typed for it (the name) or set for it (the values).
 */
export interface CommandInvocation {
  readonly id: string
  readonly target: InvocationTarget
  readonly note: string | null
  readonly name: string
  readonly values?: Readonly<Record<string, string>>
}
