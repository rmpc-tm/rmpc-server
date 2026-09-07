-- One session per player is an application invariant. It was enforced by a
-- delete followed by an insert, which two concurrent logins can interleave:
-- both delete, then both insert. Enforce it in the schema so a single upsert
-- can hold it.

-- Collapse duplicates that already exist, keeping each player's newest session.
-- created_at is nullable, so coalesce before comparing; id breaks ties.
DELETE FROM sessions s
USING sessions newer
WHERE s.player_id = newer.player_id
  AND (COALESCE(s.created_at, 'epoch'::timestamptz), s.id)
    < (COALESCE(newer.created_at, 'epoch'::timestamptz), newer.id);

-- Redundant once player_id is unique: the constraint creates its own index.
DROP INDEX IF EXISTS idx_sessions_player_id;

ALTER TABLE sessions ADD CONSTRAINT sessions_player_id_key UNIQUE (player_id);
