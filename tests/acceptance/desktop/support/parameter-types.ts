import { defineParameterType } from 'playwright-bdd'

defineParameterType({
  name: 'actor',
  regexp: /the person|another program/,
  transformer: (actor: string) => actor,
})
