import type { DataTable } from 'playwright-bdd'
import { Given } from '../support/bdd.js'

Given('{vault} has the following card models:', ({ person }, vault: string, models: DataTable) => {
  person.vault.addCardModels(vault, models.raw())
})

Given('{vault} has the following cards:', ({ person }, vault: string, cards: DataTable) => {
  person.vault.addCards(vault, cards.raw())
})

Given('{vault} has the following presets:', ({ person }, vault: string, presets: DataTable) => {
  person.vault.addPresets(vault, presets.raw())
})
