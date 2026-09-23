/**
 * Conventional Commits — https://www.conventionalcommits.org
 *
 * Enforced locally by the .husky/commit-msg hook and in CI by
 * .github/workflows/commits.yml (so unhooked clones still get caught).
 */
export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Scopes follow the module layout. Warning, not error: adding a module
    // should not be blocked by a forgotten entry in this list — but a typo
    // should still be visible.
    'scope-enum': [
      1,
      'always',
      [
        'desktop',   // modules/apps/desktop
        'mobile',    // modules/apps/mobile
        'landing',   // modules/apps/landing
        'core',      // modules/libs/core
        'domain',    // modules/libs/core/domain
        'protocol',  // modules/libs/protocol
        'ui',        // modules/libs/ui
        'wire',      // modules/libs/wire
        'adr',       // docs/adr
        'docs',      // everything else under docs/
        'vault',     // vault format spec and parser-facing changes
        'ci',
        'deps',
        'repo',      // repo-level config, layout, tooling
      ],
    ],
    // git log stays readable in a 80-column terminal
    'header-max-length': [2, 'always', 72],
    'body-max-line-length': [1, 'always', 100],
  },
}
