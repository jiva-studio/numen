# Search Relevance

Sets how similar a note must be to your search to appear in the results. Matches below the threshold are left out of the results.

<!-- more -->

## What You Can Do

- **Set the threshold automatically**: Numen measures your vault once its vectors exist and uses the threshold that excludes unrelated notes.
- **Set it yourself**: Move a slider between 0.05 and 0.95 when the automatic value does not suit your vault.
- **Check a query**: See the best match in your vault for a search and whether it passes the threshold.

## Search Relevance

Open **Search relevance** in the settings of this plugin.

- **Auto / Manual**: In **Auto**, Numen picks the threshold for you. In **Manual**, a slider sets it between 0.05 and 0.95, and a marker on the slider shows the automatic value. A manual threshold applies to the current model only, so each model has its own.
- **Threshold and Recalculate**: Shows the threshold in use. **Recalculate** measures it again on your vault.
- **Source**: Shows where the threshold comes from: measured on your vault, the starting value of the model, a default, or set by you.
- **Noise**: The chart shows how similar unrelated notes are to each other, with a line at the threshold. The threshold is set just above this noise, so unrelated notes are not in your results. Below the chart you see how many titles and passages were compared and when.
- **Not enough notes yet**: Measuring needs at least 20 notes with meaning-based search indexed. Until then Numen uses the starting value of the model.
- **Check a query**: Type a search and press **Check** to see the best match in your vault and whether it passes the threshold.

Changing the model selects the threshold for the new model. If the embedding model is not running, the panel shows the error and you can try again.

## When the Plugin Is Off

Numen uses the last threshold the plugin set. Without one it uses the starting value of the model, and without that, 0.30. Turning the plugin on again restores your manual threshold and the last measurement.

## For Plugin Authors

The plugin has the permissions `index:read`, `search:policy` and `vault:read`. It reads the note titles through the vault calls and the embedding state through `HostSearchService`, embeds titles and queries with the model in use, has the host compare the title vectors with the passages of other notes (`SampleSimilarities`), and stores the floor it chooses with `SetSearchFloor`. The host stores one floor for each vault and recipe and applies it to every search. The plugin keeps its measurements in its folder of the vault's store. The window calls the plugin through `SearchRelevanceService`.
