/**
 * The width the system buttons of macOS take at the left of the bar, in pixels. The last of the three
 * 13.5 pixel circles ends 78.5 pixels from the edge of the window, and the 16 pixel space after this
 * width (titleBar.css) keeps the actions clear of it. Fixed: no scale changes it.
 */
export const NATIVE_MAC_INSET = 72

/** The width kept for the title when the room for the actions is worked out, in pixels. */
export const TITLE_MIN_WIDTH = 128
