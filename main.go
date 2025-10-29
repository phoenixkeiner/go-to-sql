package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type ColumnInfo struct {
	Name     string
	DataType string
	IsMoney  bool
	IsDate   bool
}

func main() {
	files, err := findExcelFiles(".")

	if err != nil {
		fmt.Printf("Error finding files: %v\n", err)
		return
	} else if len(files) == 0 {
		fmt.Println("No Excel files found in current directory")
		return
	}

	fmt.Println("Found Excel files:")
	for i, file := range files {
		fmt.Printf("%d. %s\n", i+1, file)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nEnter database name: ")
	dbName, _ := reader.ReadString('\n')
	dbName = strings.TrimSpace(dbName)

	fmt.Print("\nProcess all these files? (y/n): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "y" && confirm != "yes" {
		fmt.Println("Operation cancelled")
		return
	}

	for _, file := range files {
		defaultTableName := cleanTableName(file)
		fmt.Printf("\nProcessing: %s\n", file)
		fmt.Printf("Enter table name (press Enter to use '%s'): ", defaultTableName)
		tableName, _ := reader.ReadString('\n')
		tableName = strings.TrimSpace(tableName)

		if tableName == "" {
			tableName = defaultTableName
		} else {
			tableName = cleanColumnName(tableName)
		}

		err := processFile(file, dbName, tableName)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", file, err)
		}
	}
}

func findExcelFiles(dir string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".xlsx" || ext == ".xls" || ext == ".csv" {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

func processFile(filename, dbName, tableName string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	if ext == ".csv" {
		return processCSV(filename, dbName, tableName)
	}
	return processExcel(filename, dbName, tableName)
}

func processCSV(filename, dbName, tableName string) error {
	file, err := os.Open(filename)

	if err != nil {
		return err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	headers, err := reader.Read()

	if err != nil {
		return err
	}

	for i := range headers {
		if strings.TrimSpace(headers[i]) == "" {
			headers[i] = fmt.Sprintf("column_%d", i+1)
		} else {
			headers[i] = cleanColumnName(headers[i])
		}
	}

	var rows [][]string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}

	columns := analyzeColumns(headers, rows)
	generateSQL(tableName, dbName, columns, rows)

	return nil
}

func processExcel(filename, dbName, tableName string) error {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets found")
	}

	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return fmt.Errorf("sheet is empty")
	}

	headers := rows[0]
	for i := range headers {
		if strings.TrimSpace(headers[i]) == "" {
			headers[i] = fmt.Sprintf("column_%d", i+1)
		} else {
			headers[i] = cleanColumnName(headers[i])
		}
	}

	dataRows := rows[1:]
	columns := analyzeColumns(headers, dataRows)
	generateSQL(tableName, dbName, columns, dataRows)

	return nil
}

func cleanColumnName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "(", "")
	name = strings.ReplaceAll(name, ")", "")
	name = strings.ToLower(name)
	if name == "" {
		name = "column"
	}
	return name
}

func cleanTableName(filename string) string {
	name := filepath.Base(filename)
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)
	return cleanColumnName(name)
}

