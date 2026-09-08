// A screen reaches the shared folders, and never another screen.
//
// A folder under `src/` draws one screen, unless it is `shared/`, which is what
// every screen may take, or `window/`, which is the shell that mounts them. A
// screen that reaches another is two screens that cannot be read, moved or
// deleted apart. A sheet a screen draws over itself is that screen's own
// component, however deep the folder holding it.
//
// This holds for the windows alone, which are the modules `screened` names.
// `modules/libs/ui` is a library of components, where one drawing another is
// the whole point.

const apart = ['shared', 'window']

/** A screen's own folder, captured. */
const SCREEN = `^src/(?!(?:${apart.join('|')})/)([^/]+)/`

/** The same screen. */
const ITSELF = '^src/$1/'

module.exports = {
  extends: './rules.cjs',
  forbidden: [
    {
      name: 'no-screen-reaches-a-screen',
      comment:
        'A screen reaches `shared/` and never another screen. What two screens ' +
        'both need is declared where it is needed and satisfied by whoever has ' +
        'it, or it stands in `shared/`. The folders that are no screen are ' +
        `${apart.join(' and ')}.`,
      severity: 'error',
      from: { path: SCREEN },
      to: { path: SCREEN, pathNot: ITSELF },
    },
    {
      name: 'no-folder-going-round',
      comment:
        'Two folders that each reach the other have no order between them, so ' +
        'neither can be read first and neither can be taken out on its own. ' +
        'This is the same fault `no-going-round` refuses between two files, and ' +
        'that rule cannot see it: the ring runs through the folder boundary, ' +
        'and no single file of either folder is in a cycle.',
      severity: 'error',
      scope: 'folder',
      from: { path: '^src/[^/]+' },
      to: { circular: true },
    },
  ],
}
