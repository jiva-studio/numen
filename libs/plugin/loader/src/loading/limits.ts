/**
 * The two limits of a plugin start. The window is drawn at INTERFACE_WAIT_MS whatever has started,
 * and a plugin that is not ready by then goes on starting. A plugin that has not started within
 * PLUGIN_START_LIMIT_MS counts as failed; a command of a plugin waits for it as long.
 */

/** How long a window waits for the plugins of its interface before it is drawn without the late ones. */
export const INTERFACE_WAIT_MS = 3_000

/** How long a plugin may take to start before it counts as failed; the backend activation uses the same. */
export const PLUGIN_START_LIMIT_MS = 30_000
