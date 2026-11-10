/**
 * The application build a plugin runs in, as the context reports it.
 */
import type { AppInfo } from '@numen/plugin-sdk'

/** The name of the application, as its windows are titled. */
const APP_NAME = 'numen'

/** The version a build made without an injected version calls itself. */
export const DEV_VERSION = '0.0.0-dev'

export interface AppInfoSource {
  readonly version?: string | undefined
  readonly build?: string | undefined
  readonly getPlatform: () => AppInfo['platform']
}

function parseBuildNumber(text: string | undefined): number {
  // Stryker disable next-line StringLiteral: any non-numeric fallback parses to NaN and returns 0
  const value = Number.parseInt(text ?? '', 10)
  // Stryker disable next-line EqualityOperator: 0 maps to 0 under both > 0 and >= 0
  return Number.isSafeInteger(value) && value > 0 ? value : 0
}

export function createAppInfo(source: AppInfoSource): AppInfo {
  return {
    name: APP_NAME,
    version: source.version || DEV_VERSION,
    build: parseBuildNumber(source.build),
    get platform() {
      return source.getPlatform()
    },
  }
}
