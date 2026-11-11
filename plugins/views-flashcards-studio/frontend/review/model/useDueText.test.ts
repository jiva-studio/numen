import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useDueText } from './useDueText'

const format = (due: number): string => `${due} to review`

describe('the text of the number of cards due', () => {
  it('is the number as the words format it', () => {
    expect(useDueText(() => 12, () => format).value).toBe('12 to review')
  })

  it('is null while the number is not counted', () => {
    expect(useDueText(() => null, () => format).value).toBeNull()
  })

  it('follows the number', () => {
    const due = ref<number | null>(null)
    const text = useDueText(() => due.value, () => format)

    due.value = 3

    expect(text.value).toBe('3 to review')
  })
})
