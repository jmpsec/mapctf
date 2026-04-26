package logs

import (
	"fmt"
)

const (
	// ActivityEnableChallenge is the log message template for enabling a challenge
	ActivityEnableChallenge = "Challenge %s (%s) was enabled: %d points"
	// ActivityDisableChallenge is the log message template for disabling a challenge
	ActivityDisableChallenge = "Challenge %s (%s) was disabled: %d points"
	// ActivityTeamScore is the log message template for a team scoring points on a challenge
	ActivityTeamScore = "Team %s scored %d points for challenge %s (%s)"
)

func GenerateMessage(template string, args ...interface{}) string {
	return fmt.Sprintf(template, args...)
}
