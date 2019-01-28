package models

import (
	"testing"
)

func TestCreateGame(t *testing.T) {
	t.Run("with invalid user", createGameInvalidUser)
	t.Run("with invalid opponent", createGameInvalidOpponent)
	t.Run("with proper arguments", createGame)

	teardown()
}

func createGameInvalidUser(t *testing.T) {
	_, err := db.CreateGame(1234, &User{})

	expectErr(t, userNotFoundError{}, err)
}

func createGameInvalidOpponent(t *testing.T) {
	user, _ := db.CreateUser("creategame", "creategame@dude.dude", "dudedude", "dudedude")
	_, err := db.CreateGame(user.Id, &User{})

	expectErr(t, userNotFoundError{}, err)
}

func createGame(t *testing.T) {
	user1, err := db.CreateUser("creategame1", "creategame1@dude.dude", "dudedude", "dudedude")
	if err != nil {
		t.Error(err)
	}
	user2, err := db.CreateUser("creategame2", "creategame2@dude.dude", "dudedude", "dudedude")
	if err != nil {
		t.Error(err)
	}

	game, err := db.CreateGame(user1.Id, user2)

	if err != nil {
		t.Error(err)
	}

	if len(game.Players) != 2 {
		t.Error("Expected 2 Players in game")
	}
	if game.Status != "active" {
		t.Error("Expected game status to be 'active'")
	}
	if game.BoardSize != RegBoardSize {
		t.Errorf("Expected board size to be %d, got %d", RegBoardSize, game.BoardSize)
	}
}

func TestGetGames(t *testing.T) {
	t.Run("with user", getGamesByUser)
	t.Run("recent games", getGamesRecent)
	teardown()
}

func getGamesByUser(t *testing.T) {
	user1, _ := db.CreateUser("getgames1", "getgames1@dude.dude", "dudedude", "dudedude")
	user2, _ := db.CreateUser("getgames2", "getgames2@dude.dude", "dudedude", "dudedude")
	db.CreateGame(user1.Id, user2)

	games, err := db.GetGames(user1.Id)

	if err != nil {
		t.Error(err)
	}

	if len(games) != 1 {
		t.Errorf("Expected 1 game, found %d", len(games))
	}
}

func getGamesRecent(t *testing.T) {
	games, err := db.GetGames(nil)

	if err != nil {
		t.Error(err)
	}

	if len(games) != 1 {
		t.Errorf("Expected 1 game, found %d", len(games))
	}
}

func TestPass(t *testing.T) {
	user1, _ := db.CreateUser("pass1", "pass1@dude.dude", "dudedude", "dudedude")
	user2, _ := db.CreateUser("pass2", "pass2@dude.dude", "dudedude", "dudedude")
	game, _ := db.CreateGame(user1.Id, user2)

	turnId := game.PlayerTurnId

	game, err := db.Pass(user1.Id, game)

	if err != nil {
		t.Error(err)
	}

	if turnId == game.PlayerTurnId {
		t.Error("Expected player turn to change")
	}

	game, err = db.Pass(user2.Id, game)

	if err != nil {
		t.Error(err)
	}

	if game.Status != "complete" {
		t.Errorf("Expected game status to be 'complete', got '%s'", game.Status)
	}

	teardown()
}

func TestUpdateGame(t *testing.T) {
	user1, _ := db.CreateUser("UpdateGame1", "UpdateGame1@dude.dude", "dudedude", "dudedude")
	user2, _ := db.CreateUser("UpdateGame2", "UpdateGame2@dude.dude", "dudedude", "dudedude")
	game, _ := db.CreateGame(user1.Id, user2)

	stone := Stone{X: 0, Y: 0, Color: "black"}
	game.Stones = []Stone{stone}
	turnId := game.PlayerTurnId

	err := db.UpdateGame(user1.Id, game, stone, game.Stones)

	if err != nil {
		t.Error(err)
	}

	if turnId == game.PlayerTurnId {
		t.Error("Expected player turn to change")
	}

	if len(game.Stones) != 1 {
		t.Error("Expected Board to have 1 Stone")
	}

	teardown()
}
