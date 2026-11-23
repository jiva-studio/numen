/**
 * The state of the dialog that links a device: the link the core holds, read every half second while the dialog is
 * open (a timer of the window), and the refusal the person last saw. The dialog holds no key and no secret: the
 * code and the text of the QR code come from the core and are dropped with the dialog.
 */
import { computed, onBeforeUnmount, ref } from 'vue'
import type { Result } from '@numen/result'
import type { LinkInfo, LinkMode, LinkRole } from '@/entities/identity'
import { isEnded } from '../lib/phases'
import type { LinkCalls } from '../lib/types'

const POLL_MS = 500

export function useLinkSession(getCalls: () => LinkCalls, role: LinkRole) {
  const link = ref<LinkInfo | undefined>(undefined)
  const problem = ref('')
  const isBusy = ref(false)
  let timer: ReturnType<typeof setInterval> | undefined

  const stopPoll = (): void => {
    if (timer !== undefined) clearInterval(timer)
    timer = undefined
  }

  const poll = async (): Promise<void> => {
    const answer = await getCalls().get()
    if (!answer.ok) return
    link.value = answer.value
    if (isEnded(answer.value)) stopPoll()
  }

  const start = async (mode: LinkMode, scannedText = ''): Promise<void> => {
    isBusy.value = true
    problem.value = ''
    const answer = await getCalls().start(role, mode, scannedText)
    isBusy.value = false
    if (!answer.ok) {
      problem.value = answer.error
      return
    }
    link.value = answer.value
    stopPoll()
    timer = setInterval(() => void poll(), POLL_MS)
  }

  const ask = async (call: () => Promise<Result<unknown, string>>): Promise<void> => {
    isBusy.value = true
    const answer = await call()
    isBusy.value = false
    problem.value = answer.ok ? '' : answer.error
    await poll()
  }

  const cancel = async (): Promise<void> => {
    stopPoll()
    if (link.value !== undefined && !isEnded(link.value)) await getCalls().cancel()
  }

  onBeforeUnmount(() => {
    void cancel()
  })

  return { link, problem, isBusy, isChoosing: computed(() => link.value === undefined), start, ask, cancel }
}
