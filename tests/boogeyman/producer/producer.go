package main

import (
	"bufio"
	"fmt"
	"github.com/rs/xid"
	"os"
	"strconv"
	"time"
)

func writeFile(fSize int64, path string) error {
	guid := xid.New()
	filename := guid.String()

	fName := path + "/" + filename
	f, err := os.Create(fName)
	if err != nil {
		return err
	}

	const defaultBufSize = 4096
	buf := make([]byte, defaultBufSize)
	buf[len(buf)-1] = '\n'
	w := bufio.NewWriterSize(f, len(buf))

	start := time.Now()
	written := int64(0)
	for i := int64(0); i < fSize; i += int64(len(buf)) {
		nn, err := w.Write(buf)
		written += int64(nn)
		if err != nil {
			return err
		}
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	err = f.Sync()
	if err != nil {
		return err
	}

	since := time.Since(start)

	err = f.Close()
	if err != nil {
		return err
	}

	fmt.Printf("written: %dB %dns %.2fGB %.2fs %.2fMB/s\n",
		written, since,
		float64(written)/1000000000, float64(since)/float64(time.Second),
		(float64(written)/1000000)/(float64(since)/float64(time.Second)),
	)

	return nil
}

func main() {
	path := os.Getenv("STORAGE_PATH")
	fileSizeStr := os.Getenv("FILE_SIZE")

	if path == "" {
		path = "."
	}

	if fileSizeStr == "" {
		fileSizeStr = "1"
	}

	fileSize, _ := strconv.Atoi(fileSizeStr)

	fmt.Println("Boogeyman producer running:")
	fmt.Println(" path: " + path)
	fmt.Println(" file_size: " + fileSizeStr + " GiB")

	fSize := int64(fileSize) * (1024 * 1024 * 1024)

	for {
		err := writeFile(fSize, path)
		if err != nil {
			fmt.Fprintln(os.Stderr, fSize, err)
		}
	}
}
