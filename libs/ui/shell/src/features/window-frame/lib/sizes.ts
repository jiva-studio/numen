/**
 * The height of a title bar, in pixels: the macOS window keeps its system buttons centred 16 points
 * from the top, so a strip this high holds them. `BarHeight` of `apps/desktop/internal/frame` and
 * `--numen-title-bar-height` of libs/ui/core/src/shared/tokens/tokens.css hold the same number.
 */
export const TITLE_BAR_HEIGHT = 32

/**
 * The width the system buttons of macOS take at the left of the window, in pixels: the three 14
 * pixel circles span 9 to 69 pixels. The Title bar plugin keeps the same number of its own. Fixed:
 * no scale changes it.
 */
export const MAC_BUTTONS_INSET = 80
