package pet

import (
	"time"
)

type InteractionType string

const (
	InteractionVisit InteractionType = "visit"
	InteractionFeed  InteractionType = "feed"
	InteractionPlay  InteractionType = "play"
)

type Interaction struct {
	Time   time.Time       `toml:"time"`
	Action InteractionType `toml:"action"`
}

type PetState struct {
	ConfigRef    string `toml:"configRef"`
	NameOverride string `toml:"nameOverride"`

	Hunger    int `toml:"hunger"`
	Happiness int `toml:"happiness"`
	Energy    int `toml:"energy"`

	Evolution int `toml:"evolution"`

	IsInfirm bool `toml:"isInfirm"`
	IsStone  bool `toml:"isStone"`

	Message string `toml:"message"`

	LastFed     time.Time `toml:"lastFed"`
	LastPlayed  time.Time `toml:"lastPlayed"`
	LastVisited time.Time `toml:"lastVisited"`
	LastChecked time.Time `toml:"lastChecked"`

	LastVisits []Interaction `toml:"lastVisits"`
	LastFeeds  []Interaction `toml:"lastFeeds"`
	LastPlays  []Interaction `toml:"lastPlays"`
}
