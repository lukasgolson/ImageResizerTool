package main

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
	"github.com/urfave/cli/v2"
)

type PixelFormat int

const (
	Format8bppIndexed PixelFormat = iota
	Format24bppRgb
	Format32bppArgb
)

func calculateMaxResolution(originalWidth, originalHeight int, pixelFormat PixelFormat, alignment int, memoryLimit int64) (int, int) {
	bytesPerPixel := getBytesPerPixel(pixelFormat)
	aspectRatio := float64(originalWidth) / float64(originalHeight)
	estimatedHeight := math.Sqrt(float64(memoryLimit) / (float64(bytesPerPixel) * aspectRatio))

	for {
		height := int(math.Floor(estimatedHeight))
		width := int(math.Floor(aspectRatio * float64(height)))
		stride := (width*bytesPerPixel + alignment - 1) / alignment * alignment
		totalMemory := int64(stride) * int64(height)

		if totalMemory <= memoryLimit {
			return width, height
		}

		estimatedHeight = math.Sqrt(float64(memoryLimit) / (float64(stride) * float64(bytesPerPixel)))
	}
}

func getBytesPerPixel(pixelFormat PixelFormat) int {
	switch pixelFormat {
	case Format8bppIndexed:
		return 1
	case Format24bppRgb, Format32bppArgb:
		return 4
	default:
		panic(fmt.Sprintf("Unsupported PixelFormat: %v", pixelFormat))
	}
}

func getPixelFormat(fileExt string) PixelFormat {
	switch strings.ToLower(fileExt) {
	case ".png":
		return Format32bppArgb
	case ".jpg", ".jpeg":
		return Format24bppRgb
	default:
		panic("Unsupported file format")
	}
}

func resizeImage(filePath, outputPath string, memoryLimit int64) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	originalWidth, originalHeight := img.Bounds().Dx(), img.Bounds().Dy()
	pixelFormat := getPixelFormat(filepath.Ext(filePath))
	newWidth, newHeight := calculateMaxResolution(originalWidth, originalHeight, pixelFormat, 4, memoryLimit)

	if newWidth < originalWidth || newHeight < originalHeight {
		resized := resize.Resize(uint(newWidth), uint(newHeight), img, resize.Lanczos3)
		if err := saveImage(resized, outputPath, format); err != nil {
			return err
		}
	}

	return nil
}

func saveImage(img image.Image, outputPath, format string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	switch format {
	case "png":
		return png.Encode(outFile, img)
	case "jpeg":
		return jpeg.Encode(outFile, img, nil)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func processFile(filePath string, memoryLimit int64, outputDir string) {
	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		fmt.Println("Error creating output directory:", err)
		return
	}

	// Generate the output file path in the specified directory
	outputFileName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)) + "-fixed" + filepath.Ext(filePath)
	outputPath := filepath.Join(outputDir, outputFileName)

	fmt.Println("Processing file:", filePath)
	if err := resizeImage(filePath, outputPath, memoryLimit); err != nil {
		fmt.Println("Error resizing image:", err)
	} else {
		fmt.Println("Image resized and saved as:", outputPath)
	}
}

func processPath(path string, memoryLimit int64, outputDir string) {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Println("Error accessing path:", err)
		return
	}

	if info.IsDir() {
		processDirectory(path, memoryLimit, outputDir)
	} else {
		// Single file processing with a simple progress bar
		bar := pb.StartNew(1)
		processSingleFile(path, memoryLimit, outputDir)
		bar.Increment()
		bar.Finish()
	}
}

func processDirectory(path string, memoryLimit int64, outputDir string) {
	files, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}

	// Filter valid image files
	imageFiles := []os.DirEntry{}
	for _, file := range files {
		if !file.IsDir() {
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if isValidImageExtension(ext) {
				imageFiles = append(imageFiles, file)
			}
		}
	}

	// Initialize the progress bar
	bar := pb.StartNew(len(imageFiles))

	// Process each file with progress bar
	for _, file := range imageFiles {
		processFile(filepath.Join(path, file.Name()), memoryLimit, outputDir)
		bar.Increment()
	}

	// Finish the progress bar
	bar.Finish()
}

func processSingleFile(path string, memoryLimit int64, outputDir string) {
	ext := strings.ToLower(filepath.Ext(path))
	if isValidImageExtension(ext) {
		processFile(path, memoryLimit, outputDir)
	} else {
		fmt.Println("Unsupported file type:", path)
	}
}

func isValidImageExtension(ext string) bool {
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png"
}

func main() {
	app := &cli.App{
		Name:  "Resizer",
		Usage: "Resize images to fit within a memory limit",
		Flags: []cli.Flag{
			&cli.Int64Flag{
				Name:    "memory",
				Aliases: []string{"m"},
				Usage:   "Maximum memory limit in bytes (default: 2GB)",
				Value:   2 * 1024 * 1024 * 1024, // Default to 2GB
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Directory to save resized images (default: current working directory)",
				Value:   ".", // Default to the current working directory
			},
		},
		Action: func(c *cli.Context) error {
			memoryLimit := c.Int64("memory")
			outputDir := c.String("output")

			if c.NArg() == 0 {
				return fmt.Errorf("no input files or directories provided")
			}

			for _, path := range c.Args().Slice() {
				processPath(path, memoryLimit, outputDir)
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("Error:", err)
	}
}