func isDate(val string) (bool, bool) {
	val = strings.TrimSpace(val)
	if val == "" {
		return false, false
	}

	dateFormats := []string{
		"2006-01-02",
		"01/02/2006",
		"01-02-2006",
		"02/01/2006",
		"02-01-2006",
		"2-Jan-06",
		"02-Jan-06",
		"2-Jan-2006",
		"02-Jan-2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"2 January 2006",
		"2006/01/02",
		"01.02.2006",
		"02.01.2006",
	}

	datetimeFormats := []string{
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		"02/01/2006 15:04:05",
		"2006-01-02 15:04",
		"01/02/2006 15:04",
		"02/01/2006 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}

	for _, format := range datetimeFormats {
		if _, err := time.Parse(format, val); err == nil {
			return true, true
		}
	}

	for _, format := range dateFormats {
		if _, err := time.Parse(format, val); err == nil {
			return true, false
		}
	}

	datePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\d{1,2}[-/]\d{1,2}[-/]\d{2,4}$`),
		regexp.MustCompile(`^\d{4}[-/]\d{1,2}[-/]\d{1,2}$`),
		regexp.MustCompile(`^\d{1,2}[-\s][A-Za-z]{3}[-\s]\d{2,4}$`),
		regexp.MustCompile(`^[A-Za-z]{3}[-\s]\d{1,2}[-\s,]\d{2,4}$`),
	}

	for _, pattern := range datePatterns {
		if pattern.MatchString(val) {
			return true, false
		}
	}

	datetimePattern := regexp.MustCompile(`^\d{1,2}[-/]\d{1,2}[-/]\d{2,4}\s+\d{1,2}:\d{2}(:\d{2})?`)
	if datetimePattern.MatchString(val) {
		return true, true
	}

	return false, false
}

func analyzeColumns(headers []string, rows [][]string) []ColumnInfo {
	columns := make([]ColumnInfo, len(headers))

	for i, header := range headers {
		columns[i].Name = header
		columns[i].DataType = "TEXT"
		columns[i].IsMoney = isMoney(header)
		columns[i].IsDate = false

		hasInt := false
		hasFloat := false
		hasText := false
		dateCount := 0
		datetimeCount := 0
		totalNonEmpty := 0

		for _, row := range rows {
			if i >= len(row) {
				continue
			}

			val := strings.TrimSpace(row[i])
			if val == "" {
				continue
			}

			totalNonEmpty++

			isDateVal, isDatetimeVal := isDate(val)
			if isDateVal {
				dateCount++
				if isDatetimeVal {
					datetimeCount++
				}
				continue
			}

			if _, err := strconv.ParseFloat(val, 64); err == nil {
				if strings.Contains(val, ".") {
					hasFloat = true
				} else {
					hasInt = true
				}
			} else {
				hasText = true
			}
		}

		if totalNonEmpty > 0 && float64(dateCount)/float64(totalNonEmpty) > 0.8 {
			columns[i].IsDate = true
			if datetimeCount > dateCount/2 {
				columns[i].DataType = "DATETIME"
			} else {
				columns[i].DataType = "DATE"
			}
		} else if hasText {
			columns[i].DataType = "TEXT"
		} else if hasFloat || columns[i].IsMoney {
			if columns[i].IsMoney {
				columns[i].DataType = "DECIMAL(15,2)"
			} else {
				columns[i].DataType = "DECIMAL(20,8)"
			}
		} else if hasInt {
			columns[i].DataType = "INTEGER"
		}
	}

	return columns
}

func isMoney(colName string) bool {
	lower := strings.ToLower(colName)
	moneyKeywords := []string{"price", "cost", "amount", "fee", "total", "pay", "salary", "wage", "revenue", "dollar", "usd", "eur", "gbp"}

	for _, keyword := range moneyKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func formatDateValue(val string) string {
	val = strings.TrimSpace(val)

	dateFormats := []string{
		"2006-01-02",
		"01/02/2006",
		"01-02-2006",
		"02/01/2006",
		"02-01-2006",
		"2-Jan-06",
		"02-Jan-06",
		"2-Jan-2006",
		"02-Jan-2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"2 January 2006",
		"2006/01/02",
		"01.02.2006",
		"02.01.2006",
	}

	datetimeFormats := []string{
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		"02/01/2006 15:04:05",
		"2006-01-02 15:04",
		"01/02/2006 15:04",
		"02/01/2006 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}

	for _, format := range datetimeFormats {
		if t, err := time.Parse(format, val); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	for _, format := range dateFormats {
		if t, err := time.Parse(format, val); err == nil {
			return t.Format("2006-01-02")
		}
	}

	return val
}

func generateSQL(tableName string, dbName string, columns []ColumnInfo, rows [][]string) {
	outputFile := fmt.Sprintf("%s_%s.sql", tableName, dbName)
	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	defer writer.Flush()

	writer.WriteString(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;\n", dbName))
	writer.WriteString(fmt.Sprintf("USE %s;\n\n", dbName))
	writer.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", tableName))
	writer.WriteString("    id INTEGER PRIMARY KEY AUTO_INCREMENT,\n")

	for i, col := range columns {
		line := fmt.Sprintf("    %s %s", col.Name, col.DataType)
		if i < len(columns)-1 {
			line += ","
		}
		writer.WriteString(line + "\n")
	}
	writer.WriteString(");\n\n")

	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}

		writer.WriteString(fmt.Sprintf("INSERT INTO %s (", tableName))
		colNames := make([]string, len(columns))
		for i, col := range columns {
			colNames[i] = col.Name
		}
		writer.WriteString(strings.Join(colNames, ", "))
		writer.WriteString(") VALUES (")

		values := make([]string, len(columns))
		for i, col := range columns {
			var val string
			if i < len(row) {
				val = strings.TrimSpace(row[i])
			}

			if val == "" {
				values[i] = "NULL"
			} else if col.IsDate {
				formattedDate := formatDateValue(val)
				values[i] = fmt.Sprintf("'%s'", formattedDate)
			} else if col.DataType == "TEXT" {
				val = strings.ReplaceAll(val, "'", "''")
				values[i] = fmt.Sprintf("'%s'", val)
			} else {
				values[i] = val
			}
		}

		writer.WriteString(strings.Join(values, ", "))
		writer.WriteString(");\n")
	}

	fmt.Printf("SQL file generated: %s\n", outputFile)
}

func isEmptyRow(row []string) bool {
	for _, val := range row {
		if strings.TrimSpace(val) != "" {
			return false
		}
	}
	return true
}
