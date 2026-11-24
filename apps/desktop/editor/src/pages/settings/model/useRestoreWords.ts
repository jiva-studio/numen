/**
 * The twelve recovery words typed to restore an identity, one field each. Several words typed or
 * pasted into one field fill it and the fields after it. Typing clears the problem shown.
 */
import { ref, type Ref } from 'vue'

const WORD_COUNT = 12

const createEmptyWords = (): readonly string[] => Array.from({ length: WORD_COUNT }, () => '')

export const splitWords = (text: string): string[] =>
  text
    .split(/\s+/)
    .filter((word) => word !== '')
    .map((word) => word.toLowerCase())

export function useRestoreWords(problem: Ref<string>) {
  const entered = ref<readonly string[]>(createEmptyWords())

  const setWord = (index: number, value: string): void => {
    const typed = splitWords(value)
    const next = [...entered.value]
    if (typed.length <= 1) {
      next[index] = value
    } else {
      const room = WORD_COUNT - index
      for (const [at, word] of typed.slice(0, room).entries()) next[index + at] = word
      if (typed.length > room) next[WORD_COUNT - 1] = typed.slice(room - 1).join(' ')
    }
    entered.value = next
    problem.value = ''
  }

  const clear = (): void => {
    entered.value = createEmptyWords()
  }

  /** The words typed, in order, without the empty fields. */
  const readWords = (): string[] => entered.value.flatMap(splitWords)

  return { entered, setWord, clear, readWords }
}
