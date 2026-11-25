import { createStreamReader } from '@numen/core-client'
import type { SlotLayout } from '@numen/plugin-loader'
import type { WindowClients } from '@/shared/api/clients'

type SettingsService = WindowClients['settingsService']

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

function readLayout(valueJson: string | undefined): SlotLayout | undefined {
  if (!valueJson) return undefined
  try {
    const value: SlotLayout | readonly string[] | null = JSON.parse(valueJson)
    return typeof value === 'object' && value !== null && !Array.isArray(value)
      ? (value as SlotLayout)
      : undefined
  } catch {
    return undefined
  }
}

/** The layout of the slots each plugin draws, from its setting of the type slots; a plugin without one has none. */
export async function readSlotLayouts(
  settings: SettingsService,
): Promise<ReadonlyMap<string, SlotLayout>> {
  const { plugins } = await settings.getSettings({})
  return new Map(
    plugins.flatMap((one) => {
      const layout = readLayout(one.settings.find((setting) => setting.type === 'slots')?.valueJson)
      return layout ? [[one.id, layout] as const] : []
    }),
  )
}

/**
 * Calls `refresh` for each batch of the stream of changes of the settings, the empty one it opens with
 * too, and follows the stream again after it was lost, until `close`.
 */
export function createSettingsFollower(
  settings: SettingsService,
  refresh: () => Promise<void>,
  wait: (ms: number) => Promise<void> = sleep,
) {
  const abortController = new AbortController()
  let isOpen = true
  const follow = createStreamReader({ isOpen: () => isOpen, setLost: () => undefined, wait })
  return {
    start: () =>
      follow(() => settings.watchSettings({}, { signal: abortController.signal }), refresh),
    close: () => {
      isOpen = false
      abortController.abort()
    },
  }
}
