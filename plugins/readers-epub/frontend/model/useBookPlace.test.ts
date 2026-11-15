import { describe, expect, it } from 'vitest'
import { nextTick, reactive, ref } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
import { useBookPlace, type PlacedTab } from './useBookPlace'

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const createReader = (isLoading = false) => {
  const offset = ref(0)
  const loading = ref(isLoading)
  const chapter = ref('')
  const visited: string[] = []
  const pending = new Map<number, (cfi: string | null) => void>()
  const reader = {
    offset,
    isLoading: loading,
    chapter,
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
  const reported: unknown[] = []
  const tab: PlacedTab = reactive({
    selector,
    savePlace: (place: Selector) => {
      saved.push(place)
    },
    reportPlace: (place: string) => {
      reported.push(JSON.parse(place))
    },
  })
  return { tab, saved, reported }
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

  it('saves the CFI alone where the reader names no chapter', async () => {
    const { reader, pending } = createReader()
    const { tab, saved } = createTab()
    useBookPlace(reader, () => tab)

    reader.offset.value = 120
    await nextTick()
    pending.get(120)?.('epubcfi(/8/2!/4/4/1:40)')
    await settle()

    expect(saved).toStrictEqual([{ cfi: 'epubcfi(/8/2!/4/4/1:40)' }])
  })

  it('saves the CFI and the chapter the reader is in at the offset', async () => {
    const { reader, pending } = createReader()
    const { tab, saved } = createTab()
    useBookPlace(reader, () => tab)

    reader.chapter.value = 'The Reed Beds'
    reader.offset.value = 120
    await nextTick()
    pending.get(120)?.('epubcfi(/8/2!/4/4/1:40)')
    await settle()

    expect(saved).toStrictEqual([{ cfi: 'epubcfi(/8/2!/4/4/1:40)', chapter: 'The Reed Beds' }])
  })

  it('saves the chapter of the later offset when the reader moves to another chapter', async () => {
    const { reader, pending } = createReader()
    const { tab, saved } = createTab()
    useBookPlace(reader, () => tab)

    reader.chapter.value = 'The Reed Beds'
    reader.offset.value = 120
    await nextTick()
    reader.chapter.value = 'Herons in Winter'
    reader.offset.value = 900
    await nextTick()
    pending.get(120)?.('epubcfi(/8/2!/4/4/1:40)')
    pending.get(900)?.('epubcfi(/8/4!/4/2/1:0)')
    await settle()

    expect(saved).toStrictEqual([{ cfi: 'epubcfi(/8/4!/4/2/1:0)', chapter: 'Herons in Winter' }])
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

describe('the place a book tab reports to the window', () => {
  it('is the CFI and the chapter the reader is in at the offset', async () => {
    const { reader, pending } = createReader()
    const { tab, reported } = createTab()
    useBookPlace(reader, () => tab)

    reader.chapter.value = 'The Reed Beds'
    reader.offset.value = 120
    await nextTick()
    pending.get(120)?.('epubcfi(/8/2!/4/4/1:40)')
    await settle()

    expect(reported).toStrictEqual([{ cfi: 'epubcfi(/8/2!/4/4/1:40)', chapter: 'The Reed Beds' }])
  })

  it('is the CFI alone where the reader names no chapter', async () => {
    const { reader, pending } = createReader()
    const { tab, reported } = createTab()
    useBookPlace(reader, () => tab)

    reader.offset.value = 120
    await nextTick()
    pending.get(120)?.('epubcfi(/8/2!/4/4/1:40)')
    await settle()

    expect(reported).toStrictEqual([{ cfi: 'epubcfi(/8/2!/4/4/1:40)' }])
  })

  it('is the place of the later offset when the CFI of an earlier one arrives last', async () => {
    const { reader, pending } = createReader()
    const { tab, reported } = createTab()
    useBookPlace(reader, () => tab)

    reader.offset.value = 120
    await nextTick()
    reader.chapter.value = 'Herons in Winter'
    reader.offset.value = 900
    await nextTick()
    pending.get(900)?.('epubcfi(/8/4!/4/2/1:0)')
    pending.get(120)?.('epubcfi(/8/2!/4/4/1:40)')
    await settle()

    expect(reported).toStrictEqual([{ cfi: 'epubcfi(/8/4!/4/2/1:0)', chapter: 'Herons in Winter' }])
  })

  it('is nothing while loading or when the backend finds no CFI', async () => {
    const { reader, pending } = createReader(true)
    const { tab, reported } = createTab()
    useBookPlace(reader, () => tab)

    reader.offset.value = 60
    await nextTick()
    reader.isLoading.value = false
    reader.offset.value = 70
    await nextTick()
    pending.get(70)?.(null)
    await settle()

    expect(reported).toStrictEqual([])
  })
})
