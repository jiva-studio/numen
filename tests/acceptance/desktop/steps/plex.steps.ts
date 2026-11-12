import { Then, When } from '../support/bdd.js'

Then('the plex has the {relation} note titled {string}', async ({ person }, relation: string, title: string) => {
  await person.plex.expectHasNote(title, relation, true)
})

Then('the plex does not have the {relation} note titled {string}', async ({ person }, relation: string, title: string) => {
  await person.plex.expectHasNote(title, relation, false)
})

When('the person hovers over the note titled {string} in the plex', async ({ person }, title: string) => {
  await person.plex.hoverOver(title)
})

Then(
  'the plex has the heading {string} under the note titled {string}',
  async ({ person }, heading: string, title: string) => {
    await person.plex.expectHeading(heading, title, true)
  },
)

Then(
  'the plex does not have the heading {string} under the note titled {string}',
  async ({ person }, heading: string, title: string) => {
    await person.plex.expectHeading(heading, title, false)
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
