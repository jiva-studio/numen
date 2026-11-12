/**
 * The layer rules of the frontend modules, once, for every eslint config that
 * enforces them.
 *
 * `order` runs from the lowest layer to the highest. A layer reaches only the
 * layers below it, a flat layer reaches its own files, and a slice layer reaches
 * its own slice and a sibling's `crossImport` file. With `indexOnly`, a slice
 * layer is reached from above through its `public` file alone.
 */

/** The elements of the layers, in the shape `boundaries/elements` takes. */
export function createLayerElements({ order, sliceLayers, extra = [] }) {
  return [
    ...extra,
    ...order.map((type) =>
      sliceLayers.includes(type)
        ? { type, pattern: `src/${type}/*`, capture: ['slice'] }
        : { type, pattern: `src/${type}` },
    ),
  ]
}

/** The policies of the layers, in the shape `boundaries/dependencies` takes. */
export function createLayerPolicies({ order, sliceLayers, indexOnly = false, extra = [] }) {
  const reach = (type) => ({
    element: { type },
    ...(indexOnly && sliceLayers.includes(type) ? { file: { categories: 'public' } } : {}),
  })

  return [
    { allow: { to: { module: { origin: ['external', 'core'] } } } },
    ...extra,
    { from: { file: { categories: 'entry' } }, allow: { to: { element: { type: order } } } },
    ...order.flatMap((layer, index) =>
      order.slice(0, index).map((lower) => ({
        from: { element: { type: layer } },
        allow: { to: reach(lower) },
      })),
    ),
    ...order
      .filter((layer) => !sliceLayers.includes(layer))
      .map((layer) => ({
        from: { element: { type: layer } },
        allow: { to: { element: { type: layer } } },
      })),
    ...sliceLayers.flatMap((layer) => [
      {
        from: { element: { type: layer } },
        allow: {
          to: {
            element: { type: layer, captured: { slice: '{{ from.element.captured.slice }}' } },
          },
        },
      },
    ]),
  ]
}

/**
 * The eslint block that enforces the layers on `src/`. `boundaries` is the
 * eslint-plugin-boundaries instance of the calling package, `categories` the
 * `boundaries/files` entries, and `rules` any rule added to the same files.
 */
export function createLayerConfig({ boundaries, elements, policies, categories, ignores = [], rules = {} }) {
  return {
    files: ['src/**/*.{ts,vue}'],
    ignores,
    plugins: { boundaries },
    settings: {
      'import/resolver': { typescript: { alwaysTryTypes: true } },
      'boundaries/elements': elements,
      'boundaries/files': categories,
    },
    rules: {
      'boundaries/no-unknown-files': 'error',
      'boundaries/dependencies': ['error', { default: 'disallow', policies }],
      ...rules,
    },
  }
}

/**
 * The eslint block for the sources of one layer of a component library: the
 * imports `patterns` bans everywhere, and the alias `group` its own files must
 * not be imported by.
 */
export function createOwnImportConfig({ layer, patterns, group, message, outsideLayers }) {
  return {
    files: [`src/${layer}/**/*.{ts,vue}`],
    ignores: ['src/**/*.test.ts', 'src/**/*.spec.ts', ...outsideLayers],
    rules: {
      'no-restricted-imports': ['error', { patterns: [...patterns, { group, message }] }],
    },
  }
}

/**
 * The eslint block for the stories and fixtures of a component library. They stand outside the
 * layer order, but they are no place for the domain: a story draws a component with made-up props,
 * and a fixture is made-up content, so neither imports a wire type, a client, a domain type or an
 * application.
 */
export function createStoryImportConfig({ patterns }) {
  return {
    files: ['src/**/*.stories.ts', 'src/**/fixtures/**/*.ts'],
    rules: {
      'no-restricted-imports': ['error', { patterns }],
    },
  }
}

/**
 * The layer blocks of a component library (`libs/ui/components`, `libs/ui/workbench`):
 * the layers, the imports `domainImports` bans, and a slice of `features`
 * reaching no other slice. `outsideLayers` are the files that stand outside.
 */
export function createLibraryLayerConfigs({ boundaries, order, sliceLayers, outsideLayers, domainImports }) {
  return [
    createLayerConfig({
      boundaries,
      elements: createLayerElements({ order, sliceLayers }),
      policies: createLayerPolicies({ order, sliceLayers, indexOnly: true }),
      categories: [
        { category: 'public', pattern: 'src/features/*/index.ts' },
        { category: 'entry', pattern: 'src/index.ts' },
      ],
      ignores: outsideLayers,
      rules: { 'no-restricted-imports': ['error', { patterns: domainImports }] },
    }),
    createOwnImportConfig({
      layer: 'features',
      patterns: domainImports,
      group: ['@/features/**', '@/screens/**'],
      message: 'a slice imports its own files by relative path and no other slice',
      outsideLayers,
    }),
    createStoryImportConfig({ patterns: domainImports }),
  ]
}

/**
 * The imports a component library bans: every first-party package but
 * `@numen/ui` (`message` says why), and any path into `apps` or `libs`
 * (`direction` says which way the dependency runs).
 */
export function createDomainImports({ message, direction }) {
  return [
    {
      group: [
        '@numen/*',
        '@numen/*/**',
        '!@numen/ui',
        // The same packages named by the path the install put them at.
        '**/node_modules/@numen/**',
      ],
      message,
    },
    { group: ['**/apps/**', '**/libs/**'], message: direction },
  ]
}
