import { Given, Then } from '../support/bdd.js'

Given('the empty vault {string}', ({ person }, name: string) => {
  person.vault.createEmpty(name)
})

Given('{vault} has the note titled {string} with {string}', ({ person }, vault: string, title: string, text: string) => {
  person.vault.writeNote(vault, title, text)
})

Given('{vault} has the folder at {string}', ({ person }, vault: string, folder: string) => {
  person.vault.makeFolder(vault, folder)
})

Given('{vault} has the PDF at {string}', ({ person }, vault: string, filePath: string) => {
  person.vault.copyPdf(vault, filePath)
})

Given('{vault} contains the {entry} at {string}', async ({ person }, vault: string, entry: string, entryPath: string) => {
  await person.vault.expectContains(vault, entry, entryPath)
})

Then('{vault} does not contain the file at {string}', async ({ person }, vault: string, file: string) => {
  await person.vault.expectDoesNotContain(vault, file)
})
