package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/tengen-io/server/models"
	"strconv"
	"time"
)

// TODO(eac): Add validation
// TODO(eac): Switch to sqlx binding
func (r *Repository) CreateGame(gameType models.GameType, boardSize int, gameState models.GameState, users []models.User) (*models.Game, error) {
	var rv models.Game
	ts := pq.FormatTimestamp(time.Now().UTC())

	game := r.h.QueryRowx("INSERT INTO games (type, state, board_size, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, type, state, board_size", gameType, gameState, boardSize, ts, ts)
	err := game.Scan(&rv.Id, &rv.Type, &rv.State, &rv.BoardSize)
	if err != nil {
		return nil, err
	}

	insertStmt, err := r.h.Prepare("INSERT INTO game_user (game_id, user_id, type, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)")
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		_, err = insertStmt.Exec(rv.Id, user.Id, models.GameUserEdgeTypePlayer, ts, ts)
		if err != nil {
			return nil, err
		}
	}

	/*	payload := pubsub.Event{
			Event: "create",
			Payload: &rv,
		}

		r.pubsub.Publish(payload, "game", "game_"+rv.State.String()) */

	return &rv, nil
}

// TODO(eac): Validation? here or in the resolver
// TODO(eac): Need this anymore?
func (p *Repository) CreateGameUser(gameId string, userId string, edgeType models.GameUserEdgeType) (*models.Game, error) {
	/*	tx, err := p.h.BeginTx(context.TODO(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		if err != nil {
			return nil, err
		}
		defer tx.Rollback() */

	rows, err := p.h.Query("SELECT user_id, user_index, type FROM game_user WHERE game_id = $1 ORDER BY user_index ASC", gameId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	gameUsers := make([]models.GameUserEdge, 0)
	for rows.Next() {
		var gameUser models.GameUserEdge
		err := rows.Scan(&gameUser.User.Id, &gameUser.Index, &gameUser.Type)
		if err != nil {
			return nil, err
		}

		gameUsers = append(gameUsers, gameUser)
	}
	nextIndex := gameUsers[len(gameUsers)-1].Index + 1

	var rv models.Game
	ts := pq.FormatTimestamp(time.Now().UTC())
	_, err = p.h.Exec("INSERT INTO game_user (game_id, user_id, user_index, type, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)", gameId, userId, nextIndex, edgeType, ts, ts)
	if err != nil {
		return nil, err
	}

	newEdge := models.GameUserEdge{
		Index: nextIndex,
		Type:  edgeType,
		User: models.User{
			NodeFields: models.NodeFields{
				Id: userId,
			},
		},
	}
	gameUsers = append(gameUsers, newEdge)

	game := p.h.QueryRowx("SELECT id, type, state FROM games WHERE id = $1", gameId)
	err = game.Scan(&rv.Id, &rv.Type, &rv.State)
	if err != nil {
		return nil, err
	}

	return &rv, nil
}

func (p *Repository) GetGameById(id string) (*models.Game, error) {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	var game models.Game
	row := p.h.QueryRowx("SELECT * FROM games WHERE id = $1", idInt)
	err = row.StructScan(&game)
	if err != nil {
		return nil, err
	}

	return &game, nil
}

func (p *Repository) GetUsersForGame(id string) ([]models.GameUserEdge, error) {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	rows, err := p.h.Query("SELECT type, user_id, name FROM game_user gu, users u WHERE game_id = $1 AND gu.user_id = u.id", idInt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rv = make([]models.GameUserEdge, 0)
	for rows.Next() {
		var i models.GameUserEdge
		err := rows.Scan(&i.Type, &i.User.Id, &i.User.Name)
		if err != nil {
			return nil, err
		}

		rv = append(rv, i)
	}

	return rv, nil
}

func (p *Repository) GetGamesByIds(ids []string) ([]*models.Game, error) {
	idInts := make([]int, len(ids))
	for i, id := range ids {
		idInt, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		idInts[i] = idInt
	}

	query, fragArgs, err := sqlx.In("SELECT * FROM games WHERE id IN (?)", idInts)
	if err != nil {
		return nil, err
	}

	args := make([]interface{}, 0)
	args = append(args, fragArgs...)

	query = p.h.Rebind(query)
	rows, err := p.h.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rv := make([]*models.Game, 0)
	for rows.Next() {
		var i models.Game
		err := rows.StructScan(&i)
		if err != nil {
			return nil, err
		}
		rv = append(rv, &i)
	}

	return rv, nil
}

func (p *Repository) GetGamesByState(states []models.GameState) ([]*models.Game, error) {
	query, fragArgs, err := sqlx.In("SELECT * FROM games WHERE state IN (?)", states)
	if err != nil {
		return nil, err
	}

	args := make([]interface{}, 0)
	args = append(args, fragArgs...)

	query = p.h.Rebind(query)
	rows, err := p.h.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rv := make([]*models.Game, 0)
	for rows.Next() {
		var i models.Game
		err := rows.StructScan(&i)
		if err != nil {
			return nil, err
		}
		rv = append(rv, &i)
	}

	return rv, nil
}
