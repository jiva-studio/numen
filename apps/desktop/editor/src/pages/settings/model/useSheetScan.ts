/**
 * The scan of two sheets of a recovery kit with a camera that stays on: each valid sheet fills the next
 * slot, a repeated sheet or a sheet of another kit is refused in a line while the scan goes on, the
 * second valid sheet starts the restore at once, and a sheet scanned after a failed restore starts over.
 */
import { computed, ref, type Ref } from 'vue'
import type { KitProof } from '@/entities/identity'
import { addScannedSheet, type ScannedSheets } from './readSheetCode'

export function useSheetScan(problem: Ref<string>, restore: (proof: KitProof) => Promise<void>) {
  const scanned = ref<ScannedSheets | undefined>(undefined)
  const lines = ref<string[]>([])
  const note = ref('')

  const slots = computed(() => [0, 1].map((at) => scanned.value?.sheets[at]))

  const scan = async (text: string): Promise<void> => {
    if (problem.value !== '') retry()
    else if (lines.value.length >= 2) return
    const step = addScannedSheet(scanned.value, text)
    if (step.next === undefined) {
      note.value = step.problem
      return
    }
    note.value = ''
    scanned.value = step.next
    lines.value = [...lines.value, step.line]
    if (lines.value.length === 2) {
      await restore({ sheets: lines.value.map((line) => line.split(' ')) })
    }
  }

  const clear = (): void => {
    scanned.value = undefined
    lines.value = []
    note.value = ''
  }

  const retry = (): void => {
    clear()
    problem.value = ''
  }


  return { slots, note, scan, clear }
}
