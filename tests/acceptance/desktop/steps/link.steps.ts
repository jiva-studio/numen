import type { ComputerName } from '../drivers/computer.js'
import { Given, Then, When } from '../support/bdd.js'

When('the person shows the QR code to link a new device', async ({ person }) => {
  await person.computers.showQrToLink()
})

When('the second computer shows the QR code to join an account', async ({ person }) => {
  await person.computers.showQrToJoin()
})

When('{computer} scans the QR code', async ({ person }, computer: ComputerName) => {
  await person.computers.scan(computer)
})

When('the person approves the new device', async ({ person }) => {
  await person.computers.approve()
})

When('the person declines the new device', async ({ person }) => {
  await person.computers.decline()
})

When('the second computer enters the confirmation code of the person', async ({ person }) => {
  await person.computers.enterCode()
})

When('the second computer types {string} as the confirmation code', async ({ person }, code: string) => {
  await person.computers.enterCode(code)
})

When('the second computer enters a wrong confirmation code {int} times', async ({ person }, times: number) => {
  await person.computers.enterWrongCodes(times)
})

Then('the second computer has the account of the person', async ({ person }) => {
  await person.computers.expectSecondHasAccount()
})

Then('the second computer asks to join the account of the person', async ({ person }) => {
  await person.computers.expectAsksToJoin()
})

Then('the second computer has no identity', async ({ person }) => {
  await person.computers.expectSecondHasNoIdentity()
})

When('the second computer deletes the identity from this device', async ({ person }) => {
  await person.computers.deleteSecondIdentity()
})

Given('the second computer creates an identity', async ({ person }) => {
  await person.computers.createSecondIdentity()
})

Then('the account has {int} device(s)', async ({ person }, count: number) => {
  await person.computers.expectActiveDevices(count)
})

Then('the person sees the second computer in the request', async ({ person }) => {
  await person.computers.expectPeerIsSecond()
})

Then('the second computer warns about {string}', async ({ person }, text: string) => {
  await person.computers.expectWarning('second', text)
})

Given('the second computer is linked to the account of the person', async ({ person }) => {
  await person.computers.linkSecond()
})

When('the person revokes the second computer', async ({ person }) => {
  await person.computers.revokeSecond()
})

Then('the person cannot revoke this device', async ({ person }) => {
  await person.computers.expectCannotRevokeThisDevice()
})

Then('the sync server has no session for the second computer', async ({ person }) => {
  await person.computers.expectNoSessionOfSecond()
})

Then('the second computer cannot join an account', async ({ person }) => {
  await person.computers.expectSecondCannotJoin()
})
