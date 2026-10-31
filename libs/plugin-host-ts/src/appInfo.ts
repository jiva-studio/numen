/**
 * The application build a plugin runs in, as the context reports it.
 */
import type { AppInfo } from '@numen/plugin-sdk'

/** The version a build made without an injected version calls itself. */
export const DEV_VERSION = '0.0.0-dev'

export interface AppInfoSource {
  /** The version injected while the page is built. */
  readonly version?: string | undefined
  /** The build number injected while the page is built, as text. */
  readonly build?: string | undefined
  readonly userAgent: string
}

/** The build number a text holds; 0 where it holds none. */
export function parseBuildNumber(text: string | undefined): number {
  const value = Number.parseInt((text ?? '').trim(), 10)
  return Number.isSafeInteger(value) && value > 0 ? value : 0
}

/** The operating system a user agent names; linux for any it does not recognize. */
export function getPlatform(userAgent: string): AppInfo['platform'] {
  const agent = userAgent.toLowerCase()
  if (agent.includes('mac') || agent.includes('darwin')) return 'darwin'
  if (agent.includes('win')) return 'windows'
  return 'linux'
}

export function createAppInfo(source: AppInfoSource): AppInfo {
  return {
    version: source.version || DEV_VERSION,
    build: parseBuildNumber(source.build),
    platform: getPlatform(source.userAgent),
  }
}
