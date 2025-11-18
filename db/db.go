package db

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/techagentng/iweapp/config"
	"github.com/techagentng/iweapp/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GormDB struct {
	DB *gorm.DB
}

func GetDB(c *config.Config) *GormDB {
	gormDB := &GormDB{}
	gormDB.Init(c)
	return gormDB
}

func (g *GormDB) Init(c *config.Config) {
	g.DB = getPostgresDB(c)

	if err := migrate(g.DB); err != nil {
		log.Fatalf("unable to run migrations: %v", err)
	}
}

func getPostgresDB(c *config.Config) *gorm.DB {
	log.Printf("Connecting to postgres: %+v", c)
	postgresDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Africa/Lagos",
		c.PostgresHost, c.PostgresUser, c.PostgresPassword, c.PostgresDB, c.PostgresPort)

	// Create GORM DB instance
	gormConfig := &gorm.Config{}
	if c.Env != "prod" {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DSN: postgresDSN,
	}), gormConfig)
	if err != nil {
		log.Fatal(err)
	}

	return gormDB
}

func SeedRoles(db *gorm.DB) error {
	roles := []string{models.RoleAdmin, models.RoleUser}

	for _, roleName := range roles {
		var existingRole models.Role
		err := db.Where("name = ?", roleName).First(&existingRole).Error
		
		if err == gorm.ErrRecordNotFound {
			// Role doesn't exist, create it
			newRole := models.Role{
				ID:   uuid.New(),
				Name: roleName,
			}
			if err := db.Create(&newRole).Error; err != nil {
				return fmt.Errorf("failed to create role %s: %w", roleName, err)
			}
			log.Printf("Created role: %s", roleName)
		} else if err != nil {
			// Some other error occurred
			return fmt.Errorf("error checking role %s: %w", roleName, err)
		}
		// Role already exists, skip
	}

	return nil
}

func migrate(db *gorm.DB) error {
	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	// AutoMigrate all models to create tables
	err := db.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.Blacklist{},
		&models.OAuthState{},
		&models.UploadedFile{},
		&models.BankStatement{},
		&models.DocumentChunk{},
		&models.ProcessingJob{},
	)
	if err != nil {
		return fmt.Errorf("failed to run auto migrations: %w", err)
	}
	
	return nil
}
