/**
 * Public exports for the app-update feature module.
 */
export { default as UpdateIndicator } from './ui/UpdateIndicator.vue'
export { useAppUpdate } from './model/useAppUpdate'
export type { ReleaseFile, ReleaseManifest, UpdateInfo } from './model/types'
export { isNewerRelease, parseBuildNumber, resolveCurrentPlatform } from './lib/release'
