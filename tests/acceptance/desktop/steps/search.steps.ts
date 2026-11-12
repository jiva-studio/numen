import type { DataTable } from 'playwright-bdd'
import { Then, When } from '../support/bdd.js'

When('the person searches the vault for {string}', async ({ person }, query: string) => {
  await person.search.search(query)
})

Then('the search has the result {string} in the group {string}', async ({ person }, text: string, group: string) => {
  await person.search.expectResult(text, group)
})

Then('the search does not have the result {string} in the group {string}', async ({ person }, text: string, group: string) => {
  await person.search.expectNoResult(text, group)
})

When('the person chooses the search result {string} in the group {string}', async ({ person }, text: string, group: string) => {
  await person.search.chooseResult(text, group)
})

Then(
  'the search result {string} in the group {string} {containment} the following actions:',
  async ({ person }, text: string, group: string, rule: 'only' | 'include' | 'exclude', actions: DataTable) => {
    await person.search.expectActions(text, group, actions.raw(), rule)
  },
)

When(
  'the person opens the actions of the search result {string} in the group {string}',
  async ({ person }, text: string, group: string) => {
    await person.search.openActions(text, group)
  },
)

When('the person chooses the action {string}', async ({ person }, action: string) => {
  await person.search.chooseAction(action)
})
