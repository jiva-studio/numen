// Three layers under `src/`, each reaching what stands below it and no more.
//
// `shared/` is reached by every layer and reaches none of them; `features/`
// reaches `shared/`; `screens/` reaches both. One feature never reaches
// another. Inside `shared/` a component draws another — a select draws a menu,
// and a menu is shared — which the first rule below allows by naming only
// `features/`.
//
// This holds for the component library, which is the module `layered` names.

/** Any folder under `src/features/`, captured. */
const FEATURE = '^src/features/([^/]+)/'

module.exports = {
  extends: './rules.cjs',
  forbidden: [
    {
      name: 'no-feature-reaches-a-feature',
      comment:
        'A feature reaches the shared layer and never another feature. What ' +
        'two features both need is declared where it is needed and satisfied ' +
        'by whoever has it, or it stands in `shared/`.',
      severity: 'error',
      from: { path: FEATURE },
      to: { path: FEATURE, pathNot: '^src/features/$1/' },
    },
    {
      name: 'no-shared-reaches-above-itself',
      comment:
        'The shared layer is what every other layer may reach, so it knows of ' +
        'no feature and no screen. A shared module that names one is a feature ' +
        'standing one layer too low.',
      severity: 'error',
      from: { path: '^src/shared/' },
      to: { path: '^src/(features|screens)/' },
    },
    {
      name: 'no-feature-reaches-a-screen',
      comment:
        'A screen is what draws the features, so no feature knows there is ' +
        'one. A feature that names a screen belongs in that screen.',
      severity: 'error',
      from: { path: '^src/features/' },
      to: { path: '^src/screens/' },
    },
  ],
}
