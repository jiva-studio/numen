import { Given, Then, When } from '../support/bdd.js'

Given('the person has the note {string} open', async ({ person }, title: string) => {
  await person.openNote(title)
})

Given('the person has written {string} that is not saved yet', async ({ person }, text: string) => {
  await person.writeWithoutSaving(text)
})

Given('another program changes/changed the note to {string}', async ({ person }, text: string) => {
  await person.changeNoteOnDisk(text)
})

When('another program removes the note', async ({ person }) => {
  await person.removeNoteFromDisk()
})

Given('another program removed the note', async ({ person }) => {
  await person.removeNoteFromDisk()
})

When('the person keeps their version', async ({ person }) => {
  await person.keepOwnVersion()
})

When('the person takes the disk version', async ({ person }) => {
  await person.takeDiskVersion()
})

When('the person recreates the note', async ({ person }) => {
  await person.recreateNote()
})

Then('the note has the text {string}', async ({ person }, text: string) => {
  await person.expectNoteHas(text)
})

Then('the note keeps the text it had', async ({ person }) => {
  await person.expectNoteKeepsOwnText()
})

Then('the note holds {string} in place of the written text', async ({ person }, text: string) => {
  await person.expectNoteHoldsDiskVersion(text)
})

Then('the note reports that its file changed on disk', async ({ person }) => {
  await person.expectConflictReported()
})

Then('the note reports that it was removed from the vault', async ({ person }) => {
  await person.expectRemovalReported()
})

Then('the note reports no conflict', async ({ person }) => {
  await person.expectNoConflict()
})

Then('the file holds {string}', async ({ person }, text: string) => {
  await person.expectFileHolds(text)
})

Then('the file holds the written text in place of the disk text', async ({ person }) => {
  await person.expectFileKeepsWrittenText()
})

Then('the vault holds the note again with its text', async ({ person }) => {
  await person.expectNoteRecreated()
})
