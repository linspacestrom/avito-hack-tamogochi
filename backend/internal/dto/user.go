package dto

import "time"

type UserResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
