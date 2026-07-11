package graph

import (
	"github.com/wricardo/colorsortgame"
	"github.com/wricardo/colorsortgame/graphqlapi/graph/model"
)

func toDifficulty(s string) model.Difficulty {
	switch s {
	case "medium":
		return model.DifficultyMedium
	case "hard":
		return model.DifficultyHard
	default:
		return model.DifficultyEasy
	}
}

func toTubes(tubes []colorsort.Tube) [][]string {
	out := make([][]string, len(tubes))
	for i, t := range tubes {
		out[i] = append([]string{}, t...)
	}
	return out
}

func toModelLevel(l *colorsort.Level) *model.Level {
	return &model.Level{
		ID:           int32(l.ID),
		Difficulty:   toDifficulty(l.Difficulty),
		TubeCapacity: int32(l.Capacity),
		Tubes:        toTubes(l.Tubes),
	}
}

// toModelGame converts a stored save into its GraphQL representation. The
// save doesn't carry its level's difficulty, so it's looked up again.
func (r *Resolver) toModelGame(id string, s *colorsort.Save) *model.Game {
	difficulty := ""
	if lvl, err := colorsort.FindLevel(r.levels, s.LevelID); err == nil {
		difficulty = lvl.Difficulty
	}

	history := make([]*model.Move, len(s.History))
	for i, m := range s.History {
		history[i] = &model.Move{From: int32(m.From), To: int32(m.To)}
	}

	return &model.Game{
		ID:         id,
		LevelID:    int32(s.LevelID),
		Difficulty: toDifficulty(difficulty),
		Capacity:   int32(s.Capacity),
		Tubes:      toTubes(s.Tubes),
		Moves:      int32(s.Moves),
		History:    history,
		Solved:     s.Solved,
		Stuck:      s.Stuck,
	}
}
