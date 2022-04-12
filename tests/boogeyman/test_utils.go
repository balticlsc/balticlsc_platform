package boogeyman

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func CheckError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func HandleCtrlC(boogeyman *Boogeyman) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		err := boogeyman.Cleanup()
		CheckError(err)
		os.Exit(0)
	}()
}
