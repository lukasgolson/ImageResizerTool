package main

import (
	"bytes"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/inconshreveable/mousetrap"
	"github.com/rwcarlsen/goexif/exif"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nfnt/resize"
	"github.com/urfave/cli/v2"
)

type PixelFormat int

const (
	Format8bppIndexed PixelFormat = iota
	Format24bppRgb
	Format32bppArgb
)

func calculateMaxResolution(originalWidth, originalHeight int, pixelFormat PixelFormat, alignment int, memoryLimit int64, dpi int) (int, int) {
	bytesPerPixel := getBytesPerPixel(pixelFormat)
	aspectRatio := float64(originalWidth) / float64(originalHeight)
	estimatedHeight := math.Sqrt(float64(memoryLimit) / (float64(bytesPerPixel) * aspectRatio))

	for {
		height := int(math.Floor(estimatedHeight))
		width := int(math.Floor(aspectRatio * float64(height)))
		stride := (width*bytesPerPixel + alignment - 1) / alignment * alignment
		totalMemory := int64(stride) * int64(height)

		if totalMemory <= memoryLimit {
			newWidth := width - (width % dpi)
			newHeight := int(float64(newWidth) / aspectRatio)
			return newWidth, newHeight
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

func extractDPI(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	e, err := exif.Decode(file)
	if err != nil {
		// Return a more descriptive error
		return 0, fmt.Errorf("no EXIF data or corrupted EXIF data: %w", err)
	}

	xResolution, err := e.Get(exif.XResolution)
	if err != nil {
		return 0, fmt.Errorf("failed to get XResolution: %w", err)
	}
	yResolution, err := e.Get(exif.YResolution)
	if err != nil {
		return 0, fmt.Errorf("failed to get YResolution: %w", err)
	}

	xNum, xDen, err := xResolution.Rat2(0)
	if err != nil {
		return 0, fmt.Errorf("error reading XResolution: %w", err)
	}
	yNum, yDen, err := yResolution.Rat2(0)
	if err != nil {
		return 0, fmt.Errorf("error reading YResolution: %w", err)
	}

	x := float64(xNum) / float64(xDen)
	y := float64(yNum) / float64(yDen)

	if x == y {
		return int(x), nil
	}
	return int(x), nil
}

func resizeImage(filePath, outputPath string, dryRun bool, memoryLimit int64, algorithm resize.InterpolationFunction, quality, dpi int) error {
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
	newWidth, newHeight := calculateMaxResolution(originalWidth, originalHeight, pixelFormat, 4, memoryLimit, dpi)

	if newWidth < originalWidth || newHeight < originalHeight {
		resized := resize.Resize(uint(newWidth), uint(newHeight), img, algorithm)
		newDPI := int(float64(newWidth) / (float64(originalWidth) / float64(dpi)))
		fmt.Printf("Rescaled to %dx%d with DPI %d\n", newWidth, newHeight, newDPI)

		if !dryRun {
			if err := saveImage(resized, outputPath, format, quality, newDPI); err != nil {
				return err
			}
		}
	}

	return nil
}

func saveImage(img image.Image, outputPath, format string, quality, dpi int) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	switch format {
	case "png":
		err = png.Encode(outFile, img)
		if err != nil {
			return fmt.Errorf("failed to encode PNG: %w", err)
		}
		return nil

	case "jpeg":
		// Encode the image to a buffer
		buffer := new(bytes.Buffer)
		err = jpeg.Encode(buffer, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return fmt.Errorf("failed to encode JPEG: %w", err)
		}

		_, err = outFile.Write(buffer.Bytes())
		if err != nil {
			return fmt.Errorf("failed to write JPEG data: %w", err)
		}

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func main() {
	var args = os.Args[1:]
	if len(args) == 0 && mousetrap.StartedByExplorer() {
		fmt.Println("This application cannot be run by double-clicking it. Please run it from a console or drag your images onto the executable.")
		fmt.Println("Press Enter to exit...")
		_, err := fmt.Scanln()
		if err != nil {
			return
		}
		os.Exit(1)
	}

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
			&cli.StringFlag{
				Name:    "algorithm",
				Aliases: []string{"a"},
				Usage:   "Resize algorithm to use (lanczos, bilinear, nearest)",
				Value:   "lanczos",
			},
			&cli.IntFlag{
				Name:    "quality",
				Aliases: []string{"q"},
				Usage:   "JPEG quality (1-100)",
				Value:   75,
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Simulate resizing without saving files",
			},
			&cli.BoolFlag{
				Name:    "recursive",
				Aliases: []string{"r"},
				Usage:   "Process directories recursively",
			},
			&cli.IntFlag{
				Name:    "dpi",
				Aliases: []string{"d"},
				Usage:   "Set the DPI for the output image. If not set, it will be extracted from EXIF if available",
				Value:   0, // Default DPI is unset
			},
		},
		Action: func(c *cli.Context) error {
			memoryLimit := c.Int64("memory")
			outputDir := c.String("output")
			algorithm := getResizeAlgorithm(c.String("algorithm"))
			quality := c.Int("quality")
			dryRun := c.Bool("dry-run")
			recursive := c.Bool("recursive")
			dpi := c.Int("dpi")

			if c.NArg() == 0 {
				return fmt.Errorf("no input files or directories provided")
			}

			for _, path := range c.Args().Slice() {
				processPath(path, memoryLimit, outputDir, algorithm, quality, dryRun, recursive, dpi)
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("Error:", err)
	}
}

func processPath(path string, memoryLimit int64, outputDir string, algorithm resize.InterpolationFunction, quality int, dryRun, recursive bool, dpi int) {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Println("Error accessing path:", err)
		return
	}

	var files []string

	if info.IsDir() {
		files = collectFiles(path, recursive)
	} else {
		files = []string{path}
	}

	bar := pb.StartNew(len(files))

	var wg sync.WaitGroup

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		if isValidImageExtension(ext) {
			wg.Add(1)
			go func(file string) {
				defer wg.Done()
				processFile(file, memoryLimit, outputDir, algorithm, quality, dryRun, bar, dpi)
			}(file)
		}
	}

	wg.Wait()
	bar.Finish()
}

func processFile(filePath string, memoryLimit int64, outputDir string, algorithm resize.InterpolationFunction, quality int, dryRun bool, bar *pb.ProgressBar, overrideDPI int) {
	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		fmt.Printf("\nError creating output directory: %v\n", err)
		bar.Increment()
		return
	}

	// Generate the output file path in the specified directory
	outputFileName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)) + "-resized" + filepath.Ext(filePath)
	outputPath := filepath.Join(outputDir, outputFileName)

	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("\nSkipping existing file: %s\n", outputPath)

		bar.Increment()
		return
	}

	var dpi int
	if overrideDPI == 0 {
		if extractedDPI, err := extractDPI(filePath); err == nil {
			dpi = extractedDPI
			fmt.Printf("\nExtracted DPI for %s: %d\n", filePath, dpi)
		} else {
			fmt.Printf("\nFailed to extract DPI for %s: %v. Using default input DPI: 72\n", filePath, err)
			dpi = 72
		}
	} else {
		dpi = overrideDPI
	}

	fmt.Printf("\nProcessing %s\n", filePath)

	if err := resizeImage(filePath, outputPath, dryRun, memoryLimit, algorithm, quality, dpi); err != nil {
		fmt.Printf("\nError resizing image: %v\n", err)
	}

	bar.Increment()
}

func collectFiles(dir string, recursive bool) []string {
	var files []string

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if isValidImageExtension(ext) {
				files = append(files, path)
			}
		}

		if !recursive && d.IsDir() && path != dir {
			return filepath.SkipDir
		}
		return nil
	})

	return files
}

func isValidImageExtension(ext string) bool {
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png"
}

func getResizeAlgorithm(name string) resize.InterpolationFunction {
	switch strings.ToLower(name) {
	case "bilinear":
		return resize.Bilinear
	case "nearest":
		return resize.NearestNeighbor
	default:
		return resize.Lanczos3
	}
}
