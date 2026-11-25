/**
 * The state of the dialog that links a device: the link the core holds, read every half second while the dialog is
 * open (a timer of the window), and the refusal the person last saw. A host starts to show its QR code when the dialog
 * opens. A link that waits for its channel for 15 seconds is cancelled and reported as stalled, because the channel
 * comes from the Numen Sync server and the core waits for it as long as the link lives. The dialog holds no key and
 * no secret: the text of the QR code come from the core and are dropped with the dialog.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { Result } from '@numen/result'
import type { LinkInfo, LinkMode, LinkRole } from '@/entities/identity'
import { createLinkWatch } from './createLinkWatch'
import type { LinkCalls } from '../lib/types'

const STALL_MS = 15_000

export function useLinkSession(getCalls: () => LinkCalls, role: LinkRole, stallMs = STALL_MS) {
  const link = ref<LinkInfo | undefined>(undefined)
  const problem = ref('')
  const isBusy = ref(false)
  const isStalled = ref(false)
  let lastMode: LinkMode = 'show'
  let lastText = ''
  let isDisposed = false
  let generation = 0

  const { poll, cancel } = createLinkWatch(getCalls, link, isStalled, stallMs)

  const requestStart = async (mode: LinkMode, scannedText: string): Promise<boolean> => {
    isBusy.value = true
    const started = generation
    try {
      const answer = await getCalls().start(role, mode, scannedText)
      if (isDisposed && answer.ok) await getCalls().cancel()
      if (isDisposed || started !== generation) return false
      if (!answer.ok) {
        problem.value = answer.error
        return false
      }
      link.value = answer.value
      return true
    } finally {
      if (started === generation) isBusy.value = false
    }
  }

  const start = async (mode: LinkMode, scannedText = ''): Promise<void> => {
    lastMode = mode
    lastText = scannedText
    generation += 1
    const started = generation
    poll.reset()
    await cancel()
    if (isDisposed || started !== generation) return
    link.value = undefined
    isStalled.value = false
    problem.value = ''
    if (await requestStart(mode, scannedText)) poll.start()
  }

  const ask = async (call: () => Promise<Result<unknown, string>>): Promise<void> => {
    isBusy.value = true
    try {
      const answer = await call()
      problem.value = answer.ok ? '' : answer.error
    } finally {
      isBusy.value = false
    }
    await poll.readOnce()
  }

  onMounted(() => {
    if (role === 'host') runInBackground(start('show'), () => undefined)
  })
  onBeforeUnmount(() => {
    isDisposed = true
    poll.dispose()
    runInBackground(cancel(), () => undefined)
  })

  const answers = {
    decide: (isApproved: boolean) => ask(() => getCalls().decide(isApproved)),
    confirmJoin: (isApproved: boolean) => ask(() => getCalls().confirmJoin(isApproved)),
  }

  return {
    link,
    problem,
    isBusy,
    isStalled: computed(() => isStalled.value),
    start,
    retry: () => start(lastMode, lastText),
    answers,
  }
}
