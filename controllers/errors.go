package controllers

import (
	"errors"
)

var ErrSecretNotFound = errors.New("could not find existing secret")
var ErrInvalidSecret = errors.New("the secret does not contain a valid key")
