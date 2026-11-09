import type { DataTable } from 'playwright-bdd'
import { Given } from '../support/bdd.js'

Given(
  '{vault} has the card model {string} with the fields:',
  ({ person }, vault: string, title: string, fields: DataTable) => {
    person.vault.addCardModel(vault, title, fields.raw())
  },
)

Given(
  '{vault} has the card model {string} with the templates:',
  ({ person }, vault: string, title: string, templates: DataTable) => {
    person.vault.addTemplates(vault, title, templates.raw())
  },
)

Given('{vault} has the deck {string}', ({ person }, vault: string, title: string) => {
  person.vault.addDeck(vault, title)
})

Given(
  '{vault} has the card of the card model {string} in the deck {string} with:',
  ({ person }, vault: string, model: string, deck: string, values: DataTable) => {
    person.vault.addCard(vault, model, deck, values.raw())
  },
)

Given(
  '{vault} has the preset {string} with the settings:',
  ({ person }, vault: string, title: string, settings: DataTable) => {
    person.vault.addPreset(vault, title, settings.raw())
  },
)
