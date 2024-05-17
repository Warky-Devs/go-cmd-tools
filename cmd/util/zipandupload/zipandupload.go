package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
)

var (
	fgPath    = flag.String("path", "", " Path")
	fgURL     = flag.String("url", "", "URL")
	fgZipFIle = flag.String("zipfile", "", "Zip the folder")
	fgApiKey  = flag.String("apikey", "", "Api Key")
)

func FixPath(str string) string {
	str = filepath.FromSlash(str)
	s := strings.ReplaceAll(str, "\\", "/")

	return filepath.Clean(s)
}

func main() {
	flag.Parse()
	fmt.Print("Starting path upload \n")
	var err error
	if *fgPath == "" {
		fmt.Printf("no path: %s\n\n", *fgPath)

		//flag.PrintDefaults()
		//return
	}

	if *fgZipFIle == "" {
		fmt.Printf("no zipfile name: %s\n\n", *fgZipFIle)

		flag.PrintDefaults()
		return
	}

	if *fgPath != "" {
		err = zipDirectoryToFile(*fgPath, *fgZipFIle)
		if err != nil {
			log.Fatalf("Unable to zip file: %v", err)
		}
	}

	if *fgURL == "" {
		fmt.Printf("no url: %s\n\n", *fgURL)

		//flag.PrintDefaults()
		//return
	} else {
		err = uploadFile(*fgURL, *fgZipFIle)
		if err != nil {
			log.Fatalf("Upload failed: %v", err)
		}
	}

}

func zipDirectoryToFile(dirPath string, outputFilePath string) error {
	dirPath = FixPath(dirPath)
	zipFile, err := os.Create(FixPath(outputFilePath))
	if err != nil {
		return err
	}
	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	spinner.Prefix = "Zipping... "
	spinner.Color("red", "bold")
	spinner.Start()
	defer func() {
		zipFile.Close()
		spinner.Stop()
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirPath, FixPath(filePath))
		if err != nil {
			return err
		}
		relPath = FixPath(relPath)

		if info.IsDir() {
			// Create a directory entry in the zip archive
			zipWriter.Create(relPath + "/")
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipFile, file)
		return err
	})

	return err
}

func zipDirectoryToBlob(dirPath string) ([]byte, error) {
	dirPath = FixPath(dirPath)
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	defer zipWriter.Close()
	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	spinner.Prefix = "Zipping... "
	spinner.Color("red", "bold")
	spinner.Start()
	defer func() {
		zipWriter.Close()
		spinner.Stop()
	}()

	total := 0
	err := filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirPath, filePath)
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Create a directory entry in the zip archive
			zipWriter.Create(relPath + "/")
			return nil
		}
		total++

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			return err
		}

		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}
		totalSize := fileInfo.Size()

		pr := &progressReader{
			totalSize: totalSize,
			spinner:   spinner,
			reader:    file,
			total:     total,
		}

		_, err = io.Copy(zipFile, pr)
		return err
	})

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func uploadFile(url string, filePath string) error {
	// Open the file
	file, err := os.Open(FixPath(filePath))
	if err != nil {
		return err
	}

	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	spinner.Prefix = "Uploading... "
	spinner.Color("green", "bold")
	spinner.Start()
	defer func() {
		file.Close()
		spinner.Stop()
	}()

	// Get the total size of the file
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	totalSize := fileInfo.Size()

	pr := &progressReader{
		totalSize: totalSize,
		spinner:   spinner,
		reader:    file,
	}

	// Create a POST request to the specified URL with the file content in the request body
	request, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return err
	}

	// Set the Content-Type header based on the file type or a generic application/octet-stream
	// Adjust this based on the specific requirements of the server you are interacting with.
	request.Header.Set("Content-Type", "application/octet-stream")
	if *fgApiKey != "" {
		request.Header.Set("x-api-key", *fgApiKey)
	}

	// Perform the request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Check the response status
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed with status code %d", response.StatusCode)
	}

	fmt.Println("\nFile uploaded successfully")
	return nil
}

type progressReader struct {
	totalSize int64
	readSize  int64
	progress  int

	total   int
	spinner *spinner.Spinner
	reader  io.Reader
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.readSize += int64(n)
		newProgress := int((pr.readSize * 100) / pr.totalSize)
		if newProgress != pr.progress {
			pr.progress = newProgress
			if pr.total > 0 {
				pr.spinner.Suffix = fmt.Sprintf(" %d%% (%d)", pr.progress, pr.total)
			} else {
				pr.spinner.Suffix = fmt.Sprintf(" %d%%", pr.progress)
			}

		}
	}
	return n, err
}
