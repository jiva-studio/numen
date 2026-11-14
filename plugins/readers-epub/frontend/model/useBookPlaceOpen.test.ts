import { describe, expect, it } from 'vitest'
import { reactive, ref } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
import { createCfiSelector } from '../lib/place'
import { useBookPlace, type PlacedTab } from './useBookPlace'

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const createReader = (chapterThere: string, canReach = true) => {
  const chapter = ref('')
  const reader = {
    offset: ref(0),
    isLoading: ref(false),
    chapter,
    getCfi: async () => null,
    goToCfi: async () => {
      chapter.value = chapterThere
      return canReach
    },
  }
  return reader
}

const createTab = (selector: Selector) => {
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

describe('the place a book tab reports when it opens', () => {
  it('is the CFI of the selector it opens at and the chapter there, and nothing is saved', async () => {
    const { tab, saved, reported } = createTab(createCfiSelector('epubcfi(/6/8!/4/2/1:0)'))
    useBookPlace(createReader('The Reed Beds'), () => tab)
    await settle()

    expect(reported).toStrictEqual([{ cfi: 'epubcfi(/6/8!/4/2/1:0)', chapter: 'The Reed Beds' }])
    expect(saved).toStrictEqual([])
  })

  it('is nothing when the reader cannot scroll to the CFI', async () => {
    const { tab, reported } = createTab(createCfiSelector('epubcfi(/6/8!/4/2/1:0)'))
    useBookPlace(createReader('The Reed Beds', false), () => tab)
    await settle()

    expect(reported).toStrictEqual([])
  })
})
