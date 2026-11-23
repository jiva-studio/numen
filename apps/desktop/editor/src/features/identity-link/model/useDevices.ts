/** The device list kept fresh while it is on the screen, and the question before a revoke. */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { DeviceInfo } from '@/entities/identity'
import { WORDS as words } from '../words'
import type { DeviceCalls } from './types'

const REFRESH_MS = 2000

export type RevokeReason = 'lost' | 'stolen'

export function useDevices(getCalls: () => DeviceCalls, hasIdentity: () => boolean) {
  const asked = ref<DeviceInfo | undefined>(undefined)
  const reason = ref<RevokeReason>('lost')
  const since = ref('')
  const question = ref('')
  const problem = ref('')
  let timer: ReturnType<typeof setInterval> | undefined

  const refresh = (): void => {
    if (!hasIdentity()) return
    runInBackground(getCalls().load(), () => undefined)
  }

  const ask = (device: DeviceInfo): void => {
    reason.value = 'lost'
    since.value = ''
    question.value = ''
    asked.value = device
  }

  const cancel = (): void => {
    asked.value = undefined
  }

  const confirmRevoke = async (): Promise<void> => {
    const device = asked.value
    if (device === undefined) return
    const started = reason.value === 'stolen' ? new Date(`${since.value}T00:00:00`) : undefined
    if (started !== undefined && (since.value === '' || Number.isNaN(started.getTime()))) {
      question.value = words.deviceRevokeNeedsDate
      return
    }
    asked.value = undefined
    const answer = await getCalls().revoke(
      device.fingerprint,
      started === undefined
        ? { reason: 'lost' }
        : { reason: 'stolen', since: started.toISOString() },
    )
    problem.value = answer.ok ? '' : answer.error
  }

  onMounted(() => {
    refresh()
    timer = setInterval(refresh, REFRESH_MS)
  })

  onBeforeUnmount(() => {
    if (timer !== undefined) clearInterval(timer)
  })

  return { asked, reason, since, question, problem, refresh, ask, cancel, confirmRevoke }
}
