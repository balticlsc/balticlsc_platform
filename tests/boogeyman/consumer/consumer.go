package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)

func readFile(path string) error {
	start := time.Now()

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	since := time.Since(start)

	read := len(dat)

	fmt.Printf("read: %dB %dns %.2fGB %.2fs %.2fMB/s\n",
		read, since,
		float64(read)/1000000000, float64(since)/float64(time.Second),
		(float64(read)/1000000)/(float64(since)/float64(time.Second)),
	)

	return nil
}

func main() {
	path := os.Getenv("STORAGE_PATH")

	if path == "" {
		path = "."
	}

	fmt.Println("Boogeyman consumer running:")
	fmt.Println(" path: " + path)

	for {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			fmt.Println(err)
		}
		if len(files) > 0 {
			randomFileIndex := rand.Intn(len(files))
			randomFile := files[randomFileIndex]
			err := readFile(path + "/" + randomFile.Name())
			if err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println("waiting for files")
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}
