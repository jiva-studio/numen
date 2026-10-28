/**
 * Data structures describing published releases and client update state.
 */

export interface ReleaseFile {
  readonly name: string
  readonly platform: string
  readonly arch: string
  readonly kind: string
  readonly url: string
  readonly size: number
  readonly sha256: string
}

export interface ReleaseManifest {
  readonly version: string
  readonly build: number
  readonly commit: string
  readonly stamp?: string | undefined
  readonly channel?: string | undefined
  readonly notes?: string | undefined
  readonly released?: string | undefined
  readonly files?: readonly ReleaseFile[] | undefined
}

export interface UpdateInfo {
  readonly currentVersion: string
  readonly currentBuild: number
  readonly latestVersion: string
  readonly latestBuild: number
  readonly releasedAt?: string | undefined
  readonly notesUrl?: string | undefined
  readonly platformFile?: ReleaseFile | undefined
  readonly downloadUrl: string
}
