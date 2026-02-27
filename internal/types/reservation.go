package types

import "time"

// ReservationStatus represents the lifecycle state of a reservation.
type ReservationStatus string

const (
	ReservationStatusPending    ReservationStatus = "pending"
	ReservationStatusConfirmed  ReservationStatus = "confirmed"
	ReservationStatusCancelled  ReservationStatus = "cancelled"
	ReservationStatusCheckedIn  ReservationStatus = "checked_in"
	ReservationStatusCheckedOut ReservationStatus = "checked_out"
)

// Reservation represents a hotel reservation record.
type Reservation struct {
	ID                int               `db:"id"                 json:"id"`
	ReservationNumber string            `db:"reservation_number" json:"reservation_number"`
	PropertyID        int               `db:"property_id"        json:"property_id"`
	GuestName         string            `db:"guest_name"         json:"guest_name"`
	GuestEmail        string            `db:"guest_email"        json:"guest_email"`
	CheckIn           time.Time         `db:"check_in"           json:"check_in"`
	CheckOut          time.Time         `db:"check_out"          json:"check_out"`
	Status            ReservationStatus `db:"status"             json:"status"`
	// CardToken is the Vaultera-issued reference token for the stored card.
	// It is NOT raw card data and carries no PCI scope; however it is omitted
	// from serialized output when empty.
	CardToken   string    `db:"card_token"         json:"card_token,omitempty"`
	TotalAmount float64   `db:"total_amount"       json:"total_amount"`
	Currency    string    `db:"currency"           json:"currency"`
	CreatedAt   time.Time `db:"created_at"         json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"         json:"updated_at"`
}
