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
      'relative path across two of them is one the build cannot see. A test ' +
      'and a story are outside the rule: the corpora they read stand in the ' +
      'schema package, which exports none of them. So is an installed ' +
      'package: the install hoists to the source root, so where a package ' +
      'landed is the installer\'s business and `no-reach-past-the-manifest` ' +
      'is what answers for whether the module may name it.',
    severity: 'error',
    from: { path: '^(src|index\\.ts)', pathNot: '\\.(test|stories)\\.ts$' },
    to: { path: '^\\.\\.', pathNot: '(^|/)node_modules/' },
  },
  {
    name: 'no-path-into-the-install',
    comment:
      'A module names a package; where the installer put it is the ' +
      "installer's business. A relative path into node_modules is read by no " +
      'manifest, so `no-reach-past-the-manifest` has nothing to answer for, ' +
      'and the path breaks the day the install hoists one folder further up.',
    severity: 'error',
    from: {},
    to: { path: '(^|/)node_modules/', dependencyTypes: ['local'] },
  },
  {
    name: 'nothing-unresolved',
    comment: 'An import nothing answers: a package never installed, or a path that moved.',
    severity: 'error',
    from: {},
    to: { couldNotResolve: true },
  },
  {
    name: 'no-going-round',
    comment:
      'Two files that each reach the other have no order between them, so ' +
      'neither can be read first and neither can be taken out on its own. ' +
      'A ring holding an import of a type alone is not one: nothing is ' +
      'loaded to satisfy it and the build erases the edge.',
    severity: 'error',
    from: {},
    to: { circular: true, viaOnly: { dependencyTypesNot: ['type-only'] } },
  },
  {
    name: 'no-screen-at-the-root',
    comment:
      'What stands at the root of a module is what every folder under it ' +
      'shares. A component is not that: it belongs with the folder that draws ' +
      'it, and one standing at the root is reached from folders that have ' +
      'nothing else to do with each other. The window a module opens on is ' +
      'the exception, being the root itself.',
    severity: 'error',
    from: {},
    to: { path: '^src/(?!App\\.vue$)[^/]+\\.vue$' },
  },
]

module.exports = {
  forbidden,
  options: {
    // The link the workspace root makes to a package of ours is followed as the
    // link it is, so a window's use of the components reads as one edge to a
    // package and stops there. Followed to what it points at, the components
    // stop being a package and their build is walked file by file.
    preserveSymlinks: true,
    // Which imports a type alone is read off the compiler, so `import type` is
    // one edge everywhere and a `.vue` is put through TypeScript.
    tsPreCompilationDeps: true,
    // A package is one node and is not walked into, and it is still a node: an
    // import of something the manifest never named is an edge, and a graph with
    // no packages in it has no such edge to refuse.
    doNotFollow: { path: 'node_modules' },
    // The module's own build output and its coverage, anchored at its root. A
    // pattern loose enough to say `/dist/` takes every package whose entry
    // stands in one with it.
    exclude: { path: '^(dist|storybook-static|coverage)/' },
    tsConfig: { fileName: 'tsconfig.json' },
    enhancedResolveOptions: {
      extensions: ['.ts', '.tsx', '.vue', '.js', '.mjs', '.json'],
      exportsFields: ['exports'],
      conditionNames: ['import', 'require', 'node', 'default'],
    },
    reporterOptions: {
      // One box per folder of ours and one per package, which is the level a
      // drawing reads at. A scope is tried before the bare folder so that
      // `@numen/ui` is a box and `@numen` is not.
      archi: { collapsePattern: '^(node_modules/@[a-z0-9-]+|node_modules|src)/[^/]+' },
      dot: { collapsePattern: '^(node_modules/@[a-z0-9-]+|node_modules|src)/[^/]+' },
    },
  },
}
