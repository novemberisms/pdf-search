package pdfsearch

import (
	"context"
	"fmt"
	"testing"
)

func TestNewPdfSearcher(t *testing.T) {
	t.Run("creates a new pdf searcher", func(t *testing.T) {
		p, err := NewPdfSearcher(PdfSearcherOptions{
			DatabaseName: ":memory:",
			Log:          func(s string) { fmt.Println(s) },
		})
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()
	})
}

func TestPdfSearcher_GetIndexedFiles(t *testing.T) {
	t.Run("returns indexed files", func(t *testing.T) {
		p, err := NewPdfSearcher(PdfSearcherOptions{
			DatabaseName: ":memory:",
			Log:          func(s string) { fmt.Println(s) },
		})
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		files, err := p.GetIndexedFiles(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(files) != 0 {
			t.Errorf("got %d files, want 0", len(files))
		}

		err = p.createText("test.pdf", 1, "testing")
		if err != nil {
			t.Fatal(err)
		}
		err = p.createText("test.pdf", 2, "testing")
		if err != nil {
			t.Fatal(err)
		}

		files, err = p.GetIndexedFiles(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(files) != 1 {
			t.Errorf("got %d files, want 1", len(files))
		}

		// create another file
		err = p.createText("test2.pdf", 1, "testing")
		if err != nil {
			t.Fatal(err)
		}
		err = p.createText("test2.pdf", 2, "testing")
		if err != nil {
			t.Fatal(err)
		}

		files, err = p.GetIndexedFiles(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(files) != 2 {
			t.Errorf("got %d files, want 2", len(files))
		}

		// check that test.pdf is in the list
		found := false
		for _, f := range files {
			if f == "test.pdf" {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("test.pdf not found in indexed files")
		}
	})
}

func TestPdfSearcher_Search(t *testing.T) {
	t.Run("searches indexed files", func(t *testing.T) {
		p, err := NewPdfSearcher(PdfSearcherOptions{
			DatabaseName: ":memory:",
			Log:          func(s string) { fmt.Println(s) },
		})
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		var texts = []string{
			"All of the things that you say--,",
			"Deeper than roses (all my heart knows this),",
			"All that is left of our days--",
			"The dreams you've been keeping, the songs you've been singing,",
			"Año tras=añoÇ",
		}

		for i, text := range texts {
			err = p.createText("test.pdf", int64(i+1), text)
			if err != nil {
				t.Fatal(err)
			}
		}

		type searchTest struct {
			filename string
			query    string
			want     int
		}

		tests := []searchTest{
			{"test.pdf", "all", 3},
			{"test.pdf", "all of", 1},
			{"testa.pdf", "all", 0},
			{"test.pdf", "YOuVe", 1},
			{"test.pdf", "sanoc", 1},
		}

		for i, test := range tests {
			results, err := p.Search(context.Background(), test.query, test.filename)
			if err != nil {
				t.Fatal(err)
			}

			if len(results) != test.want {
				t.Errorf("test %d: got %d results, want %d", i, len(results), test.want)
			}
		}
	})
}
