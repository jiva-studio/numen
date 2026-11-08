import { Given, When } from '../support/bdd.js'

When('the window starts', async ({ person }) => {
  await person.application.startWindow()
})

When(
  'the person runs the command {string} with the {word} {string}',
  async ({ person }, command: string, kind: string, value: string) => {
    await person.application.runCommand(command, kind, value)
  },
)

Given('the vault {string} has the note {word} {string}', async ({ person }, vault: string, how: string, value: string) => {
  await person.application.expectVaultHasNote(vault, how, value)
})
