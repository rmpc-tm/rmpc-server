package db

import (
	"database/sql"
	"errors"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"

	"rmpc-server/db/.gen/rmpc/public/model"
	"rmpc-server/db/.gen/rmpc/public/table"
)

func CreateSession(db *sql.DB, playerID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	// One row per player, held by the unique constraint on player_id. A single
	// upsert is what keeps that true when two logins race: a delete followed by
	// an insert lets the second login delete before the first has inserted, and
	// both then insert. Replacing the token also ends the previous session,
	// which is the intended "one session per player" behaviour.
	stmt := table.Sessions.INSERT(
		table.Sessions.PlayerID,
		table.Sessions.TokenHash,
		table.Sessions.ExpiresAt,
	).VALUES(
		playerID,
		tokenHash,
		expiresAt,
	).ON_CONFLICT(table.Sessions.PlayerID).DO_UPDATE(
		SET(
			table.Sessions.TokenHash.SET(String(tokenHash)),
			table.Sessions.ExpiresAt.SET(TimestampzT(expiresAt)),
			table.Sessions.CreatedAt.SET(TimestampzExpression(NOW())),
		),
	)

	_, err := stmt.Exec(db)
	return err
}

type Session struct {
	ID        uuid.UUID
	PlayerID  uuid.UUID
	ExpiresAt time.Time
}

func FindSessionByTokenHash(db *sql.DB, tokenHash string) (*Session, error) {
	stmt := SELECT(
		table.Sessions.ID,
		table.Sessions.PlayerID,
		table.Sessions.ExpiresAt,
	).FROM(
		table.Sessions,
	).WHERE(
		table.Sessions.TokenHash.EQ(String(tokenHash)),
	).LIMIT(1)

	var dest model.Sessions
	err := stmt.Query(db, &dest)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &Session{
		ID:        dest.ID,
		PlayerID:  dest.PlayerID,
		ExpiresAt: dest.ExpiresAt,
	}, nil
}
