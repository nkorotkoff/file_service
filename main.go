package main

import (
	"encoding/json"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/joho/godotenv"
	"image"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const (
	uploadDirectoryEnv = "upload_directory"
)

type uploadRequest struct {
	Params []string
}

var uploadDirectory string

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	dir, exists := os.LookupEnv(uploadDirectoryEnv)

	if exists {
		uploadDirectory = dir
	} else {
		panic(uploadDirectoryEnv + "not exist")
	}
	createDirectoryIfNotExist()
}

func main() {
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/delete/", deleteFunc)
	http.HandleFunc("/uploads/", fileHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	file, handler, err := r.FormFile("image")
	if err != nil {
		log.Println("Error Retrieving the File")
		log.Println(err)
		return
	}

	defer file.Close()
	log.Printf("Uploaded File: %+v\n", handler.Filename)
	log.Printf("File Size: %+v\n", handler.Size)
	log.Printf("MIME Header: %+v\n", handler.Header)
	filePath := uploadDirectory + "/" + generateRandomName() + ".png"

	paramsValue := r.FormValue("params")
	var body uploadRequest
	if err := json.Unmarshal([]byte(paramsValue), &body); err != nil {
		log.Printf("Error decoding request body: %v\n", err)
	}

	if len(body.Params) > 0 && body.Params[0] == "resize" {
		img, err := resizeImage(file, body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = imaging.Save(img, filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Successfully Uploaded and Resized File\n")
	} else {
		out, err := os.Create(filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Successfully Uploaded File\n")
	}
}

func resizeImage(file multipart.File, body uploadRequest) (*image.NRGBA, error) {
	src, err := imaging.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode image: %v", err)
	}

	width, err := strconv.Atoi(body.Params[1])
	if err != nil {
		return nil, fmt.Errorf("Invalid width: %v", err)
	}

	height, err := strconv.Atoi(body.Params[2])
	if err != nil {
		return nil, fmt.Errorf("Invalid height: %v", err)
	}

	dstSize := image.Point{X: width, Y: height}

	dst := imaging.Resize(src, dstSize.X, dstSize.Y, imaging.Lanczos)

	return dst, nil
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path)
	if filename == "/" {
		http.NotFound(w, r)
		return
	}

	file, err := ioutil.ReadFile(uploadDirectory + "/" + filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	contentType := http.DetectContentType(file)

	w.Header().Set("Content-Type", contentType)

	w.Write(file)
}

func deleteFunc(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path)
	e := os.Remove(uploadDirectory + "/" + filename)
	if e != nil {
		log.Print(e)
		http.NotFound(w, r)
	}
	fmt.Fprintf(w, "Successfully Deleted File\n")
}

func createDirectoryIfNotExist() {
	_, err := os.Stat(uploadDirectory)

	if os.IsNotExist(err) {
		if err := os.Mkdir(uploadDirectory, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	}
}

func generateRandomName() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	name := make([]byte, 16)
	for i := range name {
		name[i] = chars[rand.Intn(len(chars))]
	}

	return string(name)
}
