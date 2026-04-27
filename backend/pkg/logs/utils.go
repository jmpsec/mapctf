package logs

import (
	"fmt"
)

const (
	// ActivityCreateChallenge is the log message template for creating a challenge
	ActivityCreateChallenge = "Challenge %s (%s) was created: %d points"
	// ActivityUpdateChallenge is the log message template for updating a challenge
	ActivityUpdateChallenge = "Challenge %s (%s) was updated: %d points"
	// ActivityEnableChallenge is the log message template for enabling a challenge
	ActivityEnableChallenge = "Challenge %s (%s) was enabled: %d points"
	// ActivityDisableChallenge is the log message template for disabling a challenge
	ActivityDisableChallenge = "Challenge %s (%s) was disabled: %d points"
	// ActivityTeamScore is the log message template for a team scoring points on a challenge
	ActivityTeamScore = "Team %s scored %d points for challenge %s (%s)"
)

func ScoreMessage(teamName string, points int, countryCode, categoryName string) string {
	return fmt.Sprintf(ActivityTeamScore, teamName, points, countryCode, categoryName)
}

func EnableMessage(countryCode, categoryName string, points int) string {
	return fmt.Sprintf(ActivityEnableChallenge, countryCode, categoryName, points)
}

func DisableMessage(countryCode, categoryName string, points int) string {
	return fmt.Sprintf(ActivityDisableChallenge, countryCode, categoryName, points)
}
