import { test } from '../../support/test.js'
import { expectNoPageBeforeARapidTurnAcrossAChapter } from './recorded-turns.js'

test.describe('EPUB page turns across a chapter, recorded', () => {
  for (const delay of [0, 40, 80, 160]) {
    test(`the reader never shows the chapter before again when the next one arrives after ${delay} ms`, async ({
      page,
    }) => {
      test.setTimeout(120_000)
      await expectNoPageBeforeARapidTurnAcrossAChapter(page, delay)
    })
  }
})
