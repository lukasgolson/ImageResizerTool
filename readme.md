# Image Resizer Tool

The **Image Resizer Tool** is a fast and efficient command-line application written in Go. It enables users to resize images while managing memory usage, ensuring compatibility with the Windows GDI+ subsystem used by many legacy applications. This tool offers flexible resizing methods, efficient batch processing, and a preview mode to test operations before saving changes.

---

## Background

GDI+ is a graphics subsystem introduced with **Windows XP** in 2001, succeeding the original Graphics Device Interface (GDI). It provides tools for rendering images, drawing graphics, and displaying formatted text. While modern frameworks like Windows Presentation Foundation (WPF) have largely replaced GDI+ for new applications, GDI+ remains crucial for legacy software and certain Windows components.

When GDI+ processes an image, it expands compressed files (e.g., PNG, JPEG) into uncompressed **bitmaps**. A bitmap is a multidimensional array of pixels, where each pixel contains data for **red, green, blue (RGB)** and sometimes an **alpha channel** (transparency). This structure requires significant memory:

1. **Contiguous Memory Allocation**: GDI+ requires memory for bitmaps to be allocated as a single, continuous block.
2. **Dynamic Allocation**: Image sizes and formats vary, making memory requirements unpredictable at compile time.

This exceeds the 2 GB heap limit on 32-bit systems and the .net platform, resulting in errors such as **"Parameter is not valid"** and **"bufferOverflow"** exceptions.

This tool addresses these limitations by resizing images to fit within a specified memory limit when uncompressed, ensuring compatibility with GDI+ and other memory-constrained environments.
---

## Features

- **Memory-Constrained Resizing**: Ensures resized images remain within a specified memory limit.
- **Customizable Resizing Algorithms**: Choose from high-quality Lanczos3, Bilinear, or NearestNeighbor methods.
- **JPEG Quality Control**: Adjust JPEG compression quality (1-100).
- **Batch Processing**: Handle large numbers of images, including recursive processing of subdirectories.
- **Dry-Run Capability**: Preview resizing operations without saving output files.
- **Progress Tracking**: Monitor progress with a built-in progress bar.
- **Custom Output Directories**: Specify where resized images should be saved.
- **Duplicate Handling**: Skip files that already have resized versions.

---

## Installation

### Prerequisites

- [Go](https://golang.org/doc/install) version 1.16 or newer.

### Build Instructions

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd <repository-folder>
   ```
2. Build the executable:
   ```bash
   go build -o resizer
   ```
3. The resulting `resizer` file is your executable for resizing images.

---

## Usage

### Basic Syntax

```bash
resizer --memory <bytes> [options] <file or directory>
```

### Command-Line Options

| Option        | Shortcut | Description                                          | Default                   |
| ------------- | -------- | ---------------------------------------------------- | ------------------------- |
| `--memory`    | `-m`     | Maximum memory limit for resized images in bytes     | `2GB` (2 × 1024^3)        |
| `--output`    | `-o`     | Directory to save resized images                     | Current working directory |
| `--algorithm` | `-a`     | Resizing method: `lanczos`, `bilinear`, or `nearest` | `lanczos`                 |
| `--quality`   | `-q`     | JPEG quality (1 to 100)                              | `75`                      |
| `--dry-run`   |          | Simulate resizing without saving files               | Disabled                  |
| `--recursive` | `-r`     | Recursively process directories                      | Disabled                  |

### Examples

#### Resize a Single Image

```bash
resizer --memory 104857600 image.jpg
```

#### Save Resized Images to a Specific Directory

```bash
resizer --memory 104857600 --output /path/to/output image.jpg
```

#### Resize All Images in a Folder

```bash
resizer --memory 104857600 /path/to/images
```

#### Use Bilinear Algorithm with High JPEG Quality

```bash
resizer --memory 104857600 --algorithm bilinear --quality 90 image.jpg
```

#### Perform a Dry Run

```bash
resizer --dry-run --memory 104857600 /path/to/images
```

#### Recursively Process a Directory

```bash
resizer --memory 104857600 --recursive /path/to/images
```

---

## Supported Formats

- **Input**: `.jpg`, `.jpeg`, `.png`
- **Output**: `.jpg`, `.jpeg`, `.png`

---

## Error Handling

- **Unsupported Formats**: Skips files not in `.jpg`, `.jpeg`, or `.png` formats.
- **File Access Errors**: Logs issues if files or folders cannot be accessed.
- **Existing Files**: Avoids processing files that already have resized versions.

---

## License

This project is licensed under the MIT License - see the [LICENSE](license.txt) file for details.
