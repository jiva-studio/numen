import { describe, expect, it } from 'vitest'
import { nextTick, reactive, ref } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
import { useBookPlace, type PlacedTab } from './useBookPlace'

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/** A reader whose CFIs resolve when the test triggers them, in any order. */
const createReader = (isLoading = false) => {
  const offset = ref(0)
  const loading = ref(isLoading)
  const visited: string[] = []
  const pending = new Map<number, (cfi: string | null) => void>()
  const reader = {
    offset,
    isLoading: loading,
    getCfi: (position: number) =>
      new Promise<string | null>((resolve) => pending.set(position, resolve)),
    goToCfi: async (cfi: string) => {
      visited.push(cfi)
      return true
    },
  }
  return { reader, visited, pending }
}

const createTab = (selector: Selector | null = null) => {
  const saved: Selector[] = []
  const tab: PlacedTab = reactive({
    selector,
    savePlace: (place: Selector) => {
      saved.push(place)
    },
  })
  return { tab, saved }
}

describe('the place of a book tab', () => {
  it('goes to the CFI of the selector once the book is loaded', async () => {
    const { reader, visited } = createReader(true)
    const { tab } = createTab({ cfi: 'epubcfi(/8/4!/4/2/1:17)' })
    useBookPlace(reader, () => tab)

    await nextTick()
    expect(visited).toStrictEqual([])
    reader.isLoading.value = false
    await nextTick()

    expect(visited).toStrictEqual(['epubcfi(/8/4!/4/2/1:17)'])
  })

  it('goes to the CFI of a selector set later', async () => {
    const { reader, visited } = createReader()
    const { tab } = createTab()
    useBookPlace(reader, () => tab)

    ;(tab as { selector: Selector | null }).selector = { cfi: 'epubcfi(/8/2!/4/4/1:65539)' }
    await nextTick()

    expect(visited).toStrictEqual(['epubcfi(/8/2!/4/4/1:65539)'])
  })

  it('saves the CFI of the current offset', async () => {
    const { reader, pending } = createReader()
    const { tab, saved } = createTab()
    useBookPlace(reader, () => tab)

    reader.offset.value = 120
    await nextTick()
    pending.get(120)?.('epubcfi(/8/2!/4/4/1:40)')
    await settle()

    expect(saved).toStrictEqual([{ cfi: 'epubcfi(/8/2!/4/4/1:40)' }])
  })

  it('ignores the CFI of an earlier offset', async () => {
    const { reader, pending } = createReader()
    const { tab, saved } = createTab()
    useBookPlace(reader, () => tab)

    reader.offset.value = 120
    await nextTick()
    reader.offset.value = 900
    await nextTick()
    pending.get(900)?.('epubcfi(/8/4!/4/2/1:0)')
    pending.get(120)?.('epubcfi(/8/2!/4/4/1:40)')
    await settle()

    expect(saved).toStrictEqual([{ cfi: 'epubcfi(/8/4!/4/2/1:0)' }])
  })

  it('saves nothing while loading or when the backend finds no CFI', async () => {
    const { reader, pending } = createReader(true)
    const { tab, saved } = createTab()
    useBookPlace(reader, () => tab)

    reader.offset.value = 60
    await nextTick()
    reader.isLoading.value = false
    reader.offset.value = 70
    await nextTick()
    pending.get(70)?.(null)
    await settle()

    expect(pending.has(60)).toBe(false)
    expect(saved).toStrictEqual([])
  })
})
