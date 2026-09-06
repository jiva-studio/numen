// A screen reaches the shared folders, and never another screen.
//
// A window is a handful of screens and a few folders every screen uses. The
// five below are shared: the tabs a window is divided into, the commands, what
// a person is told, the test harness, and the writing out of unwritten work.
// Every other folder draws one screen, and a screen that reaches another is two
// screens that cannot be read, moved or deleted apart.
//
// The screens are not listed. Anything that is not shared is one, so a folder
// added tomorrow is a screen until somebody argues otherwise — which is the
// safe way round: a list would let a new folder in unread.
//
// This stands beside `rules.cjs` rather than in it because it is not true of
// every module. `modules/libs/ui` is a library of components and composition is
// the whole point of it: its cards folder draws a divider, its heatmap draws a
// calendar, and neither is a leak. A rule stretched over both packages would
// have to permit everything the library does, and would then refuse nothing.

const shared = ['command', 'notices', 'saving', 'tabs', 'testing']

/** Any folder under `src/` that is not one of the shared ones, captured. */
const SCREEN = `^src/(?!(?:${shared.join('|')})/)([^/]+)/`

module.exports = {
  extends: './rules.cjs',
  forbidden: [
    {
      name: 'no-screen-reaches-a-screen',
      comment:
        'A screen reaches the shared folders of its window and never another ' +
        'screen. What two screens both need is declared where it is needed and ' +
        'satisfied by whoever has it, or it stands in a shared folder. ' +
        `The shared folders are ${shared.join(', ')}.`,
      severity: 'error',
      from: { path: SCREEN },
      to: { path: SCREEN, pathNot: '^src/$1/' },
    },
  ],
}
