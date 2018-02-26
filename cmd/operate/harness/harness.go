package harness

import (
	"github.com/modulesio/butler/buse"
	itchio "github.com/itchio/go-itchio"
)

type Harness interface {
	ClientFromCredentials(credentials *buse.GameCredentials) (*itchio.Client, error)
}
