# Test vaults

Two vaults, for two different jobs.

## `vault/` — the fixture

Eight notes, chosen to be awkward: no frontmatter, broken YAML, CRLF endings,
text that is not Latin, a PDF, a hidden folder, a link block. Parser and scanner
tests read it. It is small enough to assert exact counts against.

## `demo-vault/` — something to look at

Two hundred notes on mathematics, physics and computation, with a hierarchy
three levels deep, prerequisites, and links that cross from one branch to
another. Written for walking around in: the plex needs a vault where a
neighbourhood is worth seeing.

Every note carries an identifier. Notes hold a title and their links and almost
no prose — the shape is the point, not the reading.

Some of it is deliberately wrong, because a vault that is entirely correct
exercises none of the code that exists for vaults that are not:

| Where | What |
| --- | --- |
| `physics/thermodynamics/Maxwell relations.md` | a link to a note that does not exist |
| `mathematics/probability/Convergence.md` | shares a filename with one in `analysis/`, and reaches it by path |
| `Open questions.md` | a link with no role |
| `edge/Chicken.md`, `edge/Egg.md` | each is the other's parent |
| `edge/Written outside the application.md` | no identifier, as a file made in an editor has none |

### Running against it

```
numen --index /tmp/demo.db --registry /tmp/demo.json vault add tests/demo-vault --name demo
numen --index /tmp/demo.db --registry /tmp/demo.json scan demo
numen --index /tmp/demo.db --registry /tmp/demo.json links demo mathematics/algebra/Eigenvector.md
```
