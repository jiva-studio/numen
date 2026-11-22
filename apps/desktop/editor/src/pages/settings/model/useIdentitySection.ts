/**
 * The state of the identity section: showing the recovery words of a new identity and checking
 * three of them, restoring from typed words, and deleting the identity from this device. The words
 * live in this state only while a setup is open; leaving the page or cancelling drops them.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'
import type { InstallationIdentity } from '../types'
import { WORDS as words } from '../words'

type Step = 'idle' | 'setup' | 'restore'

const WORD_COUNT = 12
const CONFIRMED_COUNT = 3

/** A number in [0, limit) from the random source of the browser, without modulo bias. */
function getRandomBelow(limit: number): number {
  const range = 0x100000000
  const largest = range - (range % limit)
  const buffer = new Uint32Array(1)
  let value = 0
  do {
    crypto.getRandomValues(buffer)
    value = buffer[0] ?? 0
  } while (value >= largest)
  return value % limit
}

/** `count` distinct positions in 1..`total`, in ascending order. */
function pickPositions(count: number, total: number): readonly number[] {
  const pool = Array.from({ length: total }, (_, index) => index + 1)
  const picked: number[] = []
  while (picked.length < count) {
    const [position] = pool.splice(getRandomBelow(pool.length), 1)
    if (position !== undefined) picked.push(position)
  }
  return picked.sort((a, b) => a - b)
}

const normalize = (word: string): string => word.trim().toLowerCase()

export function useIdentitySection(getIdentity: () => InstallationIdentity) {
  const identity = computed(() => getIdentity().identity.value)
  const step = ref<Step>('idle')
  const recoveryWords = ref<readonly string[]>([])
  const positions = ref<readonly number[]>([])
  const entered = ref<Readonly<Record<number, string>>>({})
  const restoreText = ref('')
  const problem = ref('')
  const isBusy = ref(false)
  const isAskingDelete = ref(false)

  onMounted(() => {
    runInBackground(getIdentity().load(), (caughtError) => {
      problem.value = formatError(caughtError)
    })
  })

  onBeforeUnmount(() => {
    reset()
  })

  function reset(): void {
    step.value = 'idle'
    recoveryWords.value = []
    positions.value = []
    entered.value = {}
    restoreText.value = ''
    problem.value = ''
    isAskingDelete.value = false
  }

  /** Runs one call of the core while the section is busy; a failure that is thrown becomes a problem. */
  const run = async <T>(call: () => Promise<Result<T, string>>): Promise<Result<T, string>> => {
    isBusy.value = true
    problem.value = ''
    try {
      const answer = await call()
      if (!answer.ok) problem.value = answer.error
      return answer
    } catch (caughtError) {
      const error = formatError(caughtError)
      problem.value = error
      return { ok: false, error }
    } finally {
      isBusy.value = false
    }
  }

  const startSetup = async (): Promise<void> => {
    if (isBusy.value) return
    const answer = await run(() => getIdentity().generateWords())
    if (!answer.ok) return
    recoveryWords.value = answer.value
    positions.value = pickPositions(CONFIRMED_COUNT, WORD_COUNT)
    entered.value = {}
    step.value = 'setup'
  }

  const startRestore = (): void => {
    reset()
    step.value = 'restore'
  }

  const setEntered = (position: number, value: string): void => {
    entered.value = { ...entered.value, [position]: value }
    problem.value = ''
  }

  const setRestoreText = (value: string): void => {
    restoreText.value = value
    problem.value = ''
  }

  const isConfirmed = (): boolean =>
    positions.value.every(
      (position) => normalize(entered.value[position] ?? '') === recoveryWords.value[position - 1],
    )

  const keep = async (): Promise<void> => {
    if (isBusy.value) return
    if (!isConfirmed()) {
      problem.value = words.identityNotValid
      return
    }
    const answer = await run(() => getIdentity().create(recoveryWords.value))
    if (answer.ok) reset()
  }

  const restore = async (): Promise<void> => {
    if (isBusy.value) return
    const typed = restoreText.value.split(/\s+/).filter((word) => word !== '')
    const answer = await run(() => getIdentity().create(typed))
    if (answer.ok) reset()
  }

  const confirmDelete = async (): Promise<void> => {
    if (isBusy.value) return
    isAskingDelete.value = false
    await run(() => getIdentity().delete())
  }

  return {
    state: { step, recoveryWords, positions, entered, restoreText, problem, isBusy, isAskingDelete },
    view: { identity },
    actions: {
      startSetup,
      startRestore,
      cancel: reset,
      setEntered,
      setRestoreText,
      keep,
      restore,
      askDelete: () => {
        isAskingDelete.value = true
      },
      cancelDelete: () => {
        isAskingDelete.value = false
      },
      confirmDelete,
    },
  }
}
