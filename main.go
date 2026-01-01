package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"

	// "image/jpeg"
	"log"
	"os"

	// "path/filepath"

	// "sort"
	"time"

	"github.com/unidoc/unipdf/v4/common/license"
	// "github.com/unidoc/unipdf/v4/extractor"
	// "github.com/unidoc/unipdf/v4/model"
)

const (
	UNIDOC_LICENSE_API_KEY = "38311761653b7d673d05b475e0a29fbca0d87f42b623d9f7dd016f900dbdcdd4"
	// INPUT_FILE             = "input/sample.pdf"
	INPUT_FILE = "input/1_कोशी प्रदेश_4_झापा_5031_कमल गाउँपालिका.pdf"
	OUTPUT_DIR = "output/"

)

var (
	licenseKey = flag.String("license", UNIDOC_LICENSE_API_KEY, "UniDoc license key (or set UNIDOC_LICENSE_API_KEY env var)")
	outputDir  = flag.String("output", "output/", "Output directory.")
	inputFiles stringSlice
	
	idRegex = regexp.MustCompile(`\b[\p{Nd}]{8}\b`)
	
)

type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprint(*s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func initLicense() {
	key := *licenseKey
	if err := license.SetMeteredKey(key); err != nil {
		log.Fatalf("Failed to set license key: %v", err)
		panic(err)
	}
}

func main() {

	flag.Var(&inputFiles, "input", "Input PDF file (can be used multiple times)")

	flag.Parse()

	initLicense()
	validFiles := verifyInputFilesStrict(inputFiles)

	logFileName := fmt.Sprintf("%d_app.log", time.Now().Unix())
	fmt.Printf("created log file :%s\n", logFileName)
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println("starting ...")
	startTime := time.Now()

	wg := sync.WaitGroup{}
	for _, input := range validFiles {
		wg.Add(1)
		go func(input string) {
			defer wg.Done()
			if err := extractImagesWithIDNames_v1_more(input, *outputDir); err != nil {
				errStr := fmt.Sprintf("ERROR: error encountered in file %s : %v \n\n", input, err)
				log.Print(errStr)
				fmt.Print(errStr)
			}
		}(input)
	}
	wg.Wait()
	// err = extractImagesWithIDNames_v1_more(INPUT_FILE, OUTPUT_DIR)
	// if err != nil {
	// 	fmt.Printf("Error: %v\n", err)
	// 	os.Exit(1)
	// }

	endTime := time.Since(startTime)
	fmt.Printf("Completed batch %.2f seconds\n", endTime.Seconds())
}

/* ---------- strict file verification ---------- */

func verifyInputFilesStrict(files []string) []string {
	if len(files) == 0 {
		panic("No input files provided")
	}

	validFiles := make([]string, 0, len(files))

	for _, f := range files {
		abs, err := filepath.Abs(f)
		if err != nil {
			panic(fmt.Sprintf("Invalid file path: %s (%v)", f, err))
		}

		info, err := os.Stat(abs)
		if err != nil {
			panic(fmt.Sprintf("Input file does not exist: %s", abs))
		}

		if info.IsDir() {
			panic(fmt.Sprintf("Input path is a directory, not a file: %s", abs))
		}

		validFiles = append(validFiles, abs)
	}

	return validFiles
}
