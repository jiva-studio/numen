import type { DataTable } from 'playwright-bdd'
import { Then, When } from '../support/bdd.js'

Then('the plex shows the {relation} note titled {string}', async ({ person }, relation: string, title: string) => {
  await person.plex.expectShowsNote(title, relation, true)
})

Then('the plex does not show the {relation} note titled {string}', async ({ person }, relation: string, title: string) => {
  await person.plex.expectShowsNote(title, relation, false)
})

When('the person hovers over the note titled {string} in the plex', async ({ person }, title: string) => {
  await person.plex.hoverOver(title)
})

Then(
  'the plex shows the heading {string} under the note titled {string}',
  async ({ person }, heading: string, title: string) => {
    await person.plex.expectShowsHeading(heading, title, true)
  },
)

Then(
  'the plex does not show the heading {string} under the note titled {string}',
  async ({ person }, heading: string, title: string) => {
    await person.plex.expectShowsHeading(heading, title, false)
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

Then('the plex shows the following notes:', async ({ person }, notes: DataTable) => {
  await person.plex.expectShowsNotes(notes.raw(), true)
})

Then('the plex does not show the following notes:', async ({ person }, notes: DataTable) => {
  await person.plex.expectShowsNotes(notes.raw(), false)
})

Then('the plex shows the following names:', async ({ person }, names: DataTable) => {
  await person.plex.expectShowsNames(names.raw())
})

Then('the plex shows no names under the note titled {string}', async ({ person }, title: string) => {
  await person.plex.expectShowsNoNames(title)
})

When('the person clicks the name {string} under the note titled {string}', async ({ person }, name: string, title: string) => {
  await person.plex.chooseName(name, title)
})

When(
  'the person links the note titled {string} to the note titled {string} as a {relation} in the plex',
  async ({ person }, title: string, toTitle: string, relation: string) => {
    await person.plex.dragLink(title, relation, { title: toTitle, isNew: false })
  },
)

When(
  'the person links the note titled {string} to a new note titled {string} as a {relation} in the plex',
  async ({ person }, title: string, toTitle: string, relation: string) => {
    await person.plex.dragLink(title, relation, { title: toTitle, isNew: true })
  },
)

When(
  'the person changes the link from {string} to {string} in the plex with:',
  async ({ person }, from: string, to: string, fields: DataTable) => {
    await person.plex.changeLink(from, to, fields.raw())
  },
)

When(
  'the person adds the reverse link from {string} to {string} in the plex with:',
  async ({ person }, from: string, to: string, fields: DataTable) => {
    await person.plex.addReverseLink(from, to, fields.raw())
  },
)

When('the person removes the link from {string} to {string} in the plex', async ({ person }, from: string, to: string) => {
  await person.plex.removeLink(from, to)
})

When('the person removes the link between {string} and {string} in the plex', async ({ person }, first: string, second: string) => {
  await person.plex.removeConnection(first, second)
})
