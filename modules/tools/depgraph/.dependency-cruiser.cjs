// Which way the interface modules point, and the drawing of it.
//
// Run once per module, from that module's own folder, so the module's own
// tsconfig answers for what `@/` means. Every path below is relative to the
// module being cruised.
//
// The direction rule — apps → libs/ui, never back and never sideways — is not
// written out here as a list of module names. It does not have to be: every
// manifest already names what its module may reach, and
// `no-reach-past-the-manifest` makes that list binding. A window importing
// the other window, or the components importing the schema, is an import of
// a package that module never declared.

const forbidden = [
  {
    name: 'no-reach-past-the-manifest',
    comment:
      'A module names what it may reach in its own package.json. ' +
      'An import of a package that is not in it resolves today only because ' +
      'somebody else installed it.',
    severity: 'error',
    from: {},
    to: { dependencyTypes: ['npm-no-pkg', 'npm-unknown'] },
  },
  {
    name: 'no-reach-out-of-the-module',
    comment:
      'A module reaches another by its name, not by a path out of its own ' +
      'folder. There is no module at the repository root, and a ' +
      'relative path across two of them is one the build cannot see.',
    severity: 'error',
    from: { path: '^(src|index\\.ts)', pathNot: '\\.(test|stories)\\.ts$' },
    to: { path: '^\\.\\.' },
  },
  {
    name: 'nothing-unresolved',
    comment: 'An import nothing answers: a package never installed, or a path that moved.',
    severity: 'error',
    from: {},
    to: { couldNotResolve: true },
  },
]

module.exports = {
  forbidden,
  options: {
    // The link a `file:` dependency makes is followed as the link it is, so a
    // window's use of the components reads as one edge to a package and stops
    // there. What is drawn is the shape of the source, not of an install.
    preserveSymlinks: true,
    doNotFollow: { path: 'node_modules' },
    exclude: { path: 'node_modules|/dist/|/storybook-static/|/coverage/' },
    tsConfig: { fileName: 'tsconfig.json' },
    enhancedResolveOptions: {
      extensions: ['.ts', '.tsx', '.vue', '.js', '.mjs', '.json'],
      exportsFields: ['exports'],
      conditionNames: ['import', 'require', 'node', 'default'],
    },
    reporterOptions: {
      // One box per folder, which is the level a component lives at.
      archi: { collapsePattern: '^(src/[^/]+|node_modules/(@[^/]+/)?[^/]+)' },
      dot: { collapsePattern: '^(src/[^/]+|node_modules/(@[^/]+/)?[^/]+)' },
    },
  },
}
