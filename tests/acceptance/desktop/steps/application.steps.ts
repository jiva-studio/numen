import { Given, Then, When } from '../support/bdd.js'

When('the application {string} starts', async ({ person }, name: string) => {
  await person.application.startApplication(name)
})

When('the person runs the command {string}', async ({ person }, label: string) => {
  await person.application.runCommand(label)
})

When('the person enters {string}', async ({ person }, value: string) => {
  await person.application.enter(value)
})

Given('the vault {string} has the note {word} {string}', async ({ person }, vault: string, how: string, value: string) => {
  await person.application.expectVaultHasNote(vault, how, value)
})

Then(
  'the number of the notes of the type {string} in the vault {string} is {string}',
  async ({ person }, type: string, vault: string, count: string) => {
    await person.application.expectNoteCount(type, vault, count)
  },
)
