import { test } from '../../support/test.js'
import { deleteVaultFiles } from '../../support/media.js'
import { createUniqueTitle } from '../../support/vault.js'

test.describe('a deck made through the palette', () => {
  let title: string

  test.beforeEach(() => {
    title = createUniqueTitle('Puffin colony')
  })

  test.afterEach(() => {
    deleteVaultFiles(`${title}.md`)
  })
})
