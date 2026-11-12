import type { DataTable } from 'playwright-bdd'
import type { CommandTarget } from '../drivers/rules.js'
import { Given, Then, When } from '../support/bdd.js'

When('the application {string} starts', async ({ person }, name: string) => {
  await person.application.startApplication(name)
})

When('the application {string} restarts', async ({ person }, name: string) => {
  await person.application.restartApplication(name)
})

Given('the environment is set to the following:', async ({ person }, settings: DataTable) => {
  await person.setEnvironment(settings.raw())
})

When(
  'the person runs the command {string}{target} with:',
  async ({ person }, label: string, target: CommandTarget, answers: DataTable) => {
    await person.command.run(label, target, answers.raw())
  },
)

Then('the palette has the following rows:', async ({ person }, rows: DataTable) => {
  await person.palette.expectRows(rows.raw())
})

When('the person types {string} in the palette', async ({ person }, text: string) => {
  await person.palette.writeInField(text)
})

Then('the palette lists the command {string} {int} time(s)', async ({ person }, label: string, count: number) => {
  await person.palette.expectCommandCount(label, count)
})

When('the person presses {string}', async ({ person }, keys: string) => {
  await person.window.pressKeys(keys)
})

When('the person runs the command {string}', async ({ person }, label: string) => {
  await person.application.runCommand(label)
})

Then('{vault} contains the note {reference} {string}', async ({ person }, vault: string, reference: string, name: string) => {
  await person.application.expectVaultContainsNote(vault, reference, name)
})

