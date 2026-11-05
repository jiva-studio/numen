/**
 * The words the window appearance and the sync of the title and the filename report in.
 */
// Stryker disable StringLiteral: interface copy

/** Interface text of the appearance settings. */
export interface AppearanceWords {
  /** The themes could not be listed, and one theme's file could not be read. */
  readonly unlisted: string
  readonly themeUnreadable: string
}

/** Interface text of the sync settings. */
export interface SyncWords {
  /** The setting could not be written. */
  readonly unturned: string
}
