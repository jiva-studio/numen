/**
 * The `test` of the acceptance suite: the shared one of the e2e suite (which clears the open tabs and
 * trusts the server certificate), the `test` of playwright-bdd that `bddgen` requires, and the
 * person driver, which leaves the vault as it found it when the scenario ends.
 */
import { mergeTests } from 'playwright/test'
import { test as bdd } from 'playwright-bdd'
import { test as shared } from '../../../e2e/desktop/tests/support/test.js'
import { Person } from '../drivers/person.js'

export const test = mergeTests(bdd, shared).extend<{ person: Person }>({
  person: async ({ page }, use) => {
    const person = new Person(page)
    await use(person)
    await person.removeNotes()
  },
})
