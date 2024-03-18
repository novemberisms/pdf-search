package pdfsearch

import (
	"bufio"
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/novemberisms/pdf-search/db"
	"golang.org/x/text/unicode/norm"
	"os"
	"strings"
	"unicode"
)

type PdfSearcher struct {
	conn    *sql.DB
	queries *db.Queries
	log     func(string)
}

type PdfSearcherOptions struct {
	DatabaseName string
	Log          func(string)
}

//go:embed db/schema.sql
var schema string

func NewPdfSearcher(options PdfSearcherOptions) (*PdfSearcher, error) {
	// normalize options
	if options.Log == nil {
		options.Log = func(string) {}
	}
	if options.DatabaseName == "" {
		options.DatabaseName = "index.sqlite"
	}

	connectionString := "file:" + options.DatabaseName + "?_journal_mode=WAL&_synchronous=normal&_timeout=10000"

	conn, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, err
	}

	// create the schema
	if _, err := conn.Exec(schema); err != nil {
		return nil, err
	}

	return &PdfSearcher{
		conn:    conn,
		queries: db.New(conn),
		log:     options.Log,
	}, nil
}

func (p *PdfSearcher) Close() {
	_ = p.conn.Close()
}

func (p *PdfSearcher) GetIndexedFiles(ctx context.Context) ([]string, error) {
	return p.queries.GetIndexedFiles(ctx)
}

func (p *PdfSearcher) IsIndexed(ctx context.Context, filepath string) (bool, error) {
	isIndexed, err := p.queries.IsFileIndexed(ctx, filepath)
	if err != nil {
		return false, err
	}
	return isIndexed != 0, nil
}

func (p *PdfSearcher) IndexTxtFile(ctx context.Context, filepath string) error {
	// we need to ensure that the file exists and is a txt
	if !strings.HasSuffix(filepath, ".txt") {
		return fmt.Errorf("file is not a txt")
	}

	if _, err := os.Stat(filepath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file does not exist")
	}

	// then, we need to clear out any existing entries for this file
	if err := p.queries.DeleteTextsByFile(ctx, filepath); err != nil {
		return err
	}

	// acquire a handle to the file and iterate over each line
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// iterate over each line in the txt file
	scanner := bufio.NewScanner(file)

	currentPageNo := 1
	currentPageText := strings.Builder{}

	p.log("indexing " + filepath)

	for scanner.Scan() {
		line := scanner.Text()

		// if the line begins with the marker START OF PAGE xx, then we update the current page number to xx
		if strings.HasPrefix(line, "START OF PAGE ") {
			// update the current page number
			_, err := fmt.Sscanf(line, "START OF PAGE %d", &currentPageNo)
			if err != nil {
				return err
			}
			p.log(fmt.Sprintf("page %d found", currentPageNo))
			continue
		}

		// if the line begins with the marker END OF PAGE xx, then we need to save the current page text and
		// reset the string builder
		if strings.HasPrefix(line, "END OF PAGE ") {
			// if we have any text in the current page, then we need to save it
			if err := p.createText(filepath, int64(currentPageNo), currentPageText.String()); err != nil {
				return err
			}
			p.log(fmt.Sprintf("page %d indexed", currentPageNo))
			currentPageText.Reset()
			continue
		}

		// otherwise, we trim the line and add it to the current page text if it's not empty
		line = strings.TrimSpace(line)
		if line != "" {
			currentPageText.WriteString(line)
			currentPageText.WriteString("\n")
		}
	}

	return nil
}

// createText is a helper function to create a new text entry in the database.
func (p *PdfSearcher) createText(filepath string, page int64, text string) error {
	_, err := p.queries.CreateText(context.Background(), db.CreateTextParams{
		Filepath:          filepath,
		SearchableContent: transformText(text),
		OriginalContent:   text,
		Page:              page,
	})
	return err
}

func (p *PdfSearcher) Search(ctx context.Context, query string, filename string) ([]db.PdfText, error) {
	p.log("searching for: " + query + " in " + filename)

	// the query needs to be transformed to match the transformed text in the database,
	// and we need to add % to the beginning and end of the query to make it a wildcard search
	actualQuery := "%" + transformText(query) + "%"

	results, err := p.queries.SearchTextsByFile(ctx, filename, actualQuery)
	if err != nil {
		return nil, err
	}

	p.log(fmt.Sprintf("found %d results", len(results)))

	for _, r := range results {
		out := fmt.Sprintf("%s pp %d: %s", r.Filepath, r.Page, r.OriginalContent)
		p.log(out)
	}

	return results, nil
}

// transformText will take a string, and keep only all letters a-z and numbers 0-9, converting all letters to lowercase,
// and it will remove all other characters, including spaces. It will then return the transformed string.
// For example, "The quick brown fox jumps over the lazy dog" will be transformed to "thequickbrownfoxjumpsoverthelazydog"
// This will be used to transform the text in the pdfs to make it easier to search for text.
func transformText(text string) string {
	// we first need to normalize the text. This will convert all accented characters to their base character
	// for example, "á" will be converted to "a".
	text = norm.NFKD.String(text)

	var result strings.Builder
	for _, char := range text {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			result.WriteRune(unicode.ToLower(char)) // Convert letters to lowercase and add them
		}
		// Ignore all other characters
	}
	return result.String()
}
