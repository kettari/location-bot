package entity

type Game struct {
	ID       string `json:"id"`
	Joinable bool   `json:"joinable"`
	URL      string `json:"url"`
	Title    string `json:"title"`
}

func NewGame(id string, canjoin string, url string, title string) *Game {
	joinable := false
	if canjoin == "can-join" {
		joinable = true
	}
	return &Game{ID: id, Joinable: joinable, URL: url, Title: title}
}
