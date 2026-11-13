import { Given, Then, When } from '../support/bdd.js'

When('the person opens the note titled {string}', async ({ person }, title: string) => {
  await person.note.open(title)
})

When(
  '{actor} writes {string} in the note {reference} {string}',
  async ({ person }, actor: string, text: string, reference: string, name: string) => {
    await person.note.write(actor, text, reference, name)
  },
)

When(
  'the person replaces {string} with {string} in the note {reference} {string}',
  async ({ person }, from: string, to: string, reference: string, name: string) => {
    await person.note.replace(from, to, reference, name)
  },
)

Given('another program removes the note {reference} {string} from disk', async ({ person }, reference: string, name: string) => {
  await person.note.removeFromDisk(reference, name)
})

When(
  'the person chooses {string} for the note {reference} {string}',
  async ({ person }, label: string, reference: string, name: string) => {
    await person.note.choose(label, reference, name)
  },
)

When(
  'the person chooses the completion {string} in the note {reference} {string}',
  async ({ person }, label: string, reference: string, name: string) => {
    await person.note.chooseCompletion(label, reference, name)
  },
)

When(
  'the person opens the link {string} in the note {reference} {string}',
  async ({ person }, link: string, reference: string, name: string) => {
    await person.note.openLink(link, reference, name)
  },
)

Then('the note {reference} {string} has {string}', async ({ person }, reference: string, name: string, text: string) => {
  await person.note.expectHasText(reference, name, text)
})

Then('the note {reference} {string} does not have {string}', async ({ person }, reference: string, name: string, text: string) => {
  await person.note.expectDoesNotHaveText(reference, name, text)
})

Then('the note {reference} {string} on disk has {string}', async ({ person }, reference: string, name: string, text: string) => {
  await person.note.expectOnDiskHasText(reference, name, text)
})

Then(
  'the note {reference} {string} links to the note {reference} {string}',
  async ({ person }, reference: string, name: string, toReference: string, toName: string) => {
    await person.note.expectLinks(reference, name, toReference, toName)
  },
)

Then(
  'the note titled {string} does not link to the note titled {string}',
  async ({ person }, title: string, toTitle: string) => {
    await person.note.expectDoesNotLink('titled', title, 'titled', toTitle)
  },
)
