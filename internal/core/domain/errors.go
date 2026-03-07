package domain

import "errors"

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrProjectExists      = errors.New("project already registered")
	ErrAgentNotFound      = errors.New("agent not found")
	ErrPRTrackingNotFound = errors.New("pr tracking not found")
	ErrPoolExhausted      = errors.New("pool exhausted")
	ErrGiteaUnreachable   = errors.New("gitea unreachable")
	ErrForbidden          = errors.New("forbidden")
)
