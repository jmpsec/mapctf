package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapCTFVersion(t *testing.T) {
	assert.Equal(t, "1.0.0", MapCTFVersion)
}
