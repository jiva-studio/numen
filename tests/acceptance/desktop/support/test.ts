/**
 * The `test` of the acceptance suite: the shared one of the e2e suite (which clears the open tabs and
 * trusts the server certificate), the `test` of playwright-bdd that `bddgen` requires, and the
 * person driver, which leaves the vault as it found it when the scenario ends. A scenario with a
 * world of its own ends by removing the world, its server and its window, also when it failed; a
 * worker that is killed leaves that to the helper of drivers/reaper.ts.
 */
import { mergeTests } from 'playwright/test'
import { test as bdd } from 'playwright-bdd'
import { test as shared } from '../../../e2e/desktop/tests/support/test.js'
import { Person } from '../drivers/person.js'

export const test = mergeTests(bdd, shared).extend<{ person: Person }>({
  person: async ({ page, viewport }, use, testInfo) => {
    const person = new Person(page, viewport)
    try {
      await use(person)
      const log = person.readServerLog()
      if (log !== undefined && testInfo.status !== testInfo.expectedStatus) {
        await testInfo.attach('server.log', { body: log, contentType: 'text/plain' })
      }
    } finally {
      await person.removeNotes()
    }
  },
})
