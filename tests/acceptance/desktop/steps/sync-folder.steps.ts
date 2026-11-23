import type { ComputerName, Holder } from '../drivers/computer.js'
import { Then, When } from '../support/bdd.js'

Then('the second computer is asked to confirm {string}', async ({ person }, text: string) => {
  await person.computers.expectAskedToConfirm('second', text)
})

Then('the second computer is not asked to confirm', async ({ person }) => {
  await person.computers.expectNotAsked('second')
})

When('the second computer confirms the connection', async ({ person }) => {
  await person.computers.answerConfirmation('second', 'Connect')
})

When('the second computer cancels the connection', async ({ person }) => {
  await person.computers.answerConfirmation('second', 'Cancel')
})

Then('the sync settings of the second computer do not show the vault {string} as connected', async ({ person }, vault: string) => {
  await person.computers.expectNotConnected('second', vault)
})

When('{computer} removes the note {string} in {owned}', ({ person }, computer: ComputerName, title: string, vault: string) => {
  person.computers.removeNote(computer, title, vault)
})

Then('{holder} does not have the note {string}{inside}', async ({ person }, holder: Holder, title: string, vault: string | undefined) => {
  await person.computers.expectNoNote(holder, title, vault)
})
