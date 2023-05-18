package errors

import "errors"

var (
	ErrSecretNotFound          = errors.New("could not find existing secret")
	ErrInvalidSecret           = errors.New("the secret does not contain a valid key")
	ErrGitRepoPathNotSpecified = errors.New("the GIT_REPO_PATH environment variable was not specified")
)
