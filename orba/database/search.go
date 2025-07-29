package database

// search.go is manually defined SQL queries to interact with SQLite's FTS5
// extension, as sqlc doesn't natively support it.
// Note that this will need to be MANUALLY updated on schema change!

import "context"

const searchMemories = `select
	sr.rowid,
	m.user_id,
	m.source_id,
	m.created_at,
	m.created_date,
	m.memory
from memories_fts sr
join memories m
	on m.id=sr.rowid
where
	memories_fts match format('user_id:%s', ?)
	and memories_fts match ?;
`

// SearchMemories searches the fts5 index for memories in the context of a user
func (q *Queries) SearchMemories(ctx context.Context, user *User, query string) ([]Memory, error) {
	rows, err := q.db.QueryContext(ctx, searchMemories,
		user.ID,
		query,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var items []Memory
	for rows.Next() {
		var i Memory
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.SourceID,
			&i.CreatedAt,
			&i.CreatedDate,
			&i.Memory,
		); err != nil {
			return nil, err
		}

		// Safety filter! In case the user_id matcher fails magically
		if i.UserID != user.ID {
			continue
		}

		items = append(items, i)
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
