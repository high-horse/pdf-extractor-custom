package main

import (
	"fmt"
	"image/jpeg"
	"path/filepath"
	"sort"
	"strings"

	// "image/jpeg"
	"log"
	"os"

	// "path/filepath"
	"regexp"
	// "sort"

	"github.com/unidoc/unipdf/v4/extractor"
	"github.com/unidoc/unipdf/v4/model"
	// "github.com/unidoc/unipdf/v4/extractor"
	// "github.com/unidoc/unipdf/v4/model"
)

// type ImageWithPosition struct {
// 	Img   extractor.ImageMark
// 	Index int
// }

// cleaner function
func extractVoterIDs_(text string) []string {
	var voterIDs []string

	// Split text by newlines to process line by line
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Look for 10-digit numbers at the beginning of lines
		// Use regex to find 10 consecutive digits
		re := regexp.MustCompile(`^\d{10}`)
		matches := re.FindStringSubmatch(line)

		if len(matches) > 0 {
			voterID := matches[0]
			// Validate it's a 10-digit number
			if len(voterID) == 10 {
				voterIDs = append(voterIDs, voterID)
			}
		}
	}

	return voterIDs
}

func extractVoterIDs_un(text string) []string {
	var voterIDs []string

	// Use regex to find all 10-digit numbers anywhere in the text
	re := regexp.MustCompile(`\b\d{10}\b`)
	matches := re.FindAllString(text, -1)

	for _, match := range matches {
		if len(match) == 10 {
			voterIDs = append(voterIDs, match)
		}
	}

	return voterIDs
}

func extractImagesWithIDNames(inputPath, outputDir string) error {
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

	fmt.Printf("Processing %d page(s)\n", numPages)
	totalExtracted := 0

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		if pageNum == 1 {
			continue // Skip header page
		}

		fmt.Printf("\n--- Page %d ---\n", pageNum)

		// Extract text using ledongthuc/pdf
		text, err := extractTextFromPage(inputPath, pageNum)
		if err != nil {
			log.Printf("Warning: Could not extract text from page %d: %v\n", pageNum, err)
			text = ""
		}
		// log.Println("extracted text ",text)
		// panic("a")

		// Find all IDs in the text
		ids := idRegex.FindAllString(text, -1)
		fmt.Printf("Found %d ID(s): %v\n", len(ids), ids)

		// Print sample text for debugging
		if len(text) > 0 {
			if len(text) > 300 {
				fmt.Printf("Sample text (first 300 chars): %s...\n", text[:300])
			} else {
				fmt.Printf("Full text: %s\n", text)
			}
		}

		// Extract images using unipdf
		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			return err
		}

		imgExtractor, err := extractor.New(page)
		if err != nil {
			return err
		}

		pageImages, err := imgExtractor.ExtractPageImages(nil)
		if err != nil {
			return err
		}

		// Sort images by position (Y descending for top-to-bottom, then X ascending for left-to-right)
		var imagePositions []ImageWithPosition
		for i, img := range pageImages.Images {
			imagePositions = append(imagePositions, ImageWithPosition{
				Img:   img,
				Index: i,
			})
		}

		// Sort: top to bottom, left to right
		sort.Slice(imagePositions, func(i, j int) bool {
			yDiff := imagePositions[j].Img.Y - imagePositions[i].Img.Y
			// If Y positions are significantly different (different rows)
			if yDiff > 20 || yDiff < -20 {
				return imagePositions[j].Img.Y < imagePositions[i].Img.Y // Top to bottom
			}
			// Same row, sort by X
			return imagePositions[i].Img.X < imagePositions[j].Img.X // Left to right
		})

		fmt.Printf("Found %d image(s)\n", len(imagePositions))

		// Match images with IDs
		for i, imgPos := range imagePositions {
			var filename string

			// Skip first 3 images if they're headers/logos (when we have 40+ images)
			skipCount := 0
			if len(imagePositions) > 40 {
				skipCount = 3
			}

			if i < skipCount {
				filename = fmt.Sprintf("page%d_header%d.jpg", pageNum, i)
			} else {
				adjustedIndex := i - skipCount

				if adjustedIndex < len(ids) {
					filename = ids[adjustedIndex] + ".jpg"
					fmt.Printf("Matching image %d -> ID %s\n", i, ids[adjustedIndex])
				} else {
					filename = fmt.Sprintf("page%d_img%d.jpg", pageNum, i)
					fmt.Printf("Warning: No ID available for image %d\n", i)
				}
			}

			// Save image
			gimg, err := imgPos.Img.Image.ToGoImage()
			if err != nil {
				log.Printf("Warning: Could not convert image %d: %v\n", imgPos.Index, err)
				continue
			}

			fullPath := filepath.Join(outputDir, filename)
			outFile, err := os.Create(fullPath)
			if err != nil {
				log.Printf("Warning: Could not create file %s: %v\n", filename, err)
				continue
			}

			err = jpeg.Encode(outFile, gimg, &jpeg.Options{Quality: 90})
			outFile.Close()

			if err != nil {
				log.Printf("Warning: Could not encode image %s: %v\n", filename, err)
				continue
			}

			fmt.Printf("✓ Saved: %s\n", filename)
			totalExtracted++
		}
	}

	fmt.Printf("\n✓ Extracted %d images total\n", totalExtracted)
	return nil
}
