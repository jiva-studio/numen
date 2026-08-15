-- Deliberately not scoped to one vault. An identifier names one note in the
-- world, so a link written that way resolves wherever that note is — and which
-- vault it landed in comes back with it (ADR-0011).
--
-- Everywhere else in this package a missing vault_id is the silent bug ADR-0002
-- warns about. Here it is the decision.
--
-- Ordered because a copied note file carries a copied identifier, and two notes
-- claiming one identity must not resolve differently on two machines or between
-- two runs. Which of them wins is arbitrary; that it is the same one every time
-- is not.
SELECT vault_id, path
FROM notes
WHERE note_id = ?
ORDER BY vault_id, path
LIMIT 1;
