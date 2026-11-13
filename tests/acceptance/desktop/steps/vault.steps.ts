import type { DataTable } from 'playwright-bdd'
import { Given, Then, When } from '../support/bdd.js'

Given('the empty vault {string}', ({ person }, name: string) => {
  person.vault.createEmptyVault(name)
})

Given('{vault} has the following:', async ({ person }, vault: string, files: DataTable) => {
  await person.vault.addFiles(vault, files.raw())
})

Then('{vault} contains the following:', async ({ person }, vault: string, files: DataTable) => {
  await person.vault.expectContainsFiles(vault, files.raw())
})

Given('{vault} contains the {entry} at {string}', async ({ person }, vault: string, entry: string, entryPath: string) => {
  await person.vault.expectContains(vault, entry, entryPath)
})

Then('{vault} does not contain the file at {string}', async ({ person }, vault: string, file: string) => {
  await person.vault.expectDoesNotContain(vault, file)
})

When('the person switches to the vault {string}', async ({ person }, name: string) => {
  await person.vault.switchTo(name)
})

Given('the list of vaults has {vault}', async ({ person }, vault: string) => {
  await person.vault.addToList(vault)
})

Then('the window has the vault {string}', async ({ person }, name: string) => {
  await person.vault.expectWindowShows(name)
})

Given('the vault list has the following:', async ({ person }, vaults: DataTable) => {
  await person.vault.expectList(vaults.raw())
})

Then('{vault} has the following links:', async ({ person }, vault: string, links: DataTable) => {
  await person.vault.expectLinks(vault, links.raw())
})

Given('the notes have the following links:', ({ person }, links: DataTable) => {
  person.vault.addLinks(links.raw())
})
