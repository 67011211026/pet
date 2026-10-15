package models

import (
	"time"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type BookingStatus string

const (
	StatusPending   BookingStatus = "pending"
	StatusApproved  BookingStatus = "approved"
	StatusRejected  BookingStatus = "rejected"
	StatusCompleted BookingStatus = "completed"
)

type RoomStatus string

const (
	RoomStatusAvailable   RoomStatus = "available"
	RoomStatusMaintenance RoomStatus = "maintenance"
)

// User model
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Email        string    `gorm:"size:150;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         Role      `gorm:"size:20;not null;default:'user'" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Pets     []Pet     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"pets,omitempty"`
	Bookings []Booking `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"bookings,omitempty"`
}

// Pet model (Form 1)
type Pet struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	User           User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Name           string    `gorm:"size:100;not null" json:"name"`
	PetType        string    `gorm:"size:50;not null" json:"pet_type"` // Dog, Cat, Other
	Breed          string    `gorm:"size:100" json:"breed"`
	AllergiesNotes string    `gorm:"type:text" json:"allergies_notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Room model
type Room struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	RoomNumber    string     `gorm:"size:50;uniqueIndex;not null" json:"room_number"`
	RoomType      string     `gorm:"size:50;not null" json:"room_type"` // Standard, Deluxe, VIP Suite
	PricePerNight float64    `gorm:"not null" json:"price_per_night"`
	Status        RoomStatus `gorm:"size:20;not null;default:'available'" json:"status"`
	Description   string     `gorm:"size:255" json:"description"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Booking model (Form 2)
type Booking struct {
	ID           uint          `gorm:"primaryKey" json:"id"`
	UserID       uint          `gorm:"not null;index" json:"user_id"`
	User         User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	PetID        uint          `gorm:"not null" json:"pet_id"`
	Pet          Pet           `gorm:"foreignKey:PetID;constraint:OnDelete:CASCADE;" json:"pet"`
	RoomID       uint          `gorm:"not null;index" json:"room_id"`
	Room         Room          `gorm:"foreignKey:RoomID" json:"room"`
	CheckInDate  time.Time     `gorm:"not null" json:"check_in_date"`
	CheckOutDate time.Time     `gorm:"not null" json:"check_out_date"`
	Status       BookingStatus `gorm:"size:20;not null;default:'pending'" json:"status"`
	TotalPrice   float64       `gorm:"not null" json:"total_price"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}
