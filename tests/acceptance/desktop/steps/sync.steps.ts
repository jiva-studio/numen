import type { ComputerName, Holder, Wait } from '../drivers/computer.js'
import type { PlanSpec } from '../drivers/sync.js'
import { Given, Then, When } from '../support/bdd.js'

Given('a sync server runs', async ({ person }) => {
  await person.sync.start()
})

Given('the sync server needs a newer Numen', async ({ person }) => {
  await person.sync.needNewerProtocol()
})

Then('this device is signed in to the sync server', async ({ person }) => {
  await person.sync.expectSignedIn()
})

Then('this device is not signed in to the sync server', async ({ person }) => {
  await person.sync.expectNotSignedIn()
})

When('the sync server signs this device out', async ({ person }) => {
  await person.sync.endSession()
})

Then('Numen Sync {working}', async ({ person }, isWorking: boolean) => {
  await person.sync.expectRunning('Numen Sync', isWorking)
})

When('Numen Sync stops by itself', async ({ person }) => {
  await person.sync.endProcess('Numen Sync')
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

When('the person syncs the vault {string}', async ({ person }, vault: string) => {
  await person.computers.connectNew('first', vault)
})

When('the person opens the sync settings', async ({ person }) => {
  await person.syncVaults.open()
})

Then('the sync settings list the synced vault {string}', async ({ person }, vault: string) => {
  await person.computers.expectConnected('first', vault)
})

When('the person stops syncing the vault {string}', async ({ person }, vault: string) => {
  await person.computers.stopSyncingVault('first', vault)
})

Then('the vault {string} is not synced', async ({ person }, vault: string) => {
  await person.computers.expectNotConnected('first', vault)
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

Then('the sync settings of the second computer show the vault {string} as synced', async ({ person }, vault: string) => {
  await person.computers.expectConnected('second', vault)
})

When(
  '{computer} writes the note {string} with the text {string} in {owned}',
  ({ person }, computer: ComputerName, title: string, text: string, vault: string) => {
    person.computers.writeNote(computer, title, text, vault)
  },
)

When('{computer} adds the file {string} to {owned}', ({ person }, computer: ComputerName, filePath: string, vault: string) => {
  person.files.writeFile(computer, filePath, undefined, vault)
})

Then('{computer} does not have the file {string} in {owned}', async ({ person }, computer: ComputerName, filePath: string, vault: string) => {
  await person.computers.expectNoFile(computer, filePath, vault)
})

When('{computer} is the first to reach the sync server', ({ person }, computer: ComputerName) => {
  person.computers.connectToServer(computer)
})

Then(
  '{holder} has a conflict copy of the note {string} with the text {string}{inside}',
  async ({ person }, holder: Holder, title: string, text: string, vault: string | undefined) => {
    await person.computers.expectConflictCopy(holder, title, text, vault)
  },
)

Then('{holder} has no conflict copy of the note {string}{inside}', async ({ person }, holder: Holder, title: string, vault: string | undefined) => {
  await person.computers.expectNoConflictCopy(holder, title, vault)
})

Given('a sync server that sells subscriptions runs', async ({ person }) => {
  await person.sync.startSellingSubscriptions()
})

When('the subscription ended {int} days ago', async ({ person }, days: number) => {
  await person.sync.endSubscription(days)
})

Then('the subscription is active{on}', async ({ person }, computer: ComputerName) => {
  await person.computers.expectActive(computer)
})

When('the person buys the subscription', async ({ person }) => {
  await person.subscription.startPurchase()
})

When('the person opens the payment page without paying', async ({ person }) => {
  await person.subscription.openPaymentWithoutPaying()
})

Then('the page offers to buy the subscription', async ({ person }) => {
  await person.subscription.expectOffer(true)
})

Then('the page does not offer to buy the subscription', async ({ person }) => {
  await person.subscription.expectOffer(false)
})

Given('a sync server that sells the plans {plans} runs', async ({ person }, plans: PlanSpec[]) => {
  await person.sync.startSellingPlans(plans)
})

When('the person buys the plan {string}', async ({ person }, name: string) => {
  await person.subscription.choosePlan(name)
})

Then('the page offers to buy the plan {string} with {int} GB', async ({ person }, name: string, gigabytes: number) => {
  await person.subscription.expectBuyOffer(name, gigabytes)
})

Then('the page does not offer to buy a plan', async ({ person }) => {
  await person.subscription.expectNoBuyOffer()
})

Then('the subscription shows the plan {string} with {int} GB{on}', async ({ person }, name: string, gigabytes: number, computer: ComputerName) => {
  await person.computers.expectPlan(computer, name, gigabytes)
})

When('the person changes the plan to {string}', async ({ person }, name: string) => {
  await person.subscription.changePlan(name)
})

When('the person wants to change the plan to {string}', async ({ person }, name: string) => {
  await person.subscription.chooseChange(name)
})

When('the person confirms the switch to {string}', async ({ person }, name: string) => {
  await person.subscription.confirmChange(name)
})

When('the person cancels the switch', async ({ person }) => {
  await person.subscription.cancelChange()
})

Then('the page shows the plan {string} as the current plan', async ({ person }, name: string) => {
  await person.subscription.expectCurrentPlan(name)
})

Then('the page offers to change the plan to {string}', async ({ person }, name: string) => {
  await person.subscription.expectChangeOffer(name, true)
})

Then('the page does not offer to change the plan to {string}', async ({ person }, name: string) => {
  await person.subscription.expectChangeOffer(name, false)
})

Given('the cloud holds {int} GB', async ({ person }, gigabytes: number) => {
  await person.sync.holdInCloud(gigabytes)
})

When('{int} GB are removed from the cloud', async ({ person }, gigabytes: number) => {
  await person.sync.deleteFromCloud(gigabytes)
})

When('the person starts a trial of the plan {string}', async ({ person }, name: string) => {
  await person.sync.startTrial(name)
})

When('the person cancels the trial', async ({ person }) => {
  await person.sync.cancelTrial()
})

Then('the subscription shows the day the trial ends', async ({ person }) => {
  await person.subscription.expectTrialUntil(person.sync.getTrialEnd())
})

When('the operator makes a backup of the sync server', async ({ person }) => {
  await person.sync.makeBackup()
})

When('the operator restores the sync server from the backup{telling}', async ({ person }, isTold: boolean) => {
  await person.sync.restoreFromBackup(isTold)
})

When('{computer} goes offline', ({ person }, computer: ComputerName) => {
  person.computers.setOnline(computer, false)
})

When('{computer} goes online', ({ person }, computer: ComputerName) => {
  person.computers.setOnline(computer, true)
})

Then('the synced vault {string} holds the note {string} with the text {string}', async ({ person }, synced: string, title: string, text: string) => {
  await person.computers.expectHeld(synced, title, text)
})

Then('the synced vault {string} does not hold the note {string}', async ({ person }, synced: string, title: string) => {
  await person.computers.expectHeld(synced, title, undefined)
})

Then('{holder} has the same file {string}{inside}', async ({ person }, holder: Holder, filePath: string, vault: string | undefined) => {
  await person.files.expectSameFile(holder, filePath, vault)
})

Then('the vault {string} has the same file {string} as the second computer', async ({ person }, vault: string, filePath: string) => {
  await person.files.expectSameFileAsSecond(vault, filePath)
})

When('{computer} adds the file {string} of {int} MB to {owned}', ({ person }, computer: ComputerName, filePath: string, megabytes: number, vault: string) => {
  person.files.writeFile(computer, filePath, megabytes, vault)
})

When('{computer} adds {int} files to the folder {string} in {owned}', ({ person }, computer: ComputerName, count: number, folder: string, vault: string) => {
  person.files.writeFiles(computer, count, folder, vault)
})

When('{computer} replaces the file {string} with a new version in {owned}', ({ person }, computer: ComputerName, filePath: string, vault: string) => {
  person.files.changeFile(computer, filePath, vault)
})

When('{computer} moves the file {string} to {string} in {owned}', async ({ person }, computer: ComputerName, from: string, to: string, vault: string) => {
  await person.files.moveEntry(computer, 'file', { from, to }, vault)
})

When('{computer} moves the folder {string} to {string} in {owned}', async ({ person }, computer: ComputerName, from: string, to: string, vault: string) => {
  await person.files.moveEntry(computer, 'folder', { from, to }, vault)
})

When('{computer} removes the file {string} in {owned}', async ({ person }, computer: ComputerName, filePath: string, vault: string) => {
  await person.files.removeEntry(computer, 'file', filePath, vault)
})

When('{computer} removes the folder {string} in {owned}', async ({ person }, computer: ComputerName, folder: string, vault: string) => {
  await person.files.removeEntry(computer, 'folder', folder, vault)
})

Then(
  '{holder} has {int} files in its folder {string}{inside}',
  async ({ person }, holder: Holder, count: number, folder: string, vault: string | undefined) => {
    await person.files.expectFileCount(holder, count, folder, vault)
  },
)

Then('{holder} has 1 file in its folder {string}{inside}', async ({ person }, holder: Holder, folder: string, vault: string | undefined) => {
  await person.files.expectFileCount(holder, 1, folder, vault)
})

Given('the connection breaks while the file {string} is sent', async ({ person }, filePath: string) => {
  await person.files.breakConnectionWhileSending(filePath)
})

Given('the connection breaks while the second computer receives the file {string}', async ({ person }, filePath: string) => {
  await person.files.breakConnectionWhileReceiving(filePath)
})

When('the person chooses {string}', async ({ person }, action: string) => {
  await person.files.choose(action)
})

When('the computers do not look for changes by themselves', async ({ person }) => {
  await person.computers.stopLookingForChanges()
})

Given('the second computer goes offline for a moment', async ({ person }) => {
  await person.computers.dropConnection()
})

Then(
  '{holder} has the note {string} with the text {string}{wait}',
  async ({ person }, holder: Holder, title: string, text: string, wait: Wait) => {
    await person.computers.expectNoteWithin(holder, { title, text }, wait)
  },
)

When('the person asks to delete the cloud data', async ({ person }) => {
  await person.accounts.openDeleteQuestion()
})

When('the person cancels the deletion of the cloud data', async ({ person }) => {
  await person.accounts.cancelDeleteCloudData()
})

When('the person deletes the cloud data', async ({ person }) => {
  await person.accounts.deleteCloudData()
})

Then('the cloud data is being deleted', async ({ person }) => {
  await person.accounts.expectDeleting(true)
})

Then('the cloud data is kept', async ({ person }) => {
  await person.accounts.expectDeleting(false)
})

When('the person restores the cloud data', async ({ person }) => {
  await person.accounts.restoreAccount()
})

When('the person exports the identity data', async ({ person }) => {
  await person.accounts.exportAccountData()
})

Then('the saved file names the identity of the person', ({ person }) => {
  person.accounts.expectExportNamesAccount()
})
