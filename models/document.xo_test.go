package models

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/y-yagi/configure"
)

type config struct {
	DataBase  string `toml:"database"`
	SelectCmd string `tomo:"selectcmd"`
}

func BenchmarkGetDocumentsBySQL(b *testing.B) {
	b.ResetTimer()
	db := getDB()
	defer db.Close()

	search := "import"

	for i := 0; i < b.N; i++ {
		GetDocumentsBySQL(db, "WHERE english = ? OR english LIKE ?", search, search+"%")
	}
}

func getDB() *sql.DB {
	var cfg config
	configure.Load("eijiro", &cfg)

	db, err := sql.Open("sqlite3", cfg.DataBase)
	if err != nil {
		panic(err)
	}

	return db
}
