package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"math"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Settings struct {
	Hosts     map[string]string `json:"hosts"`
	Storage   string            `json:"storage"`
	Ttl       float64           `json:"ttl"`
	Listen    string            `json:"listen"`
	Quality   int               `json:"quality"`
	NumColors int               `json:"colors"`
	Ssl       bool              `json:"ssl"`
	Cert      string            `json:"cert"`
	Key       string            `json:"key"`
}

var configFile string
var storageDir string
var config Settings

func main() {

	flag.StringVar(&configFile, "config", "/etc/proximity/config.json", "The config file location")
	flag.StringVar(&storageDir, "data", os.TempDir(), "The data file storage location")
	flag.Parse()

	err := loadConfig()
	if err != nil {
		log.Fatalf("[ERROR] %s", err.Error())
	}

	r := mux.NewRouter()
	r.HandleFunc("/{width:[0-9]+}/{path:.*}", HandleImg)
	r.HandleFunc("/{path:.*}", HandleImg)
	http.HandleFunc("/script", HandleScript)
	http.HandleFunc("/up", HandleUp)
	http.Handle("/", r)

	if config.Ssl {
		err = http.ListenAndServeTLS(config.Listen, config.Cert, config.Key, nil)
		if err != nil {
			log.Fatalf("[ERROR] %s", err.Error())
		}
	} else {
		err = http.ListenAndServe(config.Listen, nil)
		if err != nil {
			log.Fatalf("[ERROR] %s", err.Error())
		}
	}

}

func Warn(err error) {
	log.Printf("[WARN] %s", err.Error())
}

func Error(w http.ResponseWriter, err error, code int) {
	log.Printf("[ERROR %v] %s", code, err.Error())
	http.Error(w, err.Error(), code)
}

func getMime(fn string) string {
	ext := filepath.Ext(fn)
	return mime.TypeByExtension(ext)
}

