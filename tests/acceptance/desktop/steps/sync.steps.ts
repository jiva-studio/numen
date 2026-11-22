import { Given, Then, When } from '../support/bdd.js'

Given('a sync server runs', async ({ person }) => {
  await person.sync.start()
})

Given('the sync server needs a newer protocol', async ({ person }) => {
  await person.sync.needNewerProtocol()
})

Then('the sync server has a session for this device', async ({ person }) => {
  await person.sync.expectSession()
})

Then('the sync server has a session for the account fingerprint of the identity', async ({ person }) => {
  await person.sync.expectSessionForAccount()
})

Then('the sync server has no session', async ({ person }) => {
  await person.sync.expectNoSession()
})

When('the sync server ends the session of this device', async ({ person }) => {
  await person.sync.endSession()
})

Then('the plugin {string} runs', async ({ person }, name: string) => {
  await person.sync.expectRunning(name, true)
})

Then('the plugin {string} does not run', async ({ person }, name: string) => {
  await person.sync.expectRunning(name, false)
})

When('the process of the plugin {string} ends', async ({ person }, name: string) => {
  await person.sync.endProcess(name)
})

Then('the plugin {string} warns about {string}', async ({ person }, name: string, text: string) => {
  await person.sync.expectWarning(name, text)
})
