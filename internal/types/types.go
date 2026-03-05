package types

// CreateTokenRequest: user takes a token for a service
type CreateTokenRequest struct {
	ServiceCode  string `json:"service_code" validate:"required,oneof=A D L"`
	CustomerName string `json:"customer_name" validate:"omitempty,max=80"`
}

type CreateTokenResponse struct {
	Token         string `json:"token"`
	Position      int    `json:"position"`
	EstimatedMins int    `json:"estimated_minutes"`
}

// QueueStatusResponse: current serving token + waiting count
type QueueStatusResponse struct {
	CurrentToken string `json:"current_token"`
	Waiting      int    `json:"waiting"`
}

// NextResponse: admin calls next token
type NextResponse struct {
	CurrentToken string `json:"current_token"`
}
