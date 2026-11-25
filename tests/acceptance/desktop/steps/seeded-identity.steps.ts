import type { Person } from '../drivers/person.js'
import { Given } from '../support/bdd.js'

/** The vault the second computer of the person keeps the synced vault in. */
const SECOND_COMPUTER_VAULT = 'Copy'

const setUpFirstComputer = async (person: Person, vault: string): Promise<void> => {
  if (!person.scenario.hasWorld) person.vault.createEmptyVault(vault)
  if (!person.sync.isRunning()) await person.sync.start()
  await person.computers.giveIdentity()
  await person.application.startApplication('editor')
  await person.computers.connectNew('first', vault)
}

Given('the person has an identity', async ({ person }) => {
  await person.computers.giveIdentity()
})

Given('the person has the synced vault {string}', async ({ person }, vault: string) => {
  await setUpFirstComputer(person, vault)
})

Given('the person has the synced vault {string} on two computers', async ({ person }, vault: string) => {
  await setUpFirstComputer(person, vault)
  await person.computers.restoreIdentity()
  await person.computers.connectExisting('second', SECOND_COMPUTER_VAULT, vault)
})
