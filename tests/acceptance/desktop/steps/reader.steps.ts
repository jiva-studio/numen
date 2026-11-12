import type { DataTable } from 'playwright-bdd'
import type { Place, TabKindChoice } from '../drivers/rules.js'
import { Then, When } from '../support/bdd.js'

When(
  'the person goes to {place} in the {tab} {reference} {string}',
  async ({ person }, place: Place, kind: TabKindChoice, reference: string, name: string) => {
    await person.reader.goToPage(place, kind, reference, name)
  },
)

When(
  'the person clicks the button {string} in the {tab} {reference} {string}',
  async ({ person }, label: string, kind: TabKindChoice, reference: string, name: string) => {
    await person.reader.clickButton(label, kind, reference, name)
  },
)

Then(
  'the button {string} in the {tab} {reference} {string} is enabled',
  async ({ person }, label: string, kind: TabKindChoice, reference: string, name: string) => {
    await person.reader.expectButtonEnabled(label, kind, reference, name)
  },
)

Then(
  'the button {string} in the {tab} {reference} {string} is disabled',
  async ({ person }, label: string, kind: TabKindChoice, reference: string, name: string) => {
    await person.reader.expectButtonDisabled(label, kind, reference, name)
  },
)

Then(
  'the contents of the {tab} {reference} {string} are:',
  async ({ person }, kind: TabKindChoice, reference: string, name: string, chapters: DataTable) => {
    await person.reader.expectContents(chapters.raw(), kind, reference, name)
  },
)

Then(
  'the contents of the {tab} {reference} {string} are not shown',
  async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
    await person.reader.expectContentsHidden(kind, reference, name)
  },
)

When(
  'the person chooses the chapter {string} in the contents of the {tab} {reference} {string}',
  async ({ person }, chapter: string, kind: TabKindChoice, reference: string, name: string) => {
    await person.reader.chooseChapter(chapter, kind, reference, name)
  },
)

When(
  'the person moves the {tab} {reference} {string} to the other pane',
  async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
    await person.reader.moveToOtherPane(kind, reference, name)
  },
)

Then(
  'the {tab} {reference} {string} is in a pane of its own',
  async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
    await person.reader.expectOwnPane(kind, reference, name)
  },
)
