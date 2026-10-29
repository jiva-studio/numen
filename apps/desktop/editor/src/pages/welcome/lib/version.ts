/**
 * What this build of the window calls itself.
 *
 * The number is injected when the page is built; a manual build uses
 * the default value below.
 */
export const VERSION: string = import.meta.env.VITE_NUMEN_VERSION ?? '0.0.0-dev'
