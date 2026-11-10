/**
 * The words the window appearance reports in.
 */
// Stryker disable StringLiteral: interface copy

/** Interface text of the appearance settings. */
export interface AppearanceWords {
  readonly unlisted: string
  readonly themeUnreadable: string
}

export const APPEARANCE_WORDS: AppearanceWords = {
  unlisted: 'Themes unavailable',
  themeUnreadable: 'Theme unreadable',
}
