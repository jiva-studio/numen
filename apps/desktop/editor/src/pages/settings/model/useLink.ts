/**
 * The state of the dialog that links a device: the link the core holds, read every half second while the dialog is
 * open (a timer of the window), and the answers of the person. The dialog holds no key and no secret: the
 * code and the text of the QR code come from the core and are dropped with the dialog.
 */
import { computed, onBeforeUnmount, ref } from 'vue'
import type { LinkInfo, LinkMode, LinkRole } from '@/entities/identity'
import type { InstallationLink } from '../types'

const POLL_MS = 500
const ENDED = new Set(['linked', 'failed'])

export function useLink(getLink: () => InstallationLink, role: LinkRole, getBlock: () => string = () => '') {
  const link = ref<LinkInfo | undefined>(undefined)
  const problem = ref('')
  const pastedText = ref('')
  const typedCode = ref('')
  const isBusy = ref(false)
  const isScanning = ref(false)
  let timer: ReturnType<typeof setInterval> | undefined

  const stopPolling = (): void => {
    if (timer !== undefined) clearInterval(timer)
    timer = undefined
  }

  const poll = async (): Promise<void> => {
    const answer = await getLink().get()
    if (!answer.ok) return
    link.value = answer.value
    if (ENDED.has(answer.value.phase)) stopPolling()
  }

  const start = async (mode: LinkMode, text = ''): Promise<void> => {
    const blocked = getBlock()
    if (blocked !== '') {
      problem.value = blocked
      return
    }
    isBusy.value = true
    problem.value = ''
    const answer = await getLink().start(role, mode, text)
    isBusy.value = false
    if (!answer.ok) {
      problem.value = answer.error
      return
    }
    link.value = answer.value
    stopPolling()
    timer = setInterval(() => void poll(), POLL_MS)
  }

  const answer = async (call: () => Promise<{ ok: boolean; error?: string }>): Promise<void> => {
    isBusy.value = true
    const result = await call()
    isBusy.value = false
    problem.value = result.ok ? '' : (result.error ?? '')
    await poll()
  }

  const cancel = async (): Promise<void> => {
    stopPolling()
    if (link.value !== undefined && !ENDED.has(link.value.phase)) await getLink().cancel()
  }

  onBeforeUnmount(() => {
    void cancel()
  })

  const isChoosing = computed(() => link.value === undefined)

  return {
    link,
    problem,
    pastedText,
    typedCode,
    isBusy,
    isScanning,
    isChoosing,
    start,
    cancel,
    decide: (approve: boolean) => answer(() => getLink().decide(approve)),
    sendCode: () => answer(() => getLink().enterCode(typedCode.value)),
    confirmJoin: (approve: boolean) => answer(() => getLink().confirmJoin(approve)),
  }
}
