package entity

type Schedule struct {
	Games []Game `json:"games"`
}

func NewSchedule() *Schedule {
	return &Schedule{}
}
