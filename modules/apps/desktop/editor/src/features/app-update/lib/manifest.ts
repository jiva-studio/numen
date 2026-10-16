/**
 * Network fetching and manifest evaluation for application updates.
 */
import type { ReleaseFile, ReleaseManifest, UpdateInfo } from '../model/types'
import { findPlatformFile, isNewerRelease, parseBuildNumber, type ClientPlatform } from './release'

export const DEFAULT_ENDPOINT = '/api/app-update/latest.json'
export const DEFAULT_INTERVAL_MS = 4 * 60 * 60 * 1000 // 4 hours
export const DEFAULT_DOWNLOAD_URL = 'https://numen.md/#get'

export interface ClientContext {
  readonly currentBuild: number
  readonly currentVersion: string
  readonly platform: ClientPlatform
}

export async function fetchManifest(
  fetchClient: typeof fetch,
  url: string,
): Promise<ReleaseManifest | null> {
  try {
    const response = await fetchClient(url, { cache: 'no-cache' })
    if (!response.ok) {
      return null
    }
    return (await response.json()) as ReleaseManifest
  } catch {
    return null
  }
}

export function evaluateManifest(
  manifest: ReleaseManifest,
  context: ClientContext,
): UpdateInfo | null {
  if (context.currentVersion && context.currentVersion === manifest.version) {
    return null
  }

  const remoteBuild = parseBuildNumber(manifest.build)
  if (!isNewerRelease(context.currentBuild, remoteBuild)) {
    return null
  }

  const platformFile: ReleaseFile | undefined = findPlatformFile(manifest.files, context.platform)
  const downloadUrl = platformFile?.url ?? DEFAULT_DOWNLOAD_URL

  return {
    currentVersion: context.currentVersion,
    currentBuild: context.currentBuild,
    latestVersion: manifest.version,
    latestBuild: remoteBuild,
    releasedAt: manifest.released,
    notesUrl: manifest.notes,
    platformFile,
    downloadUrl,
  }
}
