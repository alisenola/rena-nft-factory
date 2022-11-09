package database

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewDatabase creates a new database with given config
func NewDatabase() (*gorm.DB, error) {
	var (
		db  *gorm.DB
		err error
	)

	for i := 0; i <= 30; i++ {
		db, err = gorm.Open(postgres.Open("postgres://postgres:password@nft-market.cqtyaecn0tih.us-east-2.rds.amazonaws.com:5432/postgres?sslmode=disable"), &gorm.Config{})
		if err != nil {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil {
		return nil, err
	}

	origin, err := db.DB()
	if err != nil {
		return nil, err
	}
	origin.SetMaxOpenConns(50)
	origin.SetMaxIdleConns(5)
	origin.SetConnMaxLifetime(time.Duration(5) * time.Second)

	return db, nil
}
