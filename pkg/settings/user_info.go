package settings

import "regexp"

var userNameRegex = regexp.MustCompile(`^[_a-zA-Z0-9]{3,16}$`)

type UserInfo struct {
	Username string `json:"username" binding:"required,min=3,max=16"`
	Password string `json:"password" binding:"required,min=6,max=30"`
}

func NewUserInfo(userName, password string) *UserInfo {
	return &UserInfo{
		Username: userName,
		Password: password,
	}
}

func (u UserInfo) IsValidUsername() bool {
	return userNameRegex.MatchString(u.Username)
}
