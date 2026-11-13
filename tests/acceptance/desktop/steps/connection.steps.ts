import type { DataTable } from 'playwright-bdd'
import { readConnectionEntries } from '../drivers/connections.js'
import { Given, Then, When } from '../support/bdd.js'

Given('the service {string} is busy', async ({ person }, name: string) => {
  await person.services.setBusy(name)
})

Given('the service {string} accepts only the key {string}', async ({ person }, name: string, key: string) => {
  await person.services.setKey(name, key)
})

Given(
  'the connection {string} of the type {string} for the service {string}',
  async ({ person }, name: string, type: string, service: string) => {
    await person.connections.addAll([{ name, type, service }])
  },
)

Given('the program {string} is installed on this computer', ({ person }, name: string) => {
  person.connections.installProgram(name)
})

Given('the following connections:', async ({ person }, connections: DataTable) => {
  await person.connections.addAll(readConnectionEntries(connections.raw()))
})

When(
  'the person adds the connection {string} of the type {string} for the service {string}',
  async ({ person }, name: string, type: string, service: string) => {
    await person.connections.add(name, type, service)
  },
)

When('the person gives the connection {string} the key {string}', async ({ person }, name: string, key: string) => {
  await person.connections.giveKey(name, key)
})

When('the person opens the connection {string}', async ({ person }, name: string) => {
  await person.connections.open(name)
})

When('the person deletes the connection {string}', async ({ person }, name: string) => {
  await person.connections.delete(name)
})

Then('the key field shows that a key is saved', async ({ person }) => {
  await person.connections.expectKeySaved()
})

Then('the connections list has the following connections:', async ({ person }, connections: DataTable) => {
  await person.connections.expectListed(connections.raw())
})

Then('the connection {string} shows {string}', async ({ person }, name: string, state: string) => {
  await person.connections.expectState(name, state)
})

Then('the connection {string} cannot be opened', async ({ person }, name: string) => {
  await person.connections.expectNotOpenable(name)
})
