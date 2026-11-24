/**
 * The check that a person wrote the recovery words down: three positions picked at random, and the
 * words typed for them compared with the words of the identity being set up.
 */
import { computed, ref, type Ref } from 'vue'

const WORD_COUNT = 12
const CONFIRMED_COUNT = 3

const normalize = (word: string): string => word.trim().toLowerCase()

function getRandomBelow(limit: number): number {
  const range = 0x100000000
  const largest = range - (range % limit)
  const buffer = new Uint32Array(1)
  let value: number
  do {
    crypto.getRandomValues(buffer)
    value = buffer[0] ?? 0
  } while (value >= largest)
  return value % limit
}

function pickPositions(count: number, total: number): readonly number[] {
  const pool = Array.from({ length: total }, (_, index) => index + 1)
  const picked: number[] = []
  while (picked.length < count) {
    const [position] = pool.splice(getRandomBelow(pool.length), 1)
    if (position !== undefined) picked.push(position)
  }
  return picked.sort((a, b) => a - b)
}

export function useWordsConfirmation(recoveryWords: Readonly<Ref<readonly string[]>>) {
  const positions = ref<readonly number[]>([])
  const entered = ref<Readonly<Record<number, string>>>({})
  const wrong = ref<readonly number[]>([])

  const pick = (): void => {
    positions.value = pickPositions(CONFIRMED_COUNT, WORD_COUNT)
    entered.value = {}
    wrong.value = []
  }

  const clear = (): void => {
    positions.value = []
    entered.value = {}
    wrong.value = []
  }

  const setEntered = (position: number, value: string): void => {
    entered.value = { ...entered.value, [position]: value }
    wrong.value = wrong.value.filter((one) => one !== position)
  }

  const isRight = (position: number): boolean =>
    normalize(entered.value[position] ?? '') === recoveryWords.value[position - 1]

  /** Marks the positions whose typed word is not the word at that place; true when none is. */
  const verify = (): boolean => {
    wrong.value = positions.value.filter((position) => !isRight(position))
    return wrong.value.length === 0
  }

  const isComplete = computed(() =>
    positions.value.every((position) => (entered.value[position] ?? '').trim() !== ''),
  )

  return { positions, entered, wrong, isComplete, pick, clear, setEntered, verify }
}
