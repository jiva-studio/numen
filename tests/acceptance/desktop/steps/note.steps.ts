import { Given, Then, When } from '../support/bdd.js'

Given('the person has the note titled {string} with {string}', async ({ person }, title: string, text: string) => {
  await person.note.create(title, text)
})

Given(
  'the person has unsaved {string} in the note {word} {string}',
  async ({ person }, text: string, how: string, value: string) => {
    await person.note.writeWithoutSaving(how, value, text)
  },
)

When(
  'the person writes {string} in the note {word} {string}',
  async ({ person }, text: string, how: string, value: string) => {
    await person.note.write(how, value, text)
  },
)

Given(
  'another program writes {string} in the note {word} {string} on disk',
  async ({ person }, text: string, how: string, value: string) => {
    await person.note.changeOnDisk(how, value, text)
  },
)

Given('another program removes the note {word} {string} from disk', async ({ person }, how: string, value: string) => {
  await person.note.removeFromDisk(how, value)
})

When(
  'the person chooses {string} for the note {word} {string}',
  async ({ person }, label: string, how: string, value: string) => {
    await person.note.choose(label, how, value)
  },
)

When(
  'the person chooses the completion {string} in the note {word} {string}',
  async ({ person }, label: string, how: string, value: string) => {
    await person.note.chooseCompletion(label, how, value)
  },
)

When(
  'the person opens the link {string} in the note {word} {string}',
  async ({ person }, link: string, how: string, value: string) => {
    await person.note.openLink(link, how, value)
  },
)

Then('the note {word} {string} has {string}', async ({ person }, how: string, value: string, text: string) => {
  await person.note.expectHas(how, value, text)
})

Then('the note {word} {string} does not have {string}', async ({ person }, how: string, value: string, text: string) => {
  await person.note.expectLacks(how, value, text)
})

Then('the note {word} {string} on disk has {string}', async ({ person }, how: string, value: string, text: string) => {
  await person.note.expectOnDiskHas(how, value, text)
})

Then(
  'the note {word} {string} links to the note {word} {string}',
  async ({ person }, how: string, value: string, toHow: string, toValue: string) => {
    await person.note.expectLinks(how, value, toHow, toValue)
  },
)
