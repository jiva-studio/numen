/**
 * Composable providing periodic application update checks against latest.json.
 */
import { onMounted, onUnmounted, ref, type Ref } from 'vue'
import type { UpdateInfo } from './types'
import {
  evaluateManifest,
  fetchManifest,
  DEFAULT_DOWNLOAD_URL,
  DEFAULT_ENDPOINT,
  DEFAULT_INTERVAL_MS,
} from '../lib/manifest'
import { parseBuildNumber, resolveCurrentPlatform, type ClientPlatform } from '../lib/release'

export interface UpdateOptions {
  readonly endpointUrl?: string
  readonly checkIntervalMs?: number
  readonly currentVersion?: string
  readonly currentBuild?: number
  readonly platform?: ClientPlatform
  readonly fetchFn?: typeof fetch
  readonly openUrl?: (url: string) => void
}

const FALLBACK_VERSION: string = import.meta.env.VITE_NUMEN_VERSION ?? '0.0.0-dev'
const FALLBACK_BUILD: number =
  parseBuildNumber(import.meta.env.VITE_NUMEN_BUILD) || parseBuildNumber(FALLBACK_VERSION)

const runtime = '/wails/runtime.js'
const wails =
  typeof window !== 'undefined'
    ? import(/* @vite-ignore */ runtime).catch(() => undefined)
    : Promise.resolve(undefined)

function openExternalUrl(url: string): void {
  void wails.then((it) => {
    if (it?.Browser?.OpenURL) {
      it.Browser.OpenURL(url)
    } else if (typeof window !== 'undefined') {
      window.open(url, '_blank')
    }
  })
}

export function useAppUpdate(options: UpdateOptions = {}) {
  const endpointUrl = options.endpointUrl ?? DEFAULT_ENDPOINT
  const checkIntervalMs = options.checkIntervalMs ?? DEFAULT_INTERVAL_MS
  const currentVersion = options.currentVersion ?? FALLBACK_VERSION
  const currentBuild = options.currentBuild ?? FALLBACK_BUILD
  const platform = options.platform ?? resolveCurrentPlatform()
  const fetchClient = options.fetchFn ?? (typeof fetch !== 'undefined' ? fetch : undefined)
  const openExternal = options.openUrl ?? openExternalUrl

  const isChecking: Ref<boolean> = ref(false)
  const isDismissed: Ref<boolean> = ref(false)
  const updateInfo: Ref<UpdateInfo | null> = ref(null)
  const hasUpdate: Ref<boolean> = ref(false)
  let timerId: ReturnType<typeof setInterval> | null = null

  async function checkUpdate(): Promise<boolean> {
    if (!fetchClient) {
      return false
    }
    isChecking.value = true
    const manifest = await fetchManifest(fetchClient, endpointUrl)
    const context = { currentBuild, currentVersion, platform }
    const info = manifest ? evaluateManifest(manifest, context) : null
    updateInfo.value = info
    hasUpdate.value = info !== null
    if (info) isDismissed.value = false
    isChecking.value = false
    return info !== null
  }

  function dismissUpdate() {
    isDismissed.value = true
  }
  function openDownload() {
    openExternal(updateInfo.value?.downloadUrl ?? DEFAULT_DOWNLOAD_URL)
  }
  function openReleaseNotes() {
    openExternal(updateInfo.value?.notesUrl ?? DEFAULT_DOWNLOAD_URL)
  }

  onMounted(() => {
    void checkUpdate()
    if (checkIntervalMs > 0) {
      timerId = setInterval(() => {
        void checkUpdate()
      }, checkIntervalMs)
    }
  })

  onUnmounted(() => {
    if (timerId !== null) clearInterval(timerId)
  })

  return {
    isChecking,
    isDismissed,
    hasUpdate,
    updateInfo,
    platform,
    checkUpdate,
    dismissUpdate,
    openDownload,
    openReleaseNotes,
  }
}
