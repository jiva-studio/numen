import { defineParameterType } from 'playwright-bdd'
import { parsePlace, parseTabKind, parseVaultName } from '../drivers/rules.js'

defineParameterType({
  name: 'actor',
  regexp: /the person|another program/,
  transformer: (actor: string) => actor,
})

defineParameterType({
  name: 'reference',
  regexp: /at|titled|for the web address/,
  transformer: (reference: string) => reference,
})

defineParameterType({
  name: 'place',
  regexp: /(?:page|time) "[^"]+"/,
  transformer: (place: string) => parsePlace(place),
})

defineParameterType({
  name: 'scope',
  regexp: /vault|deck/,
  transformer: (scope: string) => scope,
})

defineParameterType({
  name: 'entry',
  regexp: /file|folder/,
  transformer: (entry: string) => entry,
})

defineParameterType({
  name: 'vault',
  regexp: /the vault(?: "[^"]+")?/,
  transformer: (vault: string) => parseVaultName(vault),
})

defineParameterType({
  name: 'tab',
  regexp: /(?:[A-Za-z]+ ){0,2}tab/,
  transformer: (tab: string) => parseTabKind(tab),
})

defineParameterType({
  name: 'text',
  regexp: /recognized text|transcript/,
  transformer: (text: string) => text,
})
