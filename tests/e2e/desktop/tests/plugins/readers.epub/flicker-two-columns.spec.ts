import { test } from '../../support/test.js'
import { expectNoPageBeforeATurn } from './recorded-turns.js'

test.describe('EPUB page turns, recorded', () => {
  test('the reader never shows the page before a turn again in two columns', async ({ page }) => {
    test.setTimeout(180_000)
    await expectNoPageBeforeATurn(page, 2400)
  })
})
