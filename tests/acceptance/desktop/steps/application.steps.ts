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

Given('the vault {string} has the note {reference} {string}', async ({ person }, vault: string, reference: string, name: string) => {
  await person.application.expectVaultHasNote(vault, reference, name)
})

Then(
  'the number of the notes of the type {string} in the vault {string} is {string}',
  async ({ person }, type: string, vault: string, count: string) => {
    await person.application.expectNoteCount(type, vault, count)
  },
)

Given('the vault {string} has the folder at {string}', async ({ person }, vault: string, folder: string) => {
  await person.application.expectVaultHasFolder(vault, folder)
})

Then('the vault {string} does not have the file at {string}', async ({ person }, vault: string, file: string) => {
  await person.application.expectVaultDoesNotHaveFile(vault, file)
})
