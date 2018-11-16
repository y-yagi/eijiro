package eijiro

import (
	"bufio"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
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
`
	insertQuery = `
INSERT INTO documents (english, japanese, parts_of_speech, text) VALUES ($1,$2, $3, $4)
`

	selectQuery = `
SELECT text FROM documents WHERE text LIKE $1
`
)

// Eijiro is a eijiro module.
type Eijiro struct {
	database string
}

// Document is type for `dictionaries` table
type Document struct {
	ID   int    `db:"id"`
	Text string `db:"text"`
}

// NewEijiro creates a new eijiro.
func NewEijiro(database string) *Eijiro {
	eijiro := &Eijiro{database: database}
	return eijiro
}

// InitDB initialize database.
func (e *Eijiro) InitDB() error {
	if osext.IsExist(e.database) {
		return nil
	}

	db, err := sqlx.Connect("sqlite3", e.database)
	if err != nil {
		return err
	}
	defer db.Close()

	db.MustExec(schema)

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

	db, err := sqlx.Connect("sqlite3", e.database)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	tx := db.MustBegin()
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "■") {
			text = strings.TrimPrefix(text, "■")
		}

		var en, ja, partsOfSpeech string

		words := strings.Split(text, ":")
		ja = strings.TrimSpace(words[1])
		startIndex := strings.IndexAny(words[0], "{")
		if startIndex != -1 {
			endIndex := strings.IndexAny(words[0], "}")
			partsOfSpeech = words[0][startIndex+1 : endIndex]
			en = strings.TrimSpace(words[0][:startIndex])
		} else {
			en = strings.TrimSpace(words[0])
		}

		tx.MustExec(insertQuery, en, ja, partsOfSpeech, text)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	tx.Commit()
	return nil
}

// Select select text from database
func (e *Eijiro) Select(search string) ([]Document, error) {
	db, err := sqlx.Connect("sqlite3", e.database)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	documents := []Document{}
	err = db.Select(&documents, selectQuery, search+"%")
	if err != nil {
		return nil, err
	}

	if len(documents) == 0 {
		if db.Select(&documents, selectQuery, "%"+search); err != nil {
			return nil, err
		}

	}

	return documents, nil
}
