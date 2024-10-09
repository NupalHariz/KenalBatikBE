package database

import "kenalbatik-be/internal/domain"

func Migrate() {
	err := DB.Migrator().AutoMigrate(
		&domain.Province{},
		&domain.Island{},
		&domain.Batik{},
		&domain.User{},
		&domain.Quiz{},
		&domain.UserAnswer{},
	)
	if err != nil {
		panic(err)
	}
}