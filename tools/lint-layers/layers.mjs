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
