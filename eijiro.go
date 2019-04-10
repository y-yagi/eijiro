package eijiro

import (
	"bufio"
	"database/sql"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/y-yagi/debuglog"
	"github.com/y-yagi/eijiro/models"
	"github.com/y-yagi/goext/osext"
)

var (
	schema = `
CREATE TABLE documents (
	id integer primary key autoincrement not null,
	english varchar not null,
	japanese varchar not null,
	parts_of_speech varchar,
	text varchar not null
);

CREATE INDEX "index_documents_on_english" ON "documents" ("english");
CREATE INDEX "index_documents_on_japanese" ON "documents" ("japanese");
`
)

// Eijiro is a eijiro module.
type Eijiro struct {
	database string
	dlogger  *debuglog.Logger
}

// NewEijiro creates a new eijiro.
func NewEijiro(database string) *Eijiro {
	eijiro := &Eijiro{database: database, dlogger: debuglog.New(os.Stderr, debuglog.Flag(log.LstdFlags))}
	return eijiro
}

// InitDB initialize database.
func (e *Eijiro) InitDB() error {
	if osext.IsExist(e.database) {
		return nil
	}

	db, err := sql.Open("sqlite3", e.database)
	if err != nil {
		return err
	}
	defer db.Close()

	db.Exec(schema)

	return nil
}

// Import file to database
func (e *Eijiro) Import(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	os.Remove(e.database)
	err = e.InitDB()
	if err != nil {
		return nil
	}

	db, err := sql.Open("sqlite3", e.database)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	tx, _ := db.Begin()
	for scanner.Scan() {
		doc := models.Document{}
		doc.Text = scanner.Text()
		if strings.HasPrefix(doc.Text, "■") {
			doc.Text = strings.TrimPrefix(doc.Text, "■")
		}

		words := strings.Split(doc.Text, ":")
		doc.Japanese = strings.TrimSpace(words[1])
		startIndex := strings.IndexAny(words[0], "{")
		if startIndex != -1 {
			endIndex := strings.IndexAny(words[0], "}")
			doc.PartsOfSpeech = words[0][startIndex+1 : endIndex]
			doc.English = strings.TrimSpace(words[0][:startIndex])
		} else {
			doc.English = strings.TrimSpace(words[0])
		}

		doc.Insert(tx)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	return tx.Commit()
}

// Select select text from database
func (e *Eijiro) Select(search string) ([]models.Document, error) {
	e.dlogger.Print("Start sql.Open")
	db, err := sql.Open("sqlite3", e.database)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	e.dlogger.Print("Start GetDocumentsBySQL")
	if isASCII(search) {
		return models.GetDocumentsBySQL(db, "WHERE english = ? OR english LIKE ?", search, search+"%")
	}

	return models.GetDocumentsBySQL(db, "WHERE japanese LIKE ?", search+"%")
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
