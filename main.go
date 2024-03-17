package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/nxadm/tail"
	"github.com/urfave/cli/v2"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
	"syscall"
)

func main() {
	app := &cli.App{
		Name:                 "sslkeylogmerge",
		Usage:                "merge multiple sslkeylogs into one",
		Action:               mainFunc,
		EnableBashCompletion: true,
		Suggest:              true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "output",
				Usage:     "output `file`",
				Required:  true,
				Aliases:   []string{"o"},
				EnvVars:   []string{"SSLKEYLOGFILE"},
				TakesFile: true,
			},
			&cli.StringSliceFlag{
				Name:      "input",
				Usage:     "individual input `file`(s)",
				Aliases:   []string{"i"},
				Required:  false,
				TakesFile: true,
			},
			&cli.StringSliceFlag{
				Name:      "watch",
				Usage:     "watch `directory`(ies)",
				Required:  false,
				Hidden:    false,
				Aliases:   []string{"w"},
				TakesFile: true,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func mainFunc(ctx *cli.Context) error {
	// get notified of signals
	myCtx, _ := signal.NotifyContext(ctx.Context, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL)
	ctx.Context = myCtx

	wg := &sync.WaitGroup{}
	// open the output file
	outputFilePath := ctx.String("output")
	foutFile, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_SYNC, 0600)
	if err != nil {
		panic(err) // TODO handle this error
	}
	// make it thread-safe with a lock
	fout := &SyncWriter{
		mux:    sync.Mutex{},
		output: foutFile,
	}

	// make a list of the individual input files
	inFiles := ctx.StringSlice("input")

	// For each watch directory
	/// Set up fsnotify on the directory to watch for new files
	//// when a new file is found, start a goroutine to read it
	/// List all files in the directory, add them to the list of files
	watchDirs := ctx.StringSlice("watch")
	if len(watchDirs) > 0 {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			panic(err) // todo handle this error
		}
		for _, d := range watchDirs {
			files, _ := os.ReadDir(d)
			// assume that any existing files should be included
			for _, f := range files {
				if !f.IsDir() {
					fPath := path.Join(d, f.Name())
					inFiles = append(inFiles, fPath)
				}
			}

			go HandleWatcher(ctx, watcher, wg, fout)
			// Watch each of the directories
			err := watcher.Add(d)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}

	/// begin goroutines to read each input file line-by-line
	for _, inFilePath := range inFiles {
		wg.Add(1)
		go ReadFile(ctx, wg, inFilePath, fout)
	}

	wg.Wait()
	// stop all reader goroutines

	// write
	fout.mux.Lock()
	_ = foutFile.Close()

	return nil
}

func HandleWatcher(ctx *cli.Context, watcher *fsnotify.Watcher, wg *sync.WaitGroup, output *SyncWriter) {
	for {
		select {
		case event, okay := <-watcher.Events:
			{
				if okay {
					if event.Has(fsnotify.Create) {
						// TODO is event.Name an absolute path?
						fmt.Printf("New File: %s\n", event.Name)
						wg.Add(1)
						go ReadFile(ctx, wg, event.Name, output)
					}
				}
			}
		case err, okay := <-watcher.Errors:
			{
				if okay {
					fmt.Printf("Watcher Err: %s\n", err.Error())
				}
			}
		case <-ctx.Context.Done():
			{
				fmt.Println("No longer listening for fsnotify events")
				return
			}
		}
	}
}

func ReadFile(ctx *cli.Context, wg *sync.WaitGroup, inFilePath string, output *SyncWriter) {
	defer wg.Done()

	if filepath.Abs(inFilePath) == filepath.Abs(ctx.String("output")) {
		fmt.Println("Ignoring output file as input in order to avoid infinite loop")
		return
	}

	fin, err := tail.TailFile(inFilePath, tail.Config{Follow: true, ReOpen: true, CompleteLines: true})
	if err != nil {
		fmt.Printf("err tailing %s: %s\n", inFilePath, err.Error())
		return
	}

	for {
		select {
		case line := <-fin.Lines:
			{
				if line == nil {
					continue
				}
				if line.Err != nil {
					fmt.Printf("Error: %s\n", err.Error())
				}
				_, err = output.Write(append([]byte(line.Text), '\n'))
				if err != nil {
					fmt.Printf("Error writing: %s\n", err.Error())
					return
				}
			}
		case <-ctx.Context.Done():
			{
				fmt.Printf("Stopping Reader for %s\n", inFilePath)
				return
			}
		}
	}
}

type SyncWriter struct {
	mux    sync.Mutex
	output io.Writer
}

func (s *SyncWriter) Write(p []byte) (n int, err error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.output.Write(p)
}
