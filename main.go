package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const redText = "31"
const redBackground = "41"
const blueText = "34"
const noStyle = "0"

type recordError struct {
	startIndex    int
	recordSection string
	record1Text   string
	record2Text   string
}

const logPath = "efw2.log"

var logFile *os.File

var validateAlphaNumeric bool

func main() {
	var err error
	logFile, err = os.OpenFile(logPath, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Fatal("Couldn't open log file")
	}
	defer logFile.Close()
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatal("Too few arguments. Provide two files to compare")
	}

	var files []string
	if len(args) > 2 {
		for _, arg := range args {
			if arg == "-a" {
				validateAlphaNumeric = true
			} else {
				files = append(files, arg)
			}
		}
	} else {
		files = args
	}

	file1, err := os.ReadFile(files[0])
	if err != nil {
		log.Fatal("Error opening file 1:", err)
	}

	file2, err := os.ReadFile(files[1])
	if err != nil {
		log.Fatal("Error opening file 2:", err)
	}

	file1String := strings.ReplaceAll(string(file1), "\r\n", "\n")
	file2String := strings.ReplaceAll(string(file2), "\r\n", "\n")
	file1Records := strings.Split(file1String, "\n")
	file2Records := strings.Split(file2String, "\n")

	compare(file1Records, file2Records)
}

func compare(file1Records, file2Records []string) {
	recordErrorCount := 0
	index2 := 0
	for index1, record := range file1Records {
		if len(record) != 512 {
			if index1+1 == len(file1Records) {
				break
			}
			writeError("File 1 line %d is the wrong length (%d bytes)", index1+1, len(record))
			continue
		}
		recordType := record[:2]
		if recordType == "RS" || recordType == "RV" {
			continue
		}
		var matchingRecord string
		matchingRecord, index2 = findMatchingRecord(recordType, index2, file2Records)
		if matchingRecord == "" {
			writeError("File 2 is missing %s record found in File 1, Line %d", recordType, index1+1)
			continue
		}

		var errors []recordError
		switch recordType {
		case "RA":
			writeInfo("Submitter records:\nFile 1:\n%s\nFile 2:\n%s", record, matchingRecord)
		case "RE":
			errors = compareRE(record, matchingRecord)
		case "RW":
			errors = compareRW(record, matchingRecord)
		case "RO":
			errors = compareRO(record, matchingRecord)
		case "RT":
			errors = compareRT(record, matchingRecord)
		case "RU":
			errors = compareRU(record, matchingRecord)
		case "RF":
			errors = compareRF(record, matchingRecord)
		}
		if len(errors) > 0 {
			recordErrorCount++
			writeError("%s Record in File 1, Line %d does not match record in File 2, Line %d", recordType, index1+1, index2)
			for _, error := range errors {
				writeRecordError(error)
			}
		}
	}
	writeInfo("%d/%d Records contain errors", recordErrorCount, len(file1Records))
}

func compareRE(record1, record2 string) []recordError {
	var errors []recordError
	errors = checkSection("Record Key", record1[:39], record2[:39], 0, errors)
	if validateAlphaNumeric {
		errors = checkSection("Employer Name", record1[39:96], record2[39:96], 0, errors)
		locationErrors := checkSection("Employer Address", record1[96:118], record2[96:118], 96, errors)
		if len(locationErrors) != 0 {
			errors = checkSection("Employer Address", record1[96:118], record2[118:140], 96, errors)
		}
		deliveryErrors := checkSection("Employer Address", record1[118:140], record2[118:140], 118, errors)
		if len(deliveryErrors) != 0 {
			errors = checkSection("Employer Address", record1[118:140], record2[96:118], 118, errors)
		}
		errors = checkSection("Employer Address", record1[140:173], record2[140:173], 140, errors)
		errors = checkSection("Employer Info", record1[173:], record2[173:], 173, errors)
	}
	return errors
}

func compareRW(record1, record2 string) []recordError {
	var errors []recordError
	errors = checkSection("Record Key", record1[:12], record2[:12], 0, errors)
	if validateAlphaNumeric {
		errors = checkSection("Employee Info", record1[12:65], record2[12:65], 0, errors)
		locationErrors := checkSection("Employee Address", record1[65:87], record2[65:87], 65, []recordError{})
		if len(locationErrors) != 0 {
			errors = checkSection("Employee Address", record1[65:87], record2[87:109], 65, errors)
		}
		deliveryErrors := checkSection("Employee Address", record1[87:109], record2[87:109], 87, []recordError{})
		if len(deliveryErrors) != 0 {
			errors = checkSection("Employee Address", record1[87:109], record2[65:87], 87, errors)
		}
		errors = checkSection("Employee Address", record1[109:138], record2[109:138], 109, errors)
		errors = checkSection("Employee Info", record1[142:187], record2[142:187], 142, errors)
	}
	errors = checkSection("Employee Amounts", record1[187:264], record2[187:264], 187, errors)
	errors = checkSection("Employee Amounts", record1[275:341], record2[275:341], 275, errors)
	errors = checkSection("Employee Amounts", record1[353:396], record2[353:396], 353, errors)
	errors = checkSection("Employee Amounts", record1[407:484], record2[407:484], 407, errors)
	errors = checkSection("Employee Indicators", record1[484:], record2[484:], 484, errors)
	return errors
}

