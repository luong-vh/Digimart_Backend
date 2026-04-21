package dto

import "time"

type BanUserRequest struct {
	Reason   string     `json:"reason" binding:"required,max=500"`
	BanUntil *time.Time `json:"ban_until,omitempty"` // null = permanent ban
}