func HandleUp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"up":true}`)
}

func HandleScript(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/javascript")
	server := r.FormValue("server")

	if server == "" {
		prefix := "http://"
		if r.TLS != nil {
			prefix = "https://"
		}
		server = prefix + r.Host
	}

	class := r.FormValue("c")
	if class == "" {
		class = "fluid"
	}

	script := `!function(t,e){for(var a=document.getElementsByClassName(e),s=0;s<a.length;s++){var r=a[s],n=r.getAttribute("data-src"),o=r.getAttribute("data-bg");n&&(r.src=["/",t,r.offsetWidth,n].join("/")),o&&(r.style.backgroundImage="url("+["/",t,r.offsetWidth,o].join("/")+")")}}("%s","%s");`
	fmt.Fprintf(w, script, server, class)
}

func HandleImg(w http.ResponseWriter, r *http.Request) {

	host, ok := config.Hosts[strings.Split(r.Host, ":")[0]]
	if !ok {
		http.Error(w, fmt.Sprintf("Invalid host %s", r.Host), 404)
		return
	}

	w.Header().Set("X-Forwarded-Host", host)
	vars := mux.Vars(r)

	// Check if original file exists and serve if it does.
	basename := path.Clean(strings.TrimSuffix(vars["path"], "/"))
	fn := config.Storage + host + "/orig/" + basename

	// Make sure is an image
	ftype := getMime(fn)
	if !strings.HasPrefix(ftype, "image/") || ftype == "image/vnd.microsoft.icon" {
		msg := fmt.Sprintf("Invalid file %s", strings.TrimPrefix(strings.TrimPrefix(fn, config.Storage), host))
		Error(w, errors.New(msg), 415)
		return
	}

	// Try to open original image, and on failure fetch it and reload.
	of, err := os.Open(fn)
	if err != nil {
		url := "http://" + host + "/" + basename
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		err, code := saveFile(url, fn, ip)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		HandleImg(w, r)
		return
	}
	defer of.Close()

	// Check for existence of scaled image in sizes we want.
	widthStr := "orig"
	width := getWidth(r)
	if width != 0 {
		widthStr = string(width)
	}

	// Ensure (probably) scaled directory exists
	scaledName := fmt.Sprintf("%s/%s/%v/%s", strings.TrimSuffix(config.Storage, "/"), host, widthStr, basename)
	scaledDir := path.Dir(scaledName)
	err = os.MkdirAll(scaledDir, 0755)
	if err != nil {
		Error(w, err, 500)
		return
	}

	// Now try to open/create file
	_, err = os.Open(scaledName)
	if err != nil {
		if os.IsNotExist(err) {
			err = WriteServeImg(w, r, of, scaledName)
			if err != nil {
				Error(w, err, 500)
			}
		} else {
			Error(w, err, 500)
		}
		return
	}

	// @fixme No errors. Now check the scaled copy age.
	http.ServeFile(w, r, scaledName)

	manageCache(fn)
	manageCache(scaledName)

	return

}

func WriteServeGif(w http.ResponseWriter, r *http.Request, of *os.File, f *os.File) (err error) {

	img, err := gif.Decode(of)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	writer := io.MultiWriter(w, f)
	width := getWidth(r)
	method := getMethod(img, width)

	m := resize.Thumbnail(uint(width), 0, img, method)
	err = gif.Encode(writer, m, &gif.Options{NumColors: config.NumColors})

	return

}

func WriteServeImg(w http.ResponseWriter, r *http.Request, of *os.File, scaledName string) (err error) {

	log.Printf("[CREATE] %s", scaledName)
	f, err := os.Create(scaledName)
	if err != nil {
		log.Printf("[ERROR] Could not create %s: %s", scaledName, err)
		return
	}
	defer f.Close()

	switch getMime(scaledName) {
	case "image/png":
		err = WriteServePng(w, r, of, f)
		break
	case "image/jpg":
		err = WriteServeJpeg(w, r, of, f)
		break
	case "image/gif":
		err = WriteServeGif(w, r, of, f)
		break
	default:
		err = errors.New("Unsupported image type")
		break
	}

	if err != nil {
		Error(w, err, 500)
	}

	return

}

func WriteServeJpeg(w http.ResponseWriter, r *http.Request, of *os.File, f *os.File) (err error) {

	img, err := jpeg.Decode(of)
	if err != nil {
		Error(w, err, 500)
		return
	}

	writer := io.MultiWriter(w, f)
	width := getWidth(r)
	method := getMethod(img, width)
	m := resize.Resize(uint(width), 0, img, method)
	err = jpeg.Encode(writer, m, &jpeg.Options{Quality: config.Quality})

	return

}

func getMethod(img image.Image, width int) (method resize.InterpolationFunction) {
	// Check to see if upsizing or downsizing
	method = resize.NearestNeighbor
	if img.Bounds().Max.X-1 < width {
		method = resize.Lanczos3
	}
	return
}

func WriteServePng(w http.ResponseWriter, r *http.Request, of *os.File, f *os.File) (err error) {

	img, err := png.Decode(of)
	if err != nil {
		Error(w, err, 500)
		return
	}

	writer := io.MultiWriter(w, f)
	width := getWidth(r)
	if width == 0 {
		width = img.Bounds().Max.X - 1
	}
	method := getMethod(img, width)

	m := resize.Resize(uint(width), 0, img, method)
	err = png.Encode(writer, m)

	return

}

func getWidth(r *http.Request) (size int) {

	vars := mux.Vars(r)
	width, _ := vars["width"]
	if width == "" {
		width = r.FormValue("w")
	}

	if width != "" {
		newWidth, err := strconv.Atoi(width)
		if err == nil {
			size = newWidth
		}
	}

	return

}

func findClosest(subject float64, choices []float64) (result float64) {
	lastDif := math.Inf(1)
	for _, val := range choices {
		dif := math.Max(subject, val) - math.Min(subject, val)
		if dif <= lastDif {
			lastDif = dif
			result = val
		}
	}
	return
}

func loadConfig() (err error) {

	log.Printf("[STARTUP] Starting up...\n")

	// Load remote configuration file via http/https
	if strings.HasPrefix(configFile, "http") {

		resp, err := http.Get(configFile)
		if err != nil {
			log.Fatalf("[FATAL] Could not load remote config file: %v. Exiting.\n", configFile, err)
		}
		defer resp.Body.Close()
		c, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("[FATAL] Could not read remote config file: %v. Exiting.\n", configFile, err)
		}
		err = json.Unmarshal(c, &config)
		if err != nil {
			log.Fatalf("[FATAL]  %v\n", err)
		}

	} else {

		f, err := os.Open(configFile)
		if err != nil {
			log.Fatalf("[FATAL] Could not load config file %s: %v. Exiting.\n", configFile, err)
		}

		log.Printf("[STARTUP] Loaded config file: %s\n", configFile)

		c, err := ioutil.ReadAll(f)
		err = json.Unmarshal(c, &config)
		if err != nil {
			log.Fatalf("[FATAL]  %v\n", err)
		}

	}

	if config.Storage == "" || storageDir != os.TempDir() {
		config.Storage = storageDir
	}

	if !strings.HasSuffix(config.Storage, "/") {
		config.Storage += "/"
	}

	log.Printf("[STARTUP] Storage directory is: %s\n", config.Storage)
	log.Printf("[STARTUP] Listening on: %s\n", config.Listen)
	log.Printf("[STARTUP] Startup succeeded\n")

	return

}

func manageCache(fn string) (err error) {

	// Now delete if expired.
	f, err := os.Open(fn)
	if err != nil {
		log.Printf("[WARN] Could not open file: %s\n", fn)
		return
	}

	info, err := f.Stat()
	if err != nil {
		log.Printf("[WARN] Could not read fileinfo: %s\n", fn)
		return
	}

	age := time.Since(info.ModTime()).Minutes()
	size := info.Size()
	if age > config.Ttl || size == 0 {
		err = os.Remove(fn)
		if err == nil {
			log.Printf("[EXPIRED] %f %s\n", age, fn)
		}
	} else {
		log.Printf("[CACHE] %f %s\n", age, fn)
	}

	// Append usage to server base dir.
	base := strings.Join(strings.Split(fn, "/")[0:3], "/")
	logFile := fmt.Sprintf("%s/usage.log", base)
	l, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664)
	if err != nil {
		log.Printf("[ERROR] Could not open log file %s", logFile)
		return
	}
	defer l.Close()

	_, err = l.WriteString(fmt.Sprintf("%d,%d\n", size, time.Now().Unix()))
	if err != nil {
		log.Printf("[ERROR] Could not write to %s", logFile)
	}

	return
}

func saveFile(url string, fn string, ip string) (err error, code int) {

	log.Printf("[DOWNLOAD] %s %s", ip, url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		code = 500
		return
	}

	req.Header.Add("X-Forwarded-For", ip)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		code = 500
		return
	}
	defer resp.Body.Close()

	code = resp.StatusCode
	if code >= 400 {
		err = errors.New(resp.Status)
		return
	}

	// Make the directory
	dir := path.Dir(fn)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		return
	}

	// Open the file for writing.
	f, err := os.Create(fn)
	if err != nil {
		log.Printf("[ERROR] %s\n", err)
		return
	}
	defer f.Close()

	// Write to the file and the http stream.
	_, err = io.Copy(f, resp.Body)

	return

}
