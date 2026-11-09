import { Then, When } from '../support/bdd.js'

Then(
  'the plex has the note {reference} {string} as the {string}',
  async ({ person }, reference: string, name: string, relation: string) => {
    await person.plex.expectHasNote(reference, name, relation)
  },
)

When(
  'the person links the note {reference} {string} to the note {reference} {string} in the plex',
  async ({ person }, reference: string, name: string, toReference: string, toName: string) => {
    await person.plex.link(reference, name, toReference, toName)
  },
)

When('the person clicks the note {reference} {string} in the plex', async ({ person }, reference: string, name: string) => {
  await person.plex.choose(reference, name)
})
