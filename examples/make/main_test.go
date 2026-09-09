package main

import (
	"testing"

	"github.com/feizaonet/cobrax/test"
)

func TestTools(t *testing.T) {
	test.Tools(t, makeCmd(), "make")
}
