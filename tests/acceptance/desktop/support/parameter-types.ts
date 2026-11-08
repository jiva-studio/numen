import { defineParameterType } from 'playwright-bdd'

defineParameterType({
  name: 'actor',
  regexp: /the person|another program/,
  transformer: (actor: string) => actor,
})

defineParameterType({
  name: 'reference',
  regexp: /at|titled/,
  transformer: (reference: string) => reference,
})

defineParameterType({
  name: 'position',
  regexp: /page|time/,
  transformer: (position: string) => position,
})
