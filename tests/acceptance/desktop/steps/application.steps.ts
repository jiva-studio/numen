import { Then, When } from '../support/bdd.js'

When('the application {string} starts', async ({ person }, name: string) => {
  await person.application.startApplication(name)
})

When('the person runs the command {string}', async ({ person }, label: string) => {
  await person.application.runCommand(label)
})

When('the person names it {string}', async ({ person }, name: string) => {
  await person.application.enterName(name)
})

When('the person picks {string}', async ({ person }, item: string) => {
  await person.application.pickItem(item)
})

Then('{vault} contains the note {reference} {string}', async ({ person }, vault: string, reference: string, name: string) => {
  await person.application.expectVaultContainsNote(vault, reference, name)
})

Then(
  'the number of the notes of the type {string} in {vault} is {int}',
  async ({ person }, type: string, vault: string, count: number) => {
    await person.application.expectNoteCount(type, vault, count)
  },
)
