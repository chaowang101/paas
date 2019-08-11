package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chaowang101/paas/config"
	"github.com/chaowang101/paas/data"
	"github.com/chaowang101/paas/handler"
)

const (
	logFilePerm = 0644
	logFlags    = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC
)

// TODO: upgrade the logger with more sophisticated log level control, like INFO,DEBUG,VERBOSE
func initLog(logFilePath string) (res io.WriteCloser) {
	if len(logFilePath) != 0 {
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, logFilePerm)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Fail to open or create the log file %s\n", logFilePath)
			os.Exit(-1)
		}
		res = f
	} else {
		res = os.Stdout
	}

	log.SetOutput(res)
	log.SetFlags(logFlags)
	return res
}

func main() {
	configFile := flag.String("Config", "", "The path of the configuration file")
	flag.Parse()

	setting, err := config.Init(*configFile)
	if err != nil {
		log.Fatalf("Fail to load configuration file %s, err:%s\n", *configFile, err.Error())
	}

	logWriteCloser := initLog(setting.LogFilePath)
	defer logWriteCloser.Close()

	dataMgr, err := data.NewManager(setting.PasswdFilePath, setting.GroupFilePath)
	if err != nil {
		log.Fatalf("Fail to instantiate passwdMgr, err:%s\n", err.Error())
	}
	err = dataMgr.Start()
	if err != nil {
		log.Fatalf("Fail to start passwdMgr, err:%s\n", err.Error())
	}

	srv := &http.Server{
		Addr:         setting.ListenHost + ":" + setting.Port,
		WriteTimeout: time.Duration(setting.WriteTimeoutInSec) * time.Second,
		ReadTimeout:  time.Duration(setting.ReadTimeoutInSec) * time.Second,
		IdleTimeout:  time.Duration(setting.IdleTimeoutInSec) * time.Second,
		Handler:      handler.New(setting.RestDomain, dataMgr),
	}

	log.Println("Start listening")
	// Server starts in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listener err :%s \n", err.Error())
		}
	}()

	// handle terminating signal to gracefully shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// Block until signal arrives
	<-c

	log.Println("PaaS is exiting")

	// cleanup routines should be called here
	if err = srv.Shutdown(context.Background()); err != nil {
		log.Printf("server shutdown returns err:%s\n", err.Error())
	}
	dataMgr.Stop()
}
