package mizumanju

// loginParams は /api/login のリクエストパラメタを表す構造体。
type loginParams struct {
	Username, Password string
}

// imageParams は /api/saveImage のリクエストパラメタを表す構造体。
type imageParams struct {
	Image string
}

// statusParams は /api/users/me/status のリクエストパラメタを表す構造体。
type statusParams struct {
	Status string
}

// recoveryRequestParams は /api/recovery のリクエストパラメタを表す構造体
type recoveryRequestParams struct {
	Email string `json:"email"`
}

// recoveryParams は /api/recovery/{key} のリクエストパラメタを表す構造体
type recoveryParams struct {
	Password string `json:"password"`
}

// passwordParams は /api/users/{id}/password のリクエストパラメタを表す構造体
type passwordParams struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