func compareRO(record1, record2 string) []recordError {
	var errors []recordError
	errors = checkSection("Employee Amounts", record1[11:], record2[11:], 11, errors)
	return errors
}

func compareRT(record1, record2 string) []recordError {
	var errors []recordError
	errors = checkSection("Employer Count", record1[2:9], record2[2:9], 2, errors)
	errors = checkSection("Employer Totals", record1[9:114], record2[9:114], 9, errors)
	errors = checkSection("Employer Totals", record1[129:219], record2[129:219], 129, errors)
	errors = checkSection("Employer Totals", record1[234:], record2[234:], 234, errors)
	return errors
}

func compareRU(record1, record2 string) []recordError {
	var errors []recordError
	errors = checkSection("Employer Count", record1[2:9], record2[2:9], 2, errors)
	errors = checkSection("Employer Totals", record1[9:129], record2[9:129], 9, errors)
	errors = checkSection("Employer Totals", record1[144:204], record2[144:204], 144, errors)
	errors = checkSection("Employer Totals", record1[354:], record2[354:], 354, errors)
	return errors
}

func compareRF(record1, record2 string) []recordError {
	var errors []recordError
	errors = checkSection("Total Record Count", record1[7:16], record2[7:16], 7, errors)
	return errors
}

func checkSection(section, text1, text2 string, index int, errors []recordError) []recordError {
	if func() string {
		isASCII, hasLower := true, false
		for i := 0; i < len(text1); i++ {
			c := text1[i]
			if c >= utf8.RuneSelf {
				isASCII = false
				break
			}
			hasLower = hasLower || ('a' <= c && c <= 'z')
		}
		if isASCII {
			if !hasLower {
				return text1
			}
			var (
				b   strings.Builder
				pos int
			)
			b.Grow(len(text1))
			for i := 0; i < len(text1); i++ {
				c := text1[i]
				if 'a' <= c && c <= 'z' {
					c -= 'a' - 'A'
					if pos < i {
						b.WriteString(text1[pos:i])
					}
					b.WriteByte(c)
					pos = i + 1
				}
			}
			if pos < len(text1) {
				b.WriteString(text1[pos:])
			}
			return b.String()
		}
		return strings.Map(unicode.ToUpper, text1)
	}() == strings.ToUpper(text2) {
		return errors
	}
	return append(errors, recordError{
		startIndex:    index,
		recordSection: section,
		record1Text:   text1,
		record2Text:   text2,
	})
}

func findMatchingRecord(recordType string, startingIndex int, records []string) (string, int) {
	for i := startingIndex; startingIndex < len(records); i++ {
		record := records[i]
		if len(record) != 512 {
			if i+1 == len(records) {
				break
			}
			writeError("File 2 line %d is the wrong length (%d bytes)", i+1, len(record))
			continue
		}
		matchingRecordType := record[:2]
		if matchingRecordType == recordType {
			return record, i + 1
		}
		if matchingRecordType == "RS" || matchingRecordType == "RV" {
			continue
		}
		writeError("File 2 has an extra record on line %d", i+1)
	}
	return "", startingIndex
}

func writeInfo(message string, a ...any) {
	logFile.WriteString(fmt.Sprintf(message+"\n", a...))
	infoPrefix := style(blueText) + "[Info] " + style(noStyle)
	fmt.Printf(infoPrefix+message+"\n", a...)
}

func writeRecordError(error recordError) {
	text1 := error.record1Text
	var builder1 strings.Builder
	text2 := error.record2Text
	var builder2 strings.Builder
	indexLine := make([]rune, len(text1)+5)
	for i := range indexLine {
		indexLine[i] = '-'
	}
	isRed := false
	for i := 0; i < len(error.record1Text); i++ {
		if unicode.ToUpper(rune(text1[i])) == unicode.ToUpper(rune(text2[i])) {
			if isRed {
				builder1.WriteString(style(noStyle))
				builder2.WriteString(style(noStyle))
				isRed = false
			}
		} else if !isRed {
			index := "|" + strconv.Itoa(error.startIndex+i+1)
			for j, char := range index {
				if indexLine[i+j] != '-' {
					break
				}
				indexLine[i+j] = char
			}
			builder1.WriteString(style(redBackground))
			builder2.WriteString(style(redBackground))
			isRed = true
		}
		builder1.WriteByte(text1[i])
		builder2.WriteByte(text2[i])
	}
	builder1.WriteString(style(noStyle))
	builder2.WriteString(style(noStyle))
	fmt.Println(error.recordSection + ":")
	logFile.WriteString(error.recordSection + ":\n")
	fmt.Println(string(indexLine))
	logFile.WriteString(string(indexLine) + "\n")
	fmt.Println(builder1.String())
	logFile.WriteString(error.record1Text + "\n")
	fmt.Println(builder2.String())
	logFile.WriteString(error.record2Text + "\n")
}

func writeError(message string, a ...any) {
	logFile.WriteString(fmt.Sprintf(message+"\n", a...))
	errorPrefix := style(redText) + "[Error] " + style(noStyle)
	fmt.Printf(errorPrefix+message+"\n", a...)
}

func style(text string) string {
	if runtime.GOOS == "windows" {
		return ""
	}
	return "\033[" + text + "m"
}
