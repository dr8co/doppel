package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

const chunkSize int = 4096 * 16

// A program to find duplicate files in specified directories
func main() {
	flag.Parse()
	directories := flag.Args()
	if len(directories) == 0 {
		directories = []string{"."}
	}

	// Find files of similar sizes in all dirs
	allFiles := make(map[int64][]string)

	erroredFiles := uint32(0)

	for _, dir := range directories {
		err := filepath.WalkDir(dir, func(path string, dirEnt fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if dirEnt.Type().IsRegular() {
				info, err := dirEnt.Info()
				if err != nil {
					log.Println(err)
					erroredFiles++
				}

				fileSize := info.Size()
				allFiles[fileSize] = append(allFiles[fileSize], path)
			}
			return nil
		})

		if err != nil {
			log.Println(err)
			continue
		}

	}

	// Hash map of file hashes and file names
	fileHashmap := make(map[string][]string)

	filesProcessed := uint32(0)

	for _, files := range allFiles {
		numFiles := len(files)
		filesProcessed += uint32(numFiles)
		h := sha256.New()

		if numFiles > 1 {
			for _, file := range files {
				f, err := os.Open(file)
				if err != nil {
					log.Println(err)
					erroredFiles++
					continue
				}
				data := make([]byte, chunkSize)
				count, err := f.Read(data)
				if err != nil {
					if err != io.EOF {
						_ = f.Close()
						log.Println(err)
						erroredFiles++
						continue
					}
				}
				_ = f.Close()
				h.Write(data[:count])
				hash := string(h.Sum(nil))
				fileHashmap[hash] = append(fileHashmap[hash], f.Name())
				h.Reset()
			}

		}
	}

	// Display all the duplicates
	var i, numDuplicates uint32 = 0, 0
	for _, files := range fileHashmap {
		if len(files) > 1 {
			i++
			fmt.Println("\nDuplicate files set", i, ":")
			for _, file := range files {
				fmt.Println(file)
				numDuplicates++
			}
		}
	}

	if numDuplicates > 0 {
		fmt.Println("\nDuplicate files found:", numDuplicates)
	} else {
		fmt.Println("Duplicates not found")
	}

	if filesProcessed > 0 {
		fmt.Println("Total files processed:", filesProcessed)
	}
	if erroredFiles > 0 {
		fmt.Println("Failed to process", erroredFiles, "file(s)")
	}
}
