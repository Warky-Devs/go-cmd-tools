package main

import (
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
	fgPath   = flag.String("path", "", " Path")
	fgURL    = flag.String("url", "", "URL")
	fgApiKey = flag.String("apikey", "", "Api Key")
)

func FixPath(str string) string {
	str = filepath.FromSlash(str)
	s := strings.ReplaceAll(str, "\\", "/")

	return filepath.Clean(s)
}

func main() {
	flag.Parse()
	fmt.Print("Starting path download \n")

	if *fgPath == "" {
		fmt.Printf("no path: %s\n\n", *fgPath)

		flag.PrintDefaults()
		return
	}

	if *fgURL == "" {
		fmt.Printf("no url: %s\n\n", *fgPath)

		flag.PrintDefaults()
		return
	}

	err := downloadFile(*fgURL, *fgPath)
	if err != nil {
		log.Fatalf("Unable to download file to %s: %v", *fgPath, err)
	}

}

func downloadFile(url string, outputPath string) error {
	// Perform the HTTP GET request to the URL
	// Create a new HTTP client
	spinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	spinner.Prefix = "Downloading... "
	spinner.Color("yellow", "bold")
	spinner.Start()
	defer func() {
		spinner.Stop()
	}()

	client := &http.Client{}

	// Create a new request with the provided URL
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if *fgApiKey != "" {
		request.Header.Set("x-api-key", *fgApiKey)
	}

	// Perform the HTTP GET request with the custom headers
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Check if the request was successful (status code 200)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status code %d (%v) \nURL=%s", response.StatusCode, response.Status, url)
	}

	// Create the output file
	file, err := os.Create(filepath.Clean(outputPath))
	if err != nil {
		return err
	}
	defer file.Close()
	totalSize := response.ContentLength

	pw := &progressWriter{
		totalSize: totalSize,
		spinner:   spinner,
	}

	// Create a multi-writer to update the spinner and write to the file simultaneously
	writer := io.MultiWriter(file, pw)

	// Copy the response body (file content) to the output file
	_, err = io.Copy(writer, response.Body)
	if err != nil {
		return err
	}

	fmt.Printf("\nFile downloaded successfully to %s\n", outputPath)
	return nil
}

type progressWriter struct {
	totalSize   int64
	writtenSize int64
	progress    int
	spinner     *spinner.Spinner
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.writtenSize += int64(n)
	newProgress := int((pw.writtenSize * 100) / pw.totalSize)
	if newProgress != pw.progress {
		pw.progress = newProgress
		pw.spinner.Suffix = fmt.Sprintf(" %d%%", pw.progress)
	}
	return n, nil
}
