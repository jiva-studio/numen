import { Given, Then, When } from '../support/bdd.js'

Given('the person has the note {string} with {string}', async ({ person }, title: string, text: string) => {
  await person.openNote(title, text)
})

Given('the person has unsaved {string} in the note {string}', async ({ person }, text: string, title: string) => {
  await person.writeWithoutSaving(text, title)
})

Given('another program writes {string} in the note {string} on disk', async ({ person }, text: string, title: string) => {
  await person.changeNoteOnDisk(text, title)
})

Given('another program removes the note {string} from disk', async ({ person }, title: string) => {
  await person.removeNoteFromDisk(title)
})

When('the person chooses {string} for the note {string}', async ({ person }, label: string, title: string) => {
  await person.choose(label, title)
})

Then('the note {string} has {string}', async ({ person }, title: string, text: string) => {
  await person.expectHas('app', title, text)
})

Then('the note {string} does not have {string}', async ({ person }, title: string, text: string) => {
  await person.expectLacks('app', title, text)
})

Then('the note {string} on disk has {string}', async ({ person }, title: string, text: string) => {
  await person.expectHas('disk', title, text)
})

Then('the note {string} on disk does not have {string}', async ({ person }, title: string, text: string) => {
  await person.expectLacks('disk', title, text)
})
