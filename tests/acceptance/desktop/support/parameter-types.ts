import { defineParameterType } from 'playwright-bdd'
import { parseArrow, parseExtent, parsePlace, parseTabKind, parseTabRef, parseTarget, parseVaultName } from '../drivers/rules.js'

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
  regexp: /recognized text|transcript|proofread text/,
  transformer: (text: string) => text,
})

defineParameterType({
  name: 'relation',
  regexp: /focus|parent|child|jump|reference|sibling/,
  transformer: (relation: string) => relation,
})

defineParameterType({
  name: 'target',
  regexp: / on the (?:file|folder) at "[^"]+"| on the (?:[A-Za-z]+ ){0,2}tab (?:at|titled) "[^"]+"|/,
  transformer: (target: string) => parseTarget(target),
})

defineParameterType({
  name: 'bar',
  regexp: /title bar/,
  transformer: (bar: string) => bar,
})

defineParameterType({
  name: 'possession',
  regexp: /has|does not have/,
  transformer: (possession: string) => possession === 'has',
})

defineParameterType({
  name: 'containment',
  regexp: /has only|contains|does not contain/,
  transformer: (containment: string) => (containment === 'has only' ? 'only' : containment === 'contains' ? 'include' : 'exclude'),
})

defineParameterType({
  name: 'tabref',
  regexp: /(?:[A-Za-z]+ ){0,2}tab (?:at|titled|for the web address) "[^"]+"/,
  transformer: (tab: string) => parseTabRef(tab),
})

defineParameterType({
  name: 'display',
  regexp: /shows|does not show/,
  transformer: (display: string) => display === 'shows',
})

defineParameterType({
  name: 'destination',
  regexp: /the note|a new note/,
  transformer: (destination: string) => destination === 'a new note',
})

defineParameterType({
  name: 'extent',
  regexp: /from "[^"]+" to "[^"]+"|between "[^"]+" and "[^"]+"/,
  transformer: (extent: string) => parseExtent(extent),
})

defineParameterType({
  name: 'direction',
  regexp: /up|down|left/,
  transformer: (direction: string) => direction,
})

defineParameterType({
  name: 'arrow',
  regexp: / with an arrow to "[^"]+"| without an arrow|/,
  transformer: (arrow: string) => parseArrow(arrow),
})
