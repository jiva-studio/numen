import { defineParameterType } from 'playwright-bdd'

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
  name: 'position',
  regexp: /page|time/,
  transformer: (position: string) => position,
})

defineParameterType({
  name: 'entry',
  regexp: /file|folder/,
  transformer: (entry: string) => entry,
})
