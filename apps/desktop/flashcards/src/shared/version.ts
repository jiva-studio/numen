/**
 * The version this build of the window reports.
 *
 * The number is injected when the page is built; a page built without it uses
 * the default below. The two windows ship together, so this is also the
 * editor's version.
 */
export const VERSION: string = import.meta.env.VITE_NUMEN_VERSION ?? '0.0.0-dev'
