-- Which sources of a vault were modified most recently, asked in order to
-- embed newly written notes first.
CREATE INDEX sources_by_modified ON sources (vault_id, modified_at DESC, id DESC);
