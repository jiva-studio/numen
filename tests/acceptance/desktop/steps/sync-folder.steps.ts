import type { ComputerName } from '../drivers/computer.js'
import { Given, Then, When } from '../support/bdd.js'

Then('the second computer is asked to confirm {string}', async ({ person }, text: string) => {
  await person.computers.expectAskedToConfirm('second', text)
})

Then('the second computer is not asked to confirm', async ({ person }) => {
  await person.computers.expectNotAsked('second')
})

When(
  'the second computer connects its vault {string} to the synced vault {string} and confirms',
  async ({ person }, vault: string, synced: string) => {
    await person.computers.answerConfirmation('second', vault, synced, 'Connect')
  },
)

When(
  'the second computer connects its vault {string} to the synced vault {string} and refuses',
  async ({ person }, vault: string, synced: string) => {
    await person.computers.answerConfirmation('second', vault, synced, 'Cancel')
  },
)

Then('the sync settings of the second computer do not show the vault {string} as connected', async ({ person }, vault: string) => {
  await person.computers.expectNotConnected('second', vault)
})

Then(
  '{computer} has no conflict note of {string} in {owned}',
  async ({ person }, computer: ComputerName, title: string, vault: string) => {
    await person.computers.expectNoConflictNote(computer, title, vault)
  },
)

Then(
  '{computer} has a conflict note of {string} with the text {string} in {owned}',
  async ({ person }, computer: ComputerName, title: string, text: string, vault: string) => {
    await person.computers.expectConflictNote(computer, title, text, vault)
  },
)

When('{computer} removes the note {string} in {owned}', ({ person }, computer: ComputerName, title: string, vault: string) => {
  person.computers.removeNote(computer, title, vault)
})

Then('{computer} does not have the note {string} in {owned}', async ({ person }, computer: ComputerName, title: string, vault: string) => {
  await person.computers.expectNoNote(computer, title, vault)
})

Then('the vault {string} does not have the note {string}', async ({ person }, vault: string, title: string) => {
  await person.computers.expectNoNote('first', title, vault)
})

Given('the second computer is offline', async ({ person }) => {
  await person.computers.setOnline('second', false)
})

When('the second computer is online', async ({ person }) => {
  await person.computers.setOnline('second', true)
})

Given('the sync server forgets which notes were removed', async ({ person }) => {
  await person.sync.forgetRemovals()
})
