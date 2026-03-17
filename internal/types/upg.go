// Package types provides shared domain and API types used across the service.
package types

// UPGStatus represents the result status of a UPG charge operation.
type UPGStatus string

const (
	UPGStatusAccepted         UPGStatus = "Accepted"
	UPGStatusSuccess          UPGStatus = "Success"
	UPGStatusRejected         UPGStatus = "Rejected"
	UPGStatusTemporaryFailure UPGStatus = "TemporaryFailure"
	UPGStatusFatalFailure     UPGStatus = "FatalFailure"
)
