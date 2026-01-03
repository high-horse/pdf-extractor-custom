package main

import (
	// "errors"
	"bytes"
	"fmt"
	"image/jpeg"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	// "io/fs"
	"os"
	"path/filepath"

	"github.com/ledongthuc/pdf"
	"github.com/unidoc/unipdf/v4/extractor"
	"github.com/unidoc/unipdf/v4/model"
)

// Extracts images and names them using the 8-digit ID number found on the same page
func extractImagesWithIDNames_v1_more(inputPath, outputDir string) error {
	startTime := time.Now()
	pdfBase := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	pdfDir := filepath.Join(outputDir, pdfBase)
	if err := os.MkdirAll(pdfDir, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}

	pdfReader, f, err := model.NewPdfReaderFromFile(inputPath, nil)
	if err != nil {
		return err
	}
	defer f.Close()

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return err
	}

	log.Printf("Processing %d page(s)\n", numPages)

	totalExtracted := 0

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		if pageNum == 1 {
			continue
		}
		log.Printf("\n--- File %s  Page %d ---\n", inputPath, pageNum)
		fmt.Printf("\n--- File %s  Page %d ---\n", inputPath, pageNum)

		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			return err
		}

		text, err := extractTextFromPage(inputPath, pageNum)
		if err != nil {
			log.Printf("Warning: could not extract text from page %d: %v", pageNum, err)
			return fmt.Errorf("ERROR: Could not extract text from page %d of file %v\n", pageNum, inputPath)
		}
		voterIDs := extractVoterIDs(text)

		log.Printf("\n %s \n Found %d candidate ID(s) on page %d: %v\n",inputPath, len(voterIDs), pageNum, voterIDs)
		ids := append([]string{"logo1", "logo2", "logo3"}, voterIDs...)

		// Extract images from the same page
		imgExtractor, err := extractor.New(page)
		if err != nil {
			return err
		}
		pageImages, err := imgExtractor.ExtractPageImages(nil)
		if err != nil {
			return err
		}

		imgCount := len(pageImages.Images)
		log.Printf("Found %d image(s) on page %d \n", imgCount, pageNum)

		if imgCount == 0 {
			continue
		}
		// Use available IDs in order; fallback to sequential naming if not enough
		for i, img := range pageImages.Images {
			if i <= 2 {
				continue
			}
			gimg, err := img.Image.ToGoImage()
			if err != nil {
				return err
			}

			var filename string
			filename = ids[i] + ".jpg"

			// fullPath := filepath.Join(outputDir, filename)
			fullPath := filepath.Join(pdfDir, filename)

			outFile, err := os.Create(fullPath)
			if err != nil {
				return err
			}

			err = jpeg.Encode(outFile, gimg, &jpeg.Options{Quality: 90})
			outFile.Close()
			if err != nil {
				return err
			}

			log.Printf("Saved image : page Number %d file %s  saved as %s\n",  pageNum, inputPath ,filename)
			totalExtracted++
		}
	}
	log.Printf("completed for file %s om time %v seconds \n", inputPath, time.Since(startTime).Seconds())
	fmt.Printf("\nDone! Extracted %d image(s) to %s\n", totalExtracted, outputDir)
	return nil
}

func extractVoterIDs(extractedText string) []string {
	// Clean the text first
	cleaned := strings.ReplaceAll(extractedText, "\uFFFD", "")

	// Remove any garbage patterns like "12345.-" -> "12345"
	garbageRegex := regexp.MustCompile(`(\d+)[.\-]+`)
	cleaned = garbageRegex.ReplaceAllString(cleaned, "$1")

	// Split text into lines for better analysis
	lines := strings.Split(cleaned, "\n")

	var ids []string

	// Pattern to match serial numbers (क.सं. followed by digits)
	serialNumberPattern := regexp.MustCompile(`क\.सं\.\s*(\d+)`)

	// Pattern to match any sequence of 4-10 digits
	numberPattern := regexp.MustCompile(`\b(\d{4,10})\b`)

	// Track serial numbers to exclude them
	serialNumbers := make(map[string]bool)

	// First pass: identify all serial numbers (क.सं.)
	for _, line := range lines {
		matches := serialNumberPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				serialNumbers[match[1]] = true
			}
		}
	}

	log.Printf("Serial numbers to exclude: %v\n", serialNumbers)

	// Second pass: extract all numbers, excluding serial numbers
	for _, line := range lines {
		// Skip lines that contain "क.सं." entirely (these are just serial number lines)
		if strings.Contains(line, "क.सं.") || strings.Contains(line, "क.स.") {
			continue
		}

		matches := numberPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				id := match[1]

				// Skip if it's a known serial number
				if serialNumbers[id] {
					log.Printf("Skipping serial number: %s\n", id)
					continue
				}

				// Skip year-like patterns (1900-2099)
				if len(id) == 4 {
					year, _ := strconv.Atoi(id)
					if year >= 1900 && year <= 2099 {
						log.Printf("Skipping year-like number: %s\n", id)
						continue
					}
				}

				ids = append(ids, id)
			}
		}
	}

	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	uniqueIDs := []string{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	// log.Printf("Extracted voter IDs (after filtering): %v\n", uniqueIDs)

	return uniqueIDs
}

func extractVoterIDs_static(extractedText string) []string {
	var voterIDRegex = regexp.MustCompile(`\b(\d{5,10})\b`)

	cleaned := strings.ReplaceAll(extractedText, "\uFFFD", "")
	garbageRegex := regexp.MustCompile(`(\d{5,10})[.-]+`)
	cleaned = garbageRegex.ReplaceAllString(cleaned, "$1") // keep only the digits

	// Now extract the clean voter IDs
	matches := voterIDRegex.FindAllStringSubmatch(cleaned, -1)

	var ids []string
	for _, m := range matches {
		if len(m) > 1 {
			ids = append(ids, m[1])
		}
	}
	return ids
}

// Extract text from PDF using ledongthuc/pdf package
func extractTextFromPage(inputPath string, pageNum int) (string, error) {
	f, r, err := pdf.Open(inputPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	totalPages := r.NumPage()
	if pageNum > totalPages {
		return "", fmt.Errorf("page %d out of range (total: %d)", pageNum, totalPages)
	}

	p := r.Page(pageNum)
	if p.V.IsNull() {
		return "", nil
	}

	var buf bytes.Buffer
	text, err := p.GetPlainText(nil)
	if err != nil {
		return "", err
	}

	buf.WriteString(text)
	return buf.String(), nil
}
