import type { DataTable } from 'playwright-bdd'
import type { Extent } from '../drivers/rules.js'
import { Then, When } from '../support/bdd.js'

Then('the plex {display} the {relation} note titled {string}', async ({ person }, shows: boolean, relation: string, title: string) => {
  await person.plex.expectShowsNote(title, relation, shows)
})

Then('the plex {display} the following notes:', async ({ person }, shows: boolean, notes: DataTable) => {
  await person.plex.expectShowsNotes(notes.raw(), shows)
})

Then('the plex puts the following notes on the side of the focus note:', async ({ person }, notes: DataTable) => {
  await person.plex.expectSides(notes.raw())
})

When('the person rests the pointer on the note titled {string} in the plex', async ({ person }, title: string) => {
  await person.plex.restPointerOn(title)
})

Then(
  'the plex {display} the heading {string} under the note titled {string}',
  async ({ person }, shows: boolean, heading: string, title: string) => {
    await person.plex.expectShowsHeading(heading, title, shows)
  },
)

When('the person enters the heading {string} under the note titled {string} in the plex', async ({ person }, heading: string, title: string) => {
  await person.plex.enterHeading(heading, title)
})

When('the person clicks the note {reference} {string} in the plex', async ({ person }, reference: string, name: string) => {
  await person.plex.choose(reference, name)
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

When('the person clicks the handle of the note titled {string} to link the note titled {string}', async ({ person }, title: string, toTitle: string) => {
  await person.plex.clickHandle(title, toTitle)
})

When(
  'the person drags a line {direction} from the note titled {string} to {destination} titled {string}',
  async ({ person }, direction: string, title: string, isNew: boolean, toTitle: string) => {
    await person.plex.dragLine(direction, title, { title: toTitle, isNew })
  },
)

When('the person drops the handle of the note titled {string} on the note titled {string}', async ({ person }, title: string, toTitle: string) => {
  await person.plex.dragOnto(title, toTitle)
})

Then(
  'the plex draws the line from {string} to {string} with an arrow to {string}',
  async ({ person }, from: string, to: string, arrow: string) => {
    await person.plex.expectDrawsLine(from, to, arrow)
  },
)

Then('the plex draws the line from {string} to {string} without an arrow', async ({ person }, from: string, to: string) => {
  await person.plex.expectDrawsLineWithoutArrow(from, to)
})

When(
  'the person changes the link from {string} to {string} in the plex with:',
  async ({ person }, from: string, to: string, fields: DataTable) => {
    await person.plex.changeLink(from, to, fields.raw())
  },
)

When(
  'the person cancels the change of the link from {string} to {string} in the plex with:',
  async ({ person }, from: string, to: string, fields: DataTable) => {
    await person.plex.cancelLinkChange(from, to, fields.raw())
  },
)

When('the person adds the reverse link from {string} to {string} in the plex', async ({ person }, from: string, to: string) => {
  await person.plex.addReverseLink(from, to)
})

When('the person removes the link {extent} in the plex', async ({ person }, extent: Extent) => {
  await person.plex.removeLink(extent)
})

When('the person opens the menu of the note titled {string} in the plex', async ({ person }, title: string) => {
  await person.plex.openMenu(title)
})

Then('the menu {containment} the following items:', async ({ person }, rule: 'only' | 'include' | 'exclude', items: DataTable) => {
  await person.plex.expectMenuItems(items.raw(), rule)
})
