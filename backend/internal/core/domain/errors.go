package domain

import "errors"

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrProjectExists      = errors.New("project already registered")
	ErrAgentNotFound      = errors.New("agent not found")
	ErrPRTrackingNotFound = errors.New("pr tracking not found")
	ErrPoolExhausted      = errors.New("pool exhausted")
	ErrForbidden          = errors.New("forbidden")
	ErrBoardNotFound      = errors.New("board not found")
	ErrCardNotFound       = errors.New("card not found")
	ErrInvalidColumn      = errors.New("invalid column")
)
