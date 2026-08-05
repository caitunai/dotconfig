// Package db
package db

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var db *gorm.DB

func Conn() *gorm.DB {
	if db != nil {
		return db
	}
	// refer https://github.com/go-sql-driver/mysql#dsn-data-source-name for details
	user := viper.GetString("db.username")
	pass := viper.GetString("db.password")
	dbname := viper.GetString("db.database")
	host := viper.GetString("db.host")
	port := viper.GetString("db.port")
	tls := viper.GetBool("db.tls")
	if user == "" || pass == "" || dbname == "" || host == "" || port == "" {
		log.Fatal().Msg("mysql db config should not empty")
	}
	dsn := "%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC"
	if tls {
		dsn += "&tls=true"
	}
	prefix := viper.GetString("db.table.prefix")
	if prefix != "" && !strings.HasSuffix(prefix, "_") {
		prefix += "_"
	}
	dsn = fmt.Sprintf(dsn, user, pass, host, port, dbname)
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   NewLogger(),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: prefix,
		},
	})
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	migrateDB()
	return db
}

func DB() *gorm.DB {
	return Conn()
}

func migrateDB() {
	err := db.AutoMigrate()
	if err != nil {
		log.Error().Err(err).Msg("db auto migrate failed")
	}
}
