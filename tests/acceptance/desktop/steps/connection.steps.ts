import { Given, Then, When } from '../support/bdd.js'

Given('the service {string} answers with the status {string}', async ({ person }, name: string, status: string) => {
  await person.services.setStatus(name, Number(status))
})

Given('the service {string} needs the key {string}', async ({ person }, name: string, key: string) => {
  await person.services.setKey(name, key)
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

When('the person removes the key of the connection {string}', async ({ person }, name: string) => {
  await person.connections.removeKey(name)
})

When('the person deletes the connection {string}', async ({ person }, name: string) => {
  await person.connections.delete(name)
})

Then('the connection {string} {possession} a key', async ({ person }, name: string, has: boolean) => {
  await person.connections.expectKey(name, has)
})

Then('the connections list {possession} the connection {string}', async ({ person }, has: boolean, name: string) => {
  await person.connections.expectListed(name, has)
})

Then('the connection {string} shows {string}', async ({ person }, name: string, state: string) => {
  await person.connections.expectState(name, state)
})

When(
  'the person adds a row of the connection {string} with the model {string} to the plugin {string}',
  async ({ person }, connection: string, model: string, plugin: string) => {
    await person.connections.addRow(connection, model, plugin)
  },
)

Then('the plugin {string} has {int} row(s)', async ({ person }, plugin: string, count: number) => {
  await person.connections.expectRowCount(plugin, count)
})

When('the person answers the question with {string}', async ({ person }, button: string) => {
  await person.connections.answerQuestion(button)
})

Then('the question reads {string}', async ({ person }, text: string) => {
  await person.connections.expectQuestion(text)
})
