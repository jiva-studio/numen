# Search Relevance

Chooses how similar a note has to be to your search before it shows up. Matches below the threshold are left out of the results.

<!-- more -->

## What You Can Do

- **Let Numen pick the threshold**: Numen measures your vault once its vectors exist and keeps the threshold that leaves unrelated notes out.
- **Set it yourself**: Move a slider between 0.05 and 0.95 when the automatic value does not suit your vault.
- **Check a query**: See the best match in your vault for a search and whether it passes the threshold.

## Search Relevance

Open **Search relevance** in the settings of this plugin.

- **Auto / Manual**: In **Auto**, Numen picks the threshold for you. In **Manual**, a slider sets it between 0.05 and 0.95, and a marker on the slider shows the automatic value. A manual threshold applies to the current model only, so each model keeps its own.
- **Threshold and Recalculate**: Shows the threshold in use. **Recalculate** measures it again on your vault.
- **Source**: Says where the threshold comes from: measured on your vault, the starting value of the model, a default, or set by you.
- **Noise**: The chart shows how similar unrelated notes are to each other, with a line at the threshold. The threshold sits just above this noise, so unrelated notes stay out of your results. Below the chart you see how many titles and passages were compared and when.
- **Not enough notes yet**: Measuring needs at least 20 notes with meaning-based search indexed. Until then Numen uses the starting value of the model.
- **Check a query**: Type a search and press **Check** to see the best match in your vault and whether it passes the threshold.

Changing the model gives you the threshold that suits the new model. If the embedding model is not running, the panel shows the error and you can try again.

## When the Plugin Is Off

Numen keeps the last threshold the plugin set. Without one it uses the starting value of the model, and without that, 0.30. Turning the plugin on again keeps your manual threshold and the last measurement.

## For Plugin Authors

The plugin holds the permissions `index:read` and `search:policy`. It reads the note titles, the stored vectors and the embedding state of a vault through `HostSearchService`, embeds titles and queries with the model in use, and stores the floor it chooses with `SetSearchFloor`. The host keeps one floor for each vault and recipe and applies it to every search. The plugin keeps its measurements in its folder of the vault's store. The window talks to the plugin through `SearchRelevanceService`.
