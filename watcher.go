package main

import (
	"crypto/md5"
	"expvar"
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hiant/go-shutil"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/expvarhandler"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {

	path := flag.String("path", ".", "Watcher path")
	port := flag.String("port", "4000", "Listening port")

	flag.Parse()

	watcher, err := fsnotify.NewWatcher()
	check(err)
	defer watcher.Close()

	// Setup FS handler
	root := *path
	root += "/.website"
	os.RemoveAll(root)
	err = os.MkdirAll(root, os.ModePerm)
	check(err)

	fs := &fasthttp.FS{
		Root:               root,
		IndexNames:         []string{"index.html"},
		GenerateIndexPages: true,
		Compress:           false,
		AcceptByteRange:    false,
		CacheDuration:      60 * time.Second,
	}
	fsHandler := fs.NewRequestHandler()

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/stats":
			expvarhandler.ExpvarHandler(ctx)
		default:
			fsHandler(ctx)
			updateFSCounters(ctx)
		}
	}

	// Start HTTP server.
	addr := fmt.Sprintf("0.0.0.0:%s", *port)
	log.Printf("Starting HTTP server on %s", addr)

	options := &shutil.CopyTreeOptions{Symlinks: false,
		Ignore:                 nil,
		CopyFunction:           shutil.Copy,
		IgnoreDanglingSymlinks: false,
		OnlySubDir:             true}

	gitbookBuild(*path, options)

	go func() {
		if err := fasthttp.ListenAndServe(addr, requestHandler); err != nil {
			log.Fatalf("error in ListenAndServe: %s", err)
		}
	}()

	log.Printf("Serving files from directory [%s]", root)
	log.Printf("See stats at http://%s/stats", addr)
	pwd, _ := filepath.Abs(".")

	file2sum := map[string]string{}
	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				_, file := filepath.Split(event.Name)
				if file[0] != '.' && !strings.EqualFold(file, "_book") {
					if fileInfo, err := os.Stat(event.Name); err == nil {
						log.Println("Event:", event)
						if fileInfo.IsDir() {
							if event.Op&fsnotify.Create == fsnotify.Create {
								err = addWatcher(watcher, ".")
								check(err)
							}
						} else {
							sum := md5file(event.Name)
							oldSum, has := file2sum[event.Name]
							if !has || !strings.EqualFold(oldSum, sum) {
								file2sum[event.Name] = sum
								if strings.HasSuffix(event.Name, "SUMMARY.md") {
									gitbookInit(*path)
								}
								gitbookBuild(*path, options)
							} else {
								log.Println("Nothing is changed")
							}
						}
					}

				}
			case err := <-watcher.Errors:
				panic(err)
			}
		}
	}()

	err = watcher.Add(pwd)
	check(err)
	log.Printf("Directory [%v] was watched", pwd)
	err = addWatcher(watcher, ".")
	check(err)
	<-done
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
		panic(e)
	}
}

func addWatcher(watcher *fsnotify.Watcher, path string) error {
	dir, err := ioutil.ReadDir(path)
	check(err)
	for _, fd := range dir {
		if fd.IsDir() {
			subPath := fd.Name()
			prefix := subPath[0]
			if prefix == '.' || prefix == '_' || strings.HasPrefix(subPath, "node_modules") {
				continue
			}
			relativePath := filepath.Join(path, subPath)
			pwd, _ := filepath.Abs(relativePath)
			err = watcher.Add(relativePath)
			if err == nil {
				log.Printf("Directory [%v] was watched", pwd)
			}
			err = addWatcher(watcher, relativePath)
		}
	}
	return err
}

func gitbookInit(path string) {
	build := exec.Command("gitbook", "init")
	build.Dir = path
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	err := build.Run()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	log.Printf("Gitbook init success")
}

func gitbookBuild(path string, options *shutil.CopyTreeOptions) {
	build := exec.Command("gitbook", "build")
	build.Dir = path
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	err := build.Run()
	if err != nil {
		log.Fatal(err)
		panic(err)
	} else {
		err = shutil.CopyTree("_book", ".website", options)
		check(err)
		log.Printf("Gitbook rebuild success")
	}
}

func md5file(path string) string {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	return fmt.Sprint("%x", md5.Sum(content))
}

func updateFSCounters(ctx *fasthttp.RequestCtx) {
	// Increment the number of fsHandler calls.
	fsCalls.Add(1)

	// Update other stats counters
	resp := &ctx.Response
	switch resp.StatusCode() {
	case fasthttp.StatusOK:
		fsOKResponses.Add(1)
		fsResponseBodyBytes.Add(int64(resp.Header.ContentLength()))
	case fasthttp.StatusNotModified:
		fsNotModifiedResponses.Add(1)
	case fasthttp.StatusNotFound:
		fsNotFoundResponses.Add(1)
	default:
		fsOtherResponses.Add(1)
	}
}

// Various counters - see https://golang.org/pkg/expvar/ for details.
var (
	// Counter for total number of fs calls
	fsCalls = expvar.NewInt("fsCalls")

	// Counters for various response status codes
	fsOKResponses          = expvar.NewInt("fsOKResponses")
	fsNotModifiedResponses = expvar.NewInt("fsNotModifiedResponses")
	fsNotFoundResponses    = expvar.NewInt("fsNotFoundResponses")
	fsOtherResponses       = expvar.NewInt("fsOtherResponses")

	// Total size in bytes for OK response bodies served.
	fsResponseBodyBytes = expvar.NewInt("fsResponseBodyBytes")
)
