package eijiro

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/y-yagi/debuglog"
	"github.com/y-yagi/eijiro/models"
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
	db       *sql.DB
}

// NewEijiro creates a new eijiro.
func NewEijiro(database string) *Eijiro {
	eijiro := &Eijiro{database: database, dlogger: debuglog.New(os.Stderr, debuglog.Flag(log.LstdFlags))}
	return eijiro
}

// Init initialize Eijiro.
func (e *Eijiro) Init() error {
	db, err := sql.Open("sqlite3", e.database+"?_case_sensitive_like=on")
	if err != nil {
		return err
	}

	e.db = db
	return nil
}

// Terminate terminate Eijiro.
func (e *Eijiro) Terminate() error {
	return e.db.Close()
}

// Migrate run migration file.
func (e *Eijiro) Migrate() error {
	_, err := e.db.Exec(schema)
	return err
}

// Import file to database
func (e *Eijiro) Import(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	os.Remove(e.database)
	err = e.Init()
	if err != nil {
		return nil
	}

	err = e.Migrate()
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(file)
	tx, _ := e.db.Begin()
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
func (e *Eijiro) Select(search string) ([]string, error) {
	e.dlogger.Print("Start GetDocumentsBySQL")
	if isASCII(search) {
		return models.GetDocumentsBySQL(e.db, "WHERE english = ? OR english LIKE ?", search, search+"%")
	}

	return models.GetDocumentsBySQL(e.db, "WHERE japanese LIKE ?", search+"%")
}

// SelectViaCmd select text from database via uses `sqlite3` command
func (e *Eijiro) SelectViaCmd(search string) (string, error) {
	var query string

	if isASCII(search) {
		query = fmt.Sprintf("SELECT text FROM documents WHERE english = '%s' OR english LIKE '%s'", search, search+"%")
	} else {
		query = fmt.Sprintf("SELECT text FROM documents WHERE japanese LIKE '%s'", search+"%")
	}
	out, err := exec.Command("sqlite3", e.database, query).Output()
	return string(out), err
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

//lint:ignore U1000 this is for debug purpose.
func (e *Eijiro) explain() {
	query := "explain query plan SELECT text FROM documents WHERE english = 'import' OR english LIKE 'import%' LIMIT 50;"

	var result string
	var id1, id2, id3 int

	err := e.db.QueryRow(query).Scan(&id1, &id2, &id3, &result)

	if err != nil {
		fmt.Printf("err: %v\n", err)
	} else {
		fmt.Printf("explain: %s\n", result)
	}
}
