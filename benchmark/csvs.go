package benchmark

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"
)

type RowComposer func([]interface{}) []string

// for now only supports list of enr so far
type CSV struct {
	file    string
	columns []string
	// exporter
	f *os.File
	// importer
	r *csv.Reader
}

func NewCsvExporter(file string, columns []string) (*CSV, error) {
	csvFile, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	cols := make([]string, len(columns))
	for _, col := range columns {
		cols = append(cols, col)
	}
	return &CSV{
		file:    file,
		columns: cols,
		f:       csvFile,
	}, nil
}

func NewCsvImporter(file string) (*CSV, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	} // Close the file when done

	fileContent, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return &CSV{
		file: file,
		r:    csv.NewReader(strings.NewReader(string(fileContent))),
	}, nil
}

// --- Exporter ----
func (c *CSV) composeRow(items []string) string {
	newRow := ""
	for _, i := range items {
		if newRow == "" {
			newRow = i
			continue
		}
		newRow = newRow + "," + i
	}
	return newRow + "\n"
}

func (c *CSV) writeLine(row string) error {
	_, err := c.f.WriteString(row)
	return err
}

func (c *CSV) Export(rows [][]interface{}, rowComposer RowComposer) error {
	// reset index in the file
	c.f.Seek(0, 0)
	defer c.f.Sync()
	headerRow := c.composeRow(c.columns)
	err := c.writeLine(headerRow)
	if err != nil {
		return nil
	}
	for _, row := range rows {
		row := c.composeRow(rowComposer(row))
		err = c.writeLine(row)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (c *CSV) Close() error {
	return c.f.Close()
}

// --- Importer ---
func (c *CSV) items() ([][]string, error) {
	return c.r.ReadAll()
}

func (c *CSV) nextLine() ([]string, error) {
	return c.r.Read()
}

func (c *CSV) changeSeparator(sep rune) {
	c.r.Comma = sep
}

func (c *CSV) changeCommentChar(r rune) {
	c.r.Comment = r
}

// Basic String Row Composer
func StringRowComposer(rawRow []interface{}) []string {
	row := make([]string, len(rawRow), len(rawRow))
	for idx, item := range rawRow {
		switch item.(type) {
		case float64:
			row[idx] = fmt.Sprintf("%.6f", item.(float64))
		case int64:
			row[idx] = fmt.Sprintf("%d", item.(int64))
		case string:
			row[idx] = item.(string)
		case time.Duration:
			newItem := item.(time.Duration)
			row[idx] = fmt.Sprintf("%.6f", float64(newItem.Nanoseconds()))
		default:
			row[idx] = fmt.Sprint(item)
		}
	}
	return row
}
