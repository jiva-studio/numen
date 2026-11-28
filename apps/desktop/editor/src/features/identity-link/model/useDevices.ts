/** The device list kept fresh while it is on the screen, and the question before a revoke. */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { DeviceInfo } from '@/entities/identity'
import type { DeviceCalls } from './types'

const REFRESH_MS = 2000

export function useDevices(getCalls: () => DeviceCalls) {
  const asked = ref<DeviceInfo | undefined>(undefined)
  const problem = ref('')
  let timer: ReturnType<typeof setInterval> | undefined

  const refresh = (): void => {
    runInBackground(getCalls().load(), () => undefined)
  }

  const confirmRevoke = async (): Promise<void> => {
    const device = asked.value
    asked.value = undefined
    if (device === undefined) return
    const answer = await getCalls().revoke(device.fingerprint)
    problem.value = answer.ok ? '' : answer.error
  }

  onMounted(() => {
    refresh()
    timer = setInterval(refresh, REFRESH_MS)
  })

  onBeforeUnmount(() => {
    if (timer !== undefined) clearInterval(timer)
  })

  return { asked, problem, refresh, confirmRevoke }
}
