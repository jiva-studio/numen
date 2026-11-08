import { Given, Then, When } from '../support/bdd.js'

Given('the note {string} is open with {string}', async ({ person }, title: string, text: string) => {
  await person.openNote(title, text)
})

Given('the person writes {string}', async ({ person }, text: string) => {
  await person.writeWithoutSaving(text)
})

Given('another program writes {string} in the file', async ({ person }, text: string) => {
  await person.changeNoteOnDisk(text)
})

Given('another program removes the file', async ({ person }) => {
  await person.removeNoteFromDisk()
})

When('the person chooses {string}', async ({ person }, option: string) => {
  await person.choose(option)
})

Then('the {word} has {string}', async ({ person }, subject: string, text: string) => {
  await person.expectHas(subject, text)
})

Then('the {word} does not have {string}', async ({ person }, subject: string, text: string) => {
  await person.expectLacks(subject, text)
})

Then('the note warns about {string}', async ({ person }, kind: string) => {
  await person.expectWarning(kind)
})
