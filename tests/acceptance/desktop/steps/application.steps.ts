import { Given, When } from '../support/bdd.js'

When('the window starts', async ({ person }) => {
  await person.application.startWindow()
})

When('the person runs the command {string}', async ({ person }, label: string) => {
  await person.application.runCommand(label)
})

When('the person enters the {word} {string}', async ({ person }, kind: string, value: string) => {
  await person.application.enter(kind, value)
})

Given('the vault {string} has the note {word} {string}', async ({ person }, vault: string, how: string, value: string) => {
  await person.application.expectVaultHasNote(vault, how, value)
})
