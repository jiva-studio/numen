/**
 * The width the system buttons of macOS take at the left of the bar, in pixels. The three 14 pixel
 * circles of MacTitleBarHidden span 9 to 69 pixels, and the 4 pixel space after this width
 * (titleBar.css) keeps the actions clear of them. Fixed: no scale changes it.
 */
export const NATIVE_MAC_INSET = 80

/** The width kept for the title when the room for the actions is worked out, in pixels. */
export const TITLE_MIN_WIDTH = 128
