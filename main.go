// POC Prometheus Text Exporter
package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "embed"

	"github.com/gorilla/mux"
)

var (
	branch string
	commit string
	Debug  bool
	Path   string
)

type About struct {
	Branch  string
	Commit  string
	Tool    string
	Version string
}

var about = About{
	Branch:  branch,
	Commit:  commit,
	Tool:    "Text Exporter",
	Version: "0.2.0",
}

// Display the version and exit
func Version(b bool) {
	if b {
		if about.Commit != "" {
			// go build -ldflags="-X main.commit=$(git rev-parse --short HEAD) -X main.bbrnch=$(git branch | sed 's/ * //')"
			fmt.Printf("%s v%s (commit:%s branch:%s)\n", about.Tool, about.Version, about.Commit, about.Branch)
		} else {
			// go build
			fmt.Printf("%s v%s\n", about.Tool, about.Version)
		}

		os.Exit(0)
	}
}

// Initialize logging
func InitLogging(fileName string) error {
	fh, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return fmt.Errorf("%s: running without logging", err)
	}

	log.SetOutput(fh)
	if Debug {
		log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC | log.Ldate | log.Ldate)
	} else {
		log.SetFlags(log.Lmicroseconds | log.LUTC | log.Ldate | log.Ldate)
	}

	log.Println("BEGIN")
	log.Printf("%s v%s\n", about.Tool, about.Version)
	return nil
}

// Form an address for ListenAndServe
func SetAddress(bind string, port int) string {
	p := strconv.Itoa(port)

	if bind == "" {
		return ":" + p
	}

	return bind + ":" + p
}

//go:embed template.html
var tmplContent string

func root(w http.ResponseWriter, r *http.Request) {
	//log.Println("about **", about)
	//log.Println("tmpl **", tmpl)
	//tmpl = template.Must(template.Parse(tmplContent))
	tmpl, err := template.New("template").Parse(tmplContent)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, about)
	log.Printf("%s %s / %s\n", r.RemoteAddr, r.Method, r.Proto)
}

func metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	d, err := os.ReadDir(Path)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	for _, f := range d {
		if !f.IsDir() {
			fh, err := os.Open(Path + "/" + f.Name())
			if err != nil {
				log.Println(err)
				continue
			}

			fs, _ := f.Info()
			log.Printf("%s %s %d %s\n", r.RemoteAddr, r.Proto, fs.Size(), f.Name())
			io.Copy(w, fh)
			fh.Close()
		}
	}
}

func main() {
	_bind := flag.String("bind", "", "Identify the bind address")
	_log := flag.String("log", "local.log", "Identify the log file")
	_path := flag.String("path", "", "Specify the text export directory")
	_port := flag.Int("port", 9101, "Identify the server port")
	_version := flag.Bool("version", false, "Display the program version and exit")
	flag.Parse()

	Version(*_version)
	if *_path == "" {
		log.Println("export path (-path) left undefined. option required")
		os.Exit(1)
	}
	Path = *_path
	InitLogging(*_log)

	r := mux.NewRouter()
	r.HandleFunc("/", root).Methods("GET")
	r.HandleFunc("/metrics", metrics).Methods("GET")

	if err := http.ListenAndServe(SetAddress(*_bind, *_port), r); err != nil {
		log.Println(err)
	}
}
