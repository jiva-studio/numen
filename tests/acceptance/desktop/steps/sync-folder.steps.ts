import type { ComputerName, Holder } from '../drivers/computer.js'
import { Given, Then, When } from '../support/bdd.js'

Then('the second computer is asked to confirm {string}', async ({ person }, text: string) => {
  await person.computers.expectAskedToConfirm('second', text)
})

Then('the connection of the second computer needs no confirmation', async ({ person }) => {
  await person.computers.expectNotAsked('second')
})

When('the second computer confirms the connection', async ({ person }) => {
  await person.computers.answerConfirmation('second', 'Connect')
})

When('the second computer cancels the connection', async ({ person }) => {
  await person.computers.answerConfirmation('second', 'Cancel')
})

Then('the sync settings of the second computer list the vault {string} as unconnected', async ({ person }, vault: string) => {
  await person.computers.expectNotConnected('second', vault)
})

When('{computer} removes the note {string} in {owned}', ({ person }, computer: ComputerName, title: string, vault: string) => {
  person.computers.removeNote(computer, title, vault)
})

Then('{holder} is without the note {string}{inside}', async ({ person }, holder: Holder, title: string, vault: string | undefined) => {
  await person.computers.expectNoNote(holder, title, vault)
})

Given('the second computer is offline', ({ person }) => {
  person.computers.setOnline('second', false)
})

When('the second computer is online', ({ person }) => {
  person.computers.setOnline('second', true)
})

Given('the sync server forgets which notes were removed', async ({ person }) => {
  await person.sync.forgetRemovals()
})
