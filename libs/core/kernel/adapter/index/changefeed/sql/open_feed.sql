-- The feed of a vault, with an identity no other feed has. A vault that has one keeps it.
INSERT INTO change_feeds (vault_id, feed_id) VALUES (?, lower(hex(randomblob(16))))
ON CONFLICT (vault_id) DO NOTHING;
