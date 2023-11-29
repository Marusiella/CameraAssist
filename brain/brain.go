package brain

import (
	"fmt"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

const SYSTEMVOLUMEINFORMATION = "System Volume Information"

var (
	Source      string
	Destination string
	Del         bool
	Bytes       bool
)

func RunCopy(cmd *cobra.Command, args []string) {
	Copy(Source, Destination, Del)
}

func Copy(source string, destination string, delete bool) {
	if source == "" || destination == "" {
		fmt.Println("Source and destination must be specified")
		return
	}
	fmt.Println("Copying from", source, "to", destination)
	images, err := getAllImages(source)
	if err != nil {
		fmt.Println(err)
		return
	}

	newFolder, err := getNewFolder(destination)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("New folder:", newFolder)
	err = os.Mkdir(newFolder, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	organized := Organize(images)
	RunCopyFiles(organized, newFolder, delete)

}

func ConvertSize(size float32) string {
	if size < 1024 {
		return fmt.Sprintf("%.2f B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", size/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", size/1024/1024)
	} else if size < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", size/1024/1024/1024)
	} else {
		return fmt.Sprintf("%.2f TB", size/1024/1024/1024/1024)
	}
}

func RunCopyFiles(images map[string][]string, destination string, delete bool) {
	fmt.Println("Copying to", destination)
	m := make(map[string]int)
	size := make(map[string]int)
	sum := 0
	for key, value := range images {
		m[key] = len(value)
		if Bytes {
			for _, file := range value {
				info, err := os.Stat(file)
				if err != nil {
					fmt.Println("Skipping", file)
				}
				sum += int(info.Size())
				size[key] += int(info.Size())
			}
		} else {
			sum += len(value)
		}
	}
	for key, value := range m {
		fmt.Println(key, value)
	}
	var bar *progressbar.ProgressBar
	if Bytes {
		fmt.Println("Total size:", ConvertSize(float32(sum)))
		bar = progressbar.DefaultBytes(int64(sum), "Copying")

	} else {
		fmt.Println("Total files:", sum)
		bar = progressbar.Default(int64(sum))

	}
	var wg sync.WaitGroup
	for key, value := range images {
		err := os.Mkdir(destination+"/"+key, 0777)
		if err != nil {
			fmt.Println(err)
		}
		for _, file := range value {
			wg.Add(1)
			go func(f string, k string, delete bool) {
				err := CopyFile(f, destination+"/"+k, bar)
				if err != nil {
					fmt.Println(err)
				}
				if !Bytes {
					err = bar.Add(1)
					if err != nil {
						fmt.Println(err)
					}
				}
				if delete {
					err = DeleteFile(f)
					if err != nil {
						fmt.Println(err)
					}
				}
				wg.Done()
			}(file, key, delete)
		}
	}
	wg.Wait()
}

func DeleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}

func CopyFile(source string, destination string, bar *progressbar.ProgressBar) error {
	// Open original file
	originalFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	// Create new file
	newFile, err := os.Create(destination + "/" + strings.Split(source, "/")[len(strings.Split(source, "/"))-1])
	if err != nil {
		return err
	}
	defer newFile.Close()

	// Copy the bytes to destination from source
	if Bytes {
		_, err = io.Copy(io.MultiWriter(newFile, bar), originalFile)
	} else {
		_, err = io.Copy(newFile, originalFile)
	}
	if err != nil {
		return err
	}
	//fmt.Println("Copied Bytes:", bytesWritten)

	return nil
}

func Organize(files []string) map[string][]string {
	m := make(map[string][]string)
	for _, file := range files {
		ext := strings.Split(file, ".")[1]
		m[ext] = append(m[ext], file)
	}
	return m
}

func getNewFolder(destination string) (string, error) {
	allFolders, err := os.ReadDir(destination)
	if err != nil {
		return "", err
	}
	var folders []string
	for _, folder := range allFolders {
		if folder.IsDir() {
			folders = append(folders, folder.Name())
		}
	}
	// conv to number and get max
	for i := 0; i < len(folders); i++ {
		for j := i + 1; j < len(folders); j++ {
			folderI, err := strconv.Atoi(folders[i])
			if err != nil {
				fmt.Println(err)
			}
			folderJ, err := strconv.Atoi(folders[j])
			if err != nil {
				fmt.Println(err)
			}
			if folderI > folderJ {
				temp := folders[i]
				folders[i] = folders[j]
				folders[j] = temp
			}
		}
	}
	if len(folders) == 0 {
		return destination + "/1", nil
	}
	lastFolder := folders[len(folders)-1]
	lastFolderNumber, err := strconv.Atoi(lastFolder)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return destination + "/" + strconv.Itoa(lastFolderNumber+1), nil
}

func getAllImages(source string) ([]string, error) {
	var images []string
	data, err := os.ReadDir(source)
	if err != nil {
		return images, err
	}
	for _, file := range data {
		if file.Name() != SYSTEMVOLUMEINFORMATION {
			if file.IsDir() {
				imagesDirs, err := os.ReadDir(source + "/" + file.Name())
				if err != nil {
					fmt.Println(err)
					return images, err
				}
				for _, imageDir := range imagesDirs {
					if !imageDir.IsDir() {
						images = append(images, source+"/"+file.Name()+"/"+imageDir.Name())
					} else {
						imagesDirs2, err := os.ReadDir(source + "/" + file.Name() + "/" + imageDir.Name())
						if err != nil {
							fmt.Println(err)
							return images, err
						}
						for _, imageDir2 := range imagesDirs2 {
							if !imageDir2.IsDir() && !strings.HasSuffix(imageDir2.Name(), "CTG") {
								images = append(images, source+"/"+file.Name()+"/"+imageDir.Name()+"/"+imageDir2.Name())
							}
						}
					}
				}
			}
		}
	}
	return images, nil
}
