/**
 * The state of the dialog that links a device: the link the core holds, read every half second while the dialog is
 * open (a timer of the window), and the refusal the person last saw. A host starts to show its QR code when the dialog
 * opens. A link that waits for its channel for 15 seconds is cancelled and reported as stalled, because the channel
 * comes from the Numen Sync server and the core waits for it as long as the link lives. The dialog holds no key and
 * no secret: the code and the text of the QR code come from the core and are dropped with the dialog.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { Result } from '@numen/result'
import type { LinkInfo, LinkMode, LinkRole } from '@/entities/identity'
import { isEnded } from '../lib/phases'
import type { LinkCalls } from '../lib/types'

const POLL_MS = 500
const STALL_MS = 15_000

export function useLinkSession(
  getCalls: () => LinkCalls,
  role: LinkRole,
  stallMs = STALL_MS,
) {
  const stallPolls = Math.ceil(stallMs / POLL_MS)
  const link = ref<LinkInfo | undefined>(undefined)
  const problem = ref('')
  const isBusy = ref(false)
  const isStalled = ref(false)
  let timer: ReturnType<typeof setInterval> | undefined
  let waits = 0
  let lastMode: LinkMode = 'show'
  let lastText = ''

  const stopPoll = (): void => {
    if (timer !== undefined) clearInterval(timer)
    timer = undefined
  }

  const cancel = async (): Promise<void> => {
    stopPoll()
    if (link.value !== undefined && !isEnded(link.value)) await getCalls().cancel()
  }

  const stall = (): void => {
    isStalled.value = true
    runInBackground(cancel(), () => undefined)
  }

  const poll = async (): Promise<void> => {
    const answer = await getCalls().get()
    if (!answer.ok) return
    link.value = answer.value
    waits = answer.value.phase === 'waitingForChannel' ? waits + 1 : 0
    if (waits >= stallPolls) stall()
    if (isEnded(answer.value)) stopPoll()
  }

  const start = async (mode: LinkMode, scannedText = ''): Promise<void> => {
    lastMode = mode
    lastText = scannedText
    await cancel()
    link.value = undefined
    isStalled.value = false
    waits = 0
    isBusy.value = true
    problem.value = ''
    const answer = await getCalls().start(role, mode, scannedText)
    isBusy.value = false
    if (!answer.ok) {
      problem.value = answer.error
      return
    }
    link.value = answer.value
    timer = setInterval(() => runInBackground(poll(), () => undefined), POLL_MS)
  }

  const leave = async (): Promise<void> => {
    await cancel()
    link.value = undefined
    isStalled.value = false
    problem.value = ''
  }

  const ask = async (call: () => Promise<Result<unknown, string>>): Promise<void> => {
    isBusy.value = true
    const answer = await call()
    isBusy.value = false
    problem.value = answer.ok ? '' : answer.error
    await poll()
  }

  onMounted(() => {
    if (role === 'host') runInBackground(start('show'), () => undefined)
  })
  onBeforeUnmount(() => {
    runInBackground(cancel(), () => undefined)
  })

  const answers = {
    decide: (isApproved: boolean) => ask(() => getCalls().decide(isApproved)),
    sendCode: (code: string) => ask(() => getCalls().enterCode(code)),
    confirmJoin: (isApproved: boolean) => ask(() => getCalls().confirmJoin(isApproved)),
  }

  return {
    link,
    problem,
    isBusy,
    isStalled: computed(() => isStalled.value),
    start,
    retry: () => start(lastMode, lastText),
    leave,
    answers,
  }
}
