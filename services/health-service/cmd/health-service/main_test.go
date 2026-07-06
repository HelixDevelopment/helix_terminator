package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEndpoints_Empty(t *testing.T) {
	result := parseEndpoints("")
	assert.NotNil(t, result)
	assert.Empty(t, result)
}
