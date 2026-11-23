import type { ComputerName, Holder } from '../drivers/computer.js'
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

When('the sync server stops', async ({ person }) => {
  await person.sync.pause()
})

When('the sync server runs again', async ({ person }) => {
  await person.sync.resume()
})

Then('the sync server holds no stored text {string}', async ({ person }, text: string) => {
  await person.sync.expectNoStoredText(text)
})

When('the person connects the vault {string} as a new synced vault', async ({ person }, vault: string) => {
  await person.computers.connectNew('first', vault)
})

When('the person opens the sync settings', async ({ person }) => {
  await person.syncVaults.open()
})

Then('the sync settings list the synced vault {string}', async ({ person }, vault: string) => {
  await person.computers.expectConnected('first', vault)
})

Then('the sync settings warn about {string}', async ({ person }, text: string) => {
  await person.syncVaults.expectWarning(text)
})

When('the person renames the vault {string} to {string}', async ({ person }, vault: string, name: string) => {
  await person.syncVaults.rename(vault, name)
})

When('the person adds a copy of the vault {string} as the vault {string}', async ({ person }, vault: string, copy: string) => {
  await person.syncVaults.addCopy(vault, copy)
})

When('a second computer restores the identity from the recovery words', async ({ person }) => {
  await person.computers.restoreIdentity()
})

Then('the second computer can connect to the synced vault {string}', async ({ person }, synced: string) => {
  await person.computers.expectCanConnect('second', synced)
})

When('the second computer connects its vault {string} to the synced vault {string}', async ({ person }, vault: string, synced: string) => {
  await person.computers.connectExisting('second', vault, synced)
})

Then('the sync settings of the second computer show the vault {string} as connected', async ({ person }, vault: string) => {
  await person.computers.expectConnected('second', vault)
})

When(
  '{computer} writes the note {string} with the text {string} in {owned}',
  ({ person }, computer: ComputerName, title: string, text: string, vault: string) => {
    person.computers.writeNote(computer, title, text, vault)
  },
)

When('{computer} writes the file {string} in {owned}', ({ person }, computer: ComputerName, filePath: string, vault: string) => {
  person.computers.writeFile(computer, filePath, vault)
})

Then(
  '{holder} has the note {string} with the text {string}{inside}',
  async ({ person }, holder: Holder, title: string, text: string, vault: string | undefined) => {
    await person.computers.expectNote(holder, title, text, vault)
  },
)

Then('{computer} does not have the file {string} in {owned}', async ({ person }, computer: ComputerName, filePath: string, vault: string) => {
  await person.computers.expectNoFile(computer, filePath, vault)
})

Given('a sync server that sells subscriptions runs', async ({ person }) => {
  await person.sync.startSellingSubscriptions()
})

When('the subscription ended {int} days ago', async ({ person }, days: number) => {
  await person.sync.endSubscription(days)
})

Then('the subscription shows {string}{on}', async ({ person }, text: string, computer: ComputerName) => {
  await person.computers.expectSubscription(computer, text)
})

When('the person buys the subscription', async ({ person }) => {
  await person.identity.buy()
})

When('the person opens the payment page without paying', async ({ person }) => {
  await person.identity.openPaymentWithoutPaying()
})

Then('the page offers to buy the subscription', async ({ person }) => {
  await person.identity.expectOffer(true)
})

Then('the page does not offer to buy the subscription', async ({ person }) => {
  await person.identity.expectOffer(false)
})

Then('the page says {string}', async ({ person }, text: string) => {
  await person.identity.expectPageSays(text)
})

When('both computers check for changes only every {string}', async ({ person }, interval: string) => {
  await person.computers.setInterval(interval)
})

Given('the second computer goes offline for a moment', async ({ person }) => {
  await person.computers.dropConnection()
})

Then(
  '{holder} has the note {string} with the text {string} within {string} seconds{inside}',
  async ({ person }, holder: Holder, title: string, text: string, ...rest: [seconds: string, vault: string | undefined]) => {
    const [seconds, vault] = rest
    await person.computers.expectNoteWithin(holder, { title, text, seconds: Number(seconds) }, vault)
  },
)
