package polecat

import (
	"os"
	"testing"

	"github.com/camp-leatherneck/camp-leatherneck/internal/testutil"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testutil.TerminateDoltContainer()
	os.Exit(code)
}
