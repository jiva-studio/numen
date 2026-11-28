import { Given, Then, When } from '../support/bdd.js'

When('the person shows the QR code to link a new device', async ({ person }) => {
  await person.computers.showQrToLink()
})

When('the second computer scans the QR code', async ({ person }) => {
  await person.computers.scan()
})

When('the person approves the new device', async ({ person }) => {
  await person.computers.approve()
})

When('the person declines the new device', async ({ person }) => {
  await person.computers.decline()
})

Then('the second computer has the identity of the person', async ({ person }) => {
  await person.computers.expectSecondHasAccount()
})

Then('the second computer asks to join the identity of the person', async ({ person }) => {
  await person.computers.expectAsksToJoin()
})

Then('the second computer holds no identity', async ({ person }) => {
  await person.computers.expectSecondHasNoIdentity()
})

When('the second computer deletes the identity from this device', async ({ person }) => {
  await person.computers.deleteSecondIdentity()
})

Given('the second computer creates an identity', async ({ person }) => {
  await person.computers.createSecondIdentity()
})

Then('the identity has {int} device(s)', async ({ person }, count: number) => {
  await person.computers.expectActiveDevices(count)
})

Given('the second computer is linked to the identity of the person', async ({ person }) => {
  await person.computers.linkSecond()
})

When('the person revokes the second computer', async ({ person }) => {
  await person.computers.revokeSecond()
})

Then('the person cannot revoke this device', async ({ person }) => {
  await person.computers.expectCannotRevokeThisDevice()
})

Then('the sync server no longer lets the second computer in', async ({ person }) => {
  await person.computers.expectNoSessionOfSecond()
})

Then('the second computer cannot join another identity', async ({ person }) => {
  await person.computers.expectSecondCannotJoin()
})
