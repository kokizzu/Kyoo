package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type User struct {
	// Primary key in database
	Pk int32 `json:"-"`
	// Id of the user.
	Id uuid.UUID `json:"id" example:"e05089d6-9179-4b5b-a63e-94dd5fc2a397"`
	// Username of the user. Can be used as a login.
	Username string `json:"username" example:"zoriya"`
	// Email of the user. Can be used as a login.
	Email string `json:"email" format:"email" example:"kyoo@zoriya.dev"`
	// False if the user has never setup a password and only used oidc.
	HasPassword bool `json:"hasPassword"`
	// When was this account created?
	CreatedDate time.Time `json:"createdDate" example:"2025-03-29T18:20:05.267Z"`
	// When was the last time this account made any authorized request?
	LastSeen time.Time `json:"lastSeen" example:"2025-03-29T18:20:05.267Z"`
	// List of custom claims JWT created via get /jwt will have
	Claims jwt.MapClaims `json:"claims" example:"isAdmin: true"`
	// List of other login method available for this user. Access tokens wont be returned here.
	Oidc map[string]OidcHandle `json:"oidc"`
}

type OidcHandle struct {
	// Id of this oidc handle.
	Id string `json:"id" example:"e05089d6-9179-4b5b-a63e-94dd5fc2a397"`
	// Username of the user on the external service.
	Username string `json:"username" example:"zoriya"`
	// Link to the profile of the user on the external service. Null if unknown or irrelevant.
	ProfileUrl *string `json:"profileUrl" format:"url" example:"https://myanimelist.net/profile/zoriya"`
}
type OidcMap = map[string]OidcHandle

type RegisterDto struct {
	// Username of the new account, can't contain @ signs. Can be used for login.
	Username string `json:"username" validate:"required,excludes=@" example:"zoriya"`
	// Valid email that could be used for forgotten password requests. Can be used for login.
	Email string `json:"email" validate:"required,email" format:"email" example:"kyoo@zoriya.dev"`
	// Password to use.
	Password string `json:"password" validate:"required" example:"password1234"`
}

type EditUserDto struct {
	Username *string       `json:"username,omitempty" validate:"omitnil,excludes=@" example:"zoriya"`
	Email    *string       `json:"email,omitempty" validate:"omitnil,email" example:"kyoo@zoriya.dev"`
	Claims   jwt.MapClaims `json:"claims,omitempty" example:"preferOriginal: true"`
}

type EditPasswordDto struct {
	OldPassword *string `json:"oldPassword" example:"password1234"`
	NewPassword string  `json:"newPassword" validate:"required" example:"password1234"`
}
