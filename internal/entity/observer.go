package entity

type Observer interface {
	Update(game *Game, subject SubjectType)
}
