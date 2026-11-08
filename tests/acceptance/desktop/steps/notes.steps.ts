import { Given, Then, When } from '../support/bdd.js'

Given('the note {string} is open with {string}', async ({ person }, title: string, text: string) => {
  await person.openNote(title, text)
})

Given('the person writes {string} in the open note {string}', async ({ person }, text: string, title: string) => {
  await person.writeWithoutSaving(text, title)
})

Given('another program writes {string} in the saved note {string}', async ({ person }, text: string, title: string) => {
  await person.changeNoteOnDisk(text, title)
})

Given('another program removes the saved note {string}', async ({ person }, title: string) => {
  await person.removeNoteFromDisk(title)
})

When('the person chooses {string} for the open note {string}', async ({ person }, option: string, title: string) => {
  await person.choose(option, title)
})

Then('the {word} note {string} has {string}', async ({ person }, kind: string, title: string, text: string) => {
  await person.expectHas(kind, title, text)
})

Then('the {word} note {string} does not have {string}', async ({ person }, kind: string, title: string, text: string) => {
  await person.expectLacks(kind, title, text)
})

Then('the open note {string} warns about {string}', async ({ person }, title: string, kind: string) => {
  await person.expectWarning(title, kind)
})
