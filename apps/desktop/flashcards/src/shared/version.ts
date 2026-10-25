/**
 * What this build of the window calls itself.
 *
 * The number is written in when the page is built, and a page built by hand
 * carries what stands here. The two windows ship together, so this is the
 * editor's number as well.
 */
export const VERSION: string = import.meta.env.VITE_NUMEN_VERSION ?? '0.0.0-dev'
