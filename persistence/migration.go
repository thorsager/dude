package persistence

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var efs embed.FS

func Migrate(dbUrl string) error {
	if lst, err := efs.ReadDir("migrations"); err == nil {
		for i, f := range lst {
			fmt.Printf("File %d: %s\n", i, f.Name())
		}
	}
	fs, err := iofs.New(efs, "migrations")
	if err != nil {
		return err
	}
	defer fs.Close()

	m, err := migrate.NewWithSourceInstance("iofs", fs, dbUrl)
	if err != nil {
		return err
	}
	defer m.Close()

	err = m.Up() // or m.Site
	if err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
