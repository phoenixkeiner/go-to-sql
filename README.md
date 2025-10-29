# Excel to SQL Converter

Converts XLSX, XLS and CSV files into MySQL-compatible SQL statements.

## Features

- Supports multiple file formats: XLSX, XLS, CSV
- Automatically detects column data types (TEXT, INTEGER, DECIMAL, DATE, DATETIME)
- Intelligent money column detection based on column names
- Comprehensive date and datetime format recognition
- Handles empty column headers by auto-generating column names
- Batch processing of multiple files
- Custom table naming with fallback to filename

## Requirements

- Go 1.16 or higher
- Required Go packages:
  - `github.com/xuri/excelize/v2`

## Installation

1. Clone the repository:
```bash
git clone https://github.com/phoenixkeiner/go-to-sql
cd go-to-sql
```

2. Install dependencies:
```bash
go mod init go-to-sql
go get github.com/xuri/excelize/v2
```

3. Build the application:
```bash
go build -o excel-to-sql
```

## Usage

1. Place your Excel or CSV files in the same directory as the executable

2. Run the program:
```bash
go run .
```
Or if built:
```bash
./excel-to-sql
```

3. Follow the prompts:
   - Enter a database name
   - Confirm processing all files
   - For each file, enter a custom table name or press Enter to use the filename

4. The program generates SQL files named `{tablename}_{database}.sql` for each processed file

## Data Type Detection

### Numeric Types
- **INTEGER**: Columns containing only whole numbers
- **DECIMAL(15,2)**: Money-related columns (detected by keywords: price, cost, amount, fee, total, pay, salary, wage, revenue, dollar, usd, eur, gbp)
- **DECIMAL(20,8)**: Columns with decimal numbers

### Date Types
Automatically recognizes various date formats:
- ISO format: `2018-01-29`
- US format: `01/29/2018`
- European format: `29/01/2018`
- Excel style: `29-Jan-18`, `29-Jan-2018`
- Text style: `Jan 29, 2018`, `January 29, 2018`
- Datetime formats with time components

### Text Type
- **TEXT**: Any column containing non-numeric, non-date data or mixed data types

### Empty Columns
Columns with empty headers are automatically named as `column_1`, `column_2`, etc.

## Output

Generated SQL files include:
- Database creation statement (IF NOT EXISTS)
- Table creation statement with auto-increment primary key (IF NOT EXISTS)
- INSERT statements for all data rows
- Proper escaping of special characters in text fields
- NULL handling for empty values
- Date/datetime values formatted for MySQL compatibility

## Example

Input Excel file `sales_data.xlsx` with columns:
```
Date          | Product      | Amount | Region
29-Jan-18     | Widget       | 150.00 | North
30-Jan-18     | Gadget       | 200.50 | South
```

Output SQL file `sales_data_mydb.sql`:
```sql
CREATE DATABASE IF NOT EXISTS mydb;
USE mydb;

CREATE TABLE IF NOT EXISTS sales_data (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    date DATE,
    product TEXT,
    amount DECIMAL(15,2),
    region TEXT
);

INSERT INTO sales_data (date, product, amount, region) VALUES ('2018-01-29', 'Widget', 150.00, 'North');
INSERT INTO sales_data (date, product, amount, region) VALUES ('2018-01-30', 'Gadget', 200.50, 'South');
```

## Column Name Cleaning

Column names are automatically cleaned:
- Spaces, hyphens, and periods converted to underscores
- Parentheses removed
- Converted to lowercase
- Empty names replaced with `column_N`

## Limitations

- Only processes the first sheet in Excel files
- Requires at least one header row
- Date detection requires 80% or more of non-empty values to be valid dates
- CSV files must be properly formatted with consistent delimiters