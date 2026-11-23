/**
 * The two fields a person types the words of two sheets into, for checking, proving or restoring.
 */
import { ref } from 'vue'
import { splitSheetText } from './useRecoveryKit'

type Texts = readonly [string, string]

export function useSheetFields() {
  const texts = ref<Texts>(['', ''])

  const setText = (index: number, value: string): void => {
    const [first, second] = texts.value
    texts.value = index === 0 ? [value, second] : [first, value]
  }

  const clear = (): void => {
    texts.value = ['', '']
  }

  /** The sheets that hold text, each as its words. */
  const readSheets = (): string[][] =>
    texts.value.map((text) => splitSheetText(text)).filter((sheet) => sheet.length > 0)

  return { texts, setText, clear, readSheets }
}
