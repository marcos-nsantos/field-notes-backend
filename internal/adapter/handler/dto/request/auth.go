package request

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Name     string `json:"name" binding:"required,min=1,max=255"`
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	DeviceID   string `json:"device_id" binding:"required,max=255"`
	DeviceName string `json:"device_name" binding:"max=255"`
	Platform   string `json:"platform" binding:"required,oneof=ios android web"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
