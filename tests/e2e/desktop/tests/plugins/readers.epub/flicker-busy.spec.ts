import { test } from '../../support/test.js'
import { expectNoPageBeforeARapidTurnWhenBusy } from './recorded-turns.js'

test.describe('EPUB page turns, recorded', () => {
  test('the reader never shows the page before a turn again when frames are slow', async ({
    page,
  }) => {
    test.setTimeout(120_000)
    await expectNoPageBeforeARapidTurnWhenBusy(page)
  })
})
