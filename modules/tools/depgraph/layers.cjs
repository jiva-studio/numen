// The layers a module's folders stand in, and which way they may point.
//
// A layer reaches what stands below it and never what stands above. A layer cut
// into slices has one rule more: a slice reaches no sibling slice of its own
// layer, and what two of them both need stands on a layer below.
//
// A folder under `src/` that names no layer is a screen, which is a slice
// standing where the pages do. That is how a window laid out flat is read, and
// it is the same rule: one screen never reaches another.
//
// `shared` and the shell are cut into no slices. Inside `shared` a component
// draws another — a select draws a menu, and a menu is shared — and the shell
// is one folder mounting the rest.

/**
 * Every layer, by the folder it is named after: how high it stands, and whether
 * it is cut into slices. Two names on one rank are one layer under two words,
 * which is how a library of components calls its pages `screens`.
 */
const LAYERS = {
  shared: { rank: 0, sliced: false },
  entities: { rank: 1, sliced: true },
  features: { rank: 2, sliced: true },
  widgets: { rank: 3, sliced: true },
  pages: { rank: 4, sliced: true },
  screens: { rank: 4, sliced: true },
  app: { rank: 5, sliced: false },
  window: { rank: 5, sliced: false },
}

/** Where a screen stands: the rank a folder naming no layer is read at. */
const SCREEN_RANK = LAYERS.pages.rank

const named = Object.keys(LAYERS)

/**
 * The test harness, which stands on no layer. It assembles the whole window,
 * and the tests of every layer draw it, so it points both ways by the nature of
 * what it is. No direction rule reads it, in either direction.
 */
const HARNESS = '^src/testing/'

/** A folder under `src/` naming no layer, captured. The harness is no screen. */
const SCREEN = `^src/(?!(?:${named.join('|')}|testing)/)([^/]+)/`

/** The folders standing above `rank`, as one alternation, and nothing where none do. */
const above = (rank) => named.filter((one) => LAYERS[one].rank > rank)

/**
 * What a file on `rank` may not reach: the folders standing above it, and a
 * screen too where screens stand above.
 */
function upward(rank) {
  const higher = above(rank)
  const reaches = higher.length > 0 ? [`^src/(?:${higher.join('|')})/`] : []
  if (rank < SCREEN_RANK) reaches.push(SCREEN)
  return reaches
}

const forbidden = []

// A layer reaches what stands below it. What stands above is refused by name,
// so the message says which layer was reached.
for (const [name, { rank }] of Object.entries(LAYERS)) {
  const reaches = upward(rank)
  if (reaches.length === 0) continue
  forbidden.push({
    name: `no-${name}-reaches-above-itself`,
    comment:
      `A file under \`${name}/\` reaches the layers below it and never one above. ` +
      'A file naming one is a file standing a layer too low.',
    severity: 'error',
    from: { path: `^src/${name}/`, pathNot: HARNESS },
    to: { path: reaches.join('|'), pathNot: HARNESS },
  })
}

// A slice reaches no sibling slice. `@x` is the one way through: a slice
// declaring one says which sibling it is for, and that is a decision written
// down rather than an import nobody weighed.
for (const [name, { sliced }] of Object.entries(LAYERS)) {
  if (!sliced) continue
  forbidden.push({
    name: `no-${name}-slice-reaches-a-slice`,
    comment:
      `A slice of \`${name}/\` reaches the layers below it and never a sibling slice. ` +
      'What two slices both need stands on a layer below, or is asked for through `@x`.',
    severity: 'error',
    from: { path: `^src/${name}/([^/]+)/`, pathNot: HARNESS },
    to: {
      path: `^src/${name}/([^/]+)/`,
      pathNot: [`^src/${name}/$1/`, `^src/${name}/[^/]+/@x/`, HARNESS],
    },
  })
}

// A window laid out flat: every folder under `src/` is a screen, and the rule
// over screens is the rule over slices.
forbidden.push({
  name: 'no-screen-reaches-a-screen',
  comment:
    'A screen reaches the shared layer and never another screen. Two screens ' +
    'reaching each other cannot be read, moved or deleted apart.',
  severity: 'error',
  from: { path: SCREEN },
  to: { path: SCREEN, pathNot: ['^src/$1/', HARNESS] },
})

forbidden.push({
  name: 'no-screen-reaches-above-itself',
  comment:
    'A screen is drawn by the shell and knows of no shell. A screen naming one ' +
    'belongs in it.',
  severity: 'error',
  from: { path: SCREEN },
  to: { path: upward(SCREEN_RANK).join('|'), pathNot: HARNESS },
})

forbidden.push({
  name: 'no-folder-going-round',
  comment:
    'Two folders that each reach the other have no order between them, so ' +
    'neither can be read first and neither can be taken out on its own. This is ' +
    'the same fault `no-going-round` refuses between two files, and that rule ' +
    'cannot see it: the ring runs through the folder boundary, and no single ' +
    'file of either folder is in a cycle.',
  severity: 'error',
  scope: 'folder',
  from: { path: '^src/[^/]+', pathNot: HARNESS },
  to: { circular: true, pathNot: HARNESS },
})

module.exports = {
  extends: './rules.cjs',
  forbidden,
  options: {
    // The harness stands on no layer, and a walk that reads it finds a ring
    // through every folder there is: each layer's tests draw it, and it draws
    // the window. It is left out of the walk, which is the same thing the
    // rules above say by not reading it.
    exclude: { path: '^(dist|storybook-static|coverage)/|^src/testing/' },
  },
}
