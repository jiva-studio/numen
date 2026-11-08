import { Given, Then, When } from '../support/bdd.js'

Given('a note {string} is open', async ({ person }, title: string) => {
  await person.openNote(title)
})

Given('the person writes {string}', async ({ person }, text: string) => {
  await person.writeWithoutSaving(text)
})

Given('another program writes {string} in the note', async ({ person }, text: string) => {
  await person.changeNoteOnDisk(text)
})

Given('another program removes/removed the note', async ({ person }) => {
  await person.removeNoteFromDisk()
})

When('the person keeps their text', async ({ person }) => {
  await person.keepOwnVersion()
})

When('the person takes the text from the file', async ({ person }) => {
  await person.takeDiskVersion()
})

When('the person saves the note again', async ({ person }) => {
  await person.recreateNote()
})

Then('the note has {string}', async ({ person }, text: string) => {
  await person.expectNoteHas(text)
})

Then('the note does not have {string}', async ({ person }, text: string) => {
  await person.expectNoteLacks(text)
})

Then('the note keeps {string}', async ({ person }, text: string) => {
  await person.expectNoteHas(text)
})

Then('the note keeps its text', async ({ person }) => {
  await person.expectNoteKeepsOwnText()
})

Then('the note warns that the file changed', async ({ person }) => {
  await person.expectConflictReported()
})

Then('the note warns that the file is gone', async ({ person }) => {
  await person.expectRemovalReported()
})

Then('the file has {string}', async ({ person }, text: string) => {
  await person.expectFileHas(text)
})

Then('the file does not have {string}', async ({ person }, text: string) => {
  await person.expectFileLacks(text)
})

Then('the file is back with the text', async ({ person }) => {
  await person.expectNoteRecreated()
})
