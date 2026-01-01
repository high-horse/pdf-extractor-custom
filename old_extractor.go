package main

import (
	// "errors"
	"fmt"
	"image/jpeg"
	"log"
	"sort"

	// "io/fs"
	"os"
	"path/filepath"

	"github.com/unidoc/unipdf/v4/extractor"
	"github.com/unidoc/unipdf/v4/model"
)

// Extracts images and names them using the 8-digit ID number found on the same page
func extractImagesWithIDNames_v1(inputPath, outputDir string) error {
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
			continue
		}
		fmt.Printf("\n--- Page %d ---\n", pageNum)

		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			return err
		}

		// Extract text from the page to find IDs
		textExtractor, err := extractor.New(page)
		if err != nil {
			return err
		}
		text, err := textExtractor.ExtractText()
		if err != nil {
			fmt.Printf("Warning: Could not extract text from page %d: %v\n", pageNum, err)
			text = ""
		}
		log.Println("extracted %v", text)
		panic("panic")

		// Find all 8-digit numbers (citizen IDs)
		ids := idRegex.FindAllString(text, -1)
		fmt.Printf("Found %d candidate ID(s) on page: %v\n", len(ids), ids)

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
		fmt.Printf("Found %d image(s) on page\n", imgCount)

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
			if i < len(ids) && len(ids[i]) == 8 {
				filename = ids[i] + ".jpg" // e.g., 19326330.jpg
			} else {
				// Fallback naming
				filename = fmt.Sprintf("page%d_image%d.jpg", pageNum, i+1)
			}

			fullPath := filepath.Join(outputDir, filename)

			outFile, err := os.Create(fullPath)
			if err != nil {
				return err
			}

			err = jpeg.Encode(outFile, gimg, &jpeg.Options{Quality: 90})
			outFile.Close()
			if err != nil {
				return err
			}

			fmt.Printf("Saved: %s\n", filename)
			totalExtracted++
		}
	}

	fmt.Printf("\nDone! Extracted %d image(s) to %s\n", totalExtracted, outputDir)
	return nil
}

// Extracts images from a PDF and saves them as individual JPEG files in the output directory.
func extractImagesToFolder(inputPath, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}
	log.Println("Created dir")

	// Open PDF
	pdfReader, f, err := model.NewPdfReaderFromFile(inputPath, nil)
	log.Println("opening pdf")
	if err != nil {
		return err
	}
	defer f.Close()

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return err
	}

	fmt.Printf("PDF has %d pages\n", numPages)

	totalImages := 0

	for i := 0; i < numPages; i++ {
		fmt.Printf("-----\nProcessing Page %d:\n", i+1)

		page, err := pdfReader.GetPage(i + 1)
		if err != nil {
			return err
		}

		pextract, err := extractor.New(page)
		if err != nil {
			return err
		}

		pimages, err := pextract.ExtractPageImages(nil)
		if err != nil {
			return err
		}

		fmt.Printf("Found %d image(s) on this page\n", len(pimages.Images))

		for idx, img := range pimages.Images {
			// Convert to Go image
			gimg, err := img.Image.ToGoImage()
			if err != nil {
				return err
			}

			// Create unique filename: page_index (1-based)
			filename := fmt.Sprintf("page%d_image%d.jpg", i+1, idx+1)
			fullPath := filepath.Join(outputDir, filename)

			// Save as JPEG
			outFile, err := os.Create(fullPath)
			if err != nil {
				return err
			}

			opts := &jpeg.Options{Quality: 90} // Good quality
			err = jpeg.Encode(outFile, gimg, opts)
			outFile.Close()
			if err != nil {
				return err
			}

			fmt.Printf("Saved: %s (Position: X=%.2f Y=%.2f, Size: %.2fx%.2f)\n",
				filename, img.X, img.Y, img.Width, img.Height)
		}

		totalImages += len(pimages.Images)
	}

	fmt.Printf("Done! Extracted total %d image(s) to %s\n", totalImages, outputDir)
	return nil
}

type ImageWithPosition struct {
	Img   extractor.ImageMark
	Index int
}

func extractImagesWithIDNames_v2(inputPath, outputDir string) error {
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
		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			return err
		}

		// Try to extract text using simple method
		textExtractor, err := extractor.New(page)
		if err != nil {
			return err
		}

		text, err := textExtractor.ExtractText()
		if err != nil {
			log.Printf("Warning: Could not extract text from page %d: %v\n", pageNum, err)
			text = ""
		}

		// Find all IDs in the text
		ids := idRegex.FindAllString(text, -1)
		fmt.Printf("Found %d ID(s): %v\n", len(ids), ids)

		// If we found IDs, print a sample of the text
		if len(ids) > 0 {
			if len(text) > 200 {
				fmt.Printf("Sample text: %s...\n", text[:200])
			} else {
				fmt.Printf("Full text: %s\n", text)
			}
		}

		// Extract images
		imgExtractor, err := extractor.New(page)
		if err != nil {
			return err
		}

		pageImages, err := imgExtractor.ExtractPageImages(nil)
		if err != nil {
			return err
		}

		// Sort images by position (Y descending, then X ascending)
		// This matches the reading order: top to bottom, left to right
		var imagePositions []ImageWithPosition
		for i, img := range pageImages.Images {
			imagePositions = append(imagePositions, ImageWithPosition{
				Img:   img,
				Index: i,
			})
		}

		// Sort images: top to bottom (Y desc), left to right (X asc)
		sort.Slice(imagePositions, func(i, j int) bool {
			yDiff := imagePositions[j].Img.Y - imagePositions[i].Img.Y
			if yDiff > 20 || yDiff < -20 { // Different rows
				return imagePositions[j].Img.Y < imagePositions[i].Img.Y // Top to bottom
			}
			return imagePositions[i].Img.X < imagePositions[j].Img.X // Left to right
		})

		fmt.Printf("Found %d image(s)\n", len(imagePositions))

		// Match images with IDs based on sorted order
		for i, imgPos := range imagePositions {
			var filename string

			// Skip first 3 images (logos/headers based on your earlier comment)
			if i < 3 && len(imagePositions) > 40 {
				filename = fmt.Sprintf("page%d_header%d.jpg", pageNum, i)
			} else {
				adjustedIndex := i
				if len(imagePositions) > 40 {
					adjustedIndex = i - 3 // Skip the first 3 header images
				}

				if adjustedIndex >= 0 && adjustedIndex < len(ids) {
					filename = ids[adjustedIndex] + ".jpg"
				} else {
					filename = fmt.Sprintf("page%d_img%d.jpg", pageNum, i)
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

			fmt.Printf("✓ Saved: %s (original index: %d)\n", filename, imgPos.Index)
			totalExtracted++
		}
	}

	fmt.Printf("\n✓ Extracted %d images total\n", totalExtracted)
	return nil
}
