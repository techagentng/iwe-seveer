package models

type Blacklist struct {
	Model
	Token string `json:"token" gorm:"not null;index"`
	Email string `json:"email" gorm:"default:null"`
}
