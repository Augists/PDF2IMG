/*
 * Author: Augists
 * Date: 2022-1-18
 * Description:
 *    Convert PDF to images
 *    Execute on Web
 *    Based on pdftoppm on linux
 * Contact: augists.top
 */

package main

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var DEBUG_PRINT bool = true

func main() {
	// https://studygolang.com/articles/9121
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", index)
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/convert", convert)
	http.HandleFunc("/download", download)

	// PDF2IMG: 7219(P2IG)
	fmt.Println("Listening on http://localhost:7219")
	log.Fatal(http.ListenAndServe(":7219", nil))
}

/*
 * Handle index page
 *    template path: ./tmpl/index.html
 *    TODO: replace image
 */
func index(w http.ResponseWriter, r *http.Request) {
	if DEBUG_PRINT {
		debugFuncPrint("Index", r)
	}
	t, err := template.ParseFiles("./tmpl/index.html")
	if err != nil {
		panic(err)
	}
	t.Execute(w, nil)
}

/*
 * Handle upload page
 *    template path: ./tmpl/upload.html
 *    TODO: beautify upload file button
 *    TODO: replace image
 */
func upload(w http.ResponseWriter, r *http.Request) {
	if DEBUG_PRINT {
		debugFuncPrint("Upload", r)
	}
	crutime := time.Now().Unix()
	h := md5.New()
	io.WriteString(h, strconv.FormatInt(crutime, 10))
	token := fmt.Sprintf("%x", h.Sum(nil))

	t, err := template.ParseFiles("./tmpl/upload.html")
	if err != nil {
		panic(err)
	}
	t.Execute(w, token)
}

/*
 * Handle convert page
 *    TODO: will remove upload directory if upload is empty
 */
func convert(w http.ResponseWriter, r *http.Request) {
	if DEBUG_PRINT {
		debugFuncPrint("Convert", r)
		fmt.Println("\tGettings file")
	}
	fileName, err := getFile(r)
	if err != nil {
		fmt.Println(err)
		return
	}
	filePath := "./upload/" + fileName
	if DEBUG_PRINT {
		fmt.Println("\tFile path:", filePath)
		// fmt.Fprintf(w, "%v", handler.Header)
		fmt.Println("\tConverting")
	}
	if _, err = os.Stat(filePath); err != nil {
		fmt.Println(err)
		return
	}
	defer os.Remove(filePath)

	/*
	 * how to convert pdf to images
	 */
	outPath := "./upload/" + strings.TrimSuffix(fileName, ".pdf") + string(os.PathSeparator)
	os.Mkdir(outPath, 0777)

	doneConvert := make(chan bool)
	go func() {
		// option := []string{"jpeg", "png"}
		option := "jpeg"
		cmd := exec.Command("pdftoppm", "-"+option, filePath, outPath+strings.TrimSuffix(fileName, ".pdf"))
		if DEBUG_PRINT {
			fmt.Println("--------------------------------------------------------")
			fmt.Println("outPath: ", outPath)
			fmt.Println("fileName: ", fileName)
			fmt.Println("filePath: ", filePath)
			fmt.Println("resultPath: ", outPath+strings.TrimSuffix(fileName, ".pdf"))
			fmt.Println(cmd.Args)
			fmt.Println("--------------------------------------------------------")
		}
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
		}
		doneConvert <- true
	}()
	<-doneConvert
	if DEBUG_PRINT {
		fmt.Println("Convert done")
	}
	defer os.RemoveAll(outPath)

	doneArchive := make(chan bool)
	archivePath := "./upload/" + strings.TrimSuffix(fileName, ".pdf") + ".tar.gz"
	go func() {
		// archive the images
		cmd := exec.Command("tar", "-czf", archivePath, outPath)
		if DEBUG_PRINT {
			fmt.Println("--------------------------------------------------------")
			fmt.Println("outPath: ", outPath)
			fmt.Println("archivePath: ", archivePath)
			fmt.Println(cmd.Args)
			fmt.Println("--------------------------------------------------------")
		}
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
		}
		doneArchive <- true
	}()
	<-doneArchive
	if DEBUG_PRINT {
		fmt.Println("Archive done")
	}
	// defer os.Remove(archivePath)

	// redirect to download page
	// http.Redirect(w, r, "/download", http.StatusFound)

	t, err := template.ParseFiles("./tmpl/convert.html")
	if err != nil {
		panic(err)
	}
	// t.Execute(w, nil)
	t.Execute(w, strings.TrimSuffix(fileName, ".pdf")+".tar.gz")
	// t.Execute(os.Stdout, "/download?fn="+strings.TrimSuffix(fileName, ".pdf")+".tar.gz")
}

/*
 * Get file from request
 *    return: file name string, error
 */
func getFile(r *http.Request) (string, error) {
	if DEBUG_PRINT {
		debugFuncPrint("getFile", r)
	}
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		return "", err
	}
	defer file.Close()
	filePath := "./upload/" + handler.Filename
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer f.Close()
	io.Copy(f, file)
	return handler.Filename, nil
}

/*
 * Handle download page
 *    TODO: download file name
 */
func download(w http.ResponseWriter, r *http.Request) {
	if DEBUG_PRINT {
		debugFuncPrint("Download", r)
	}

	r.ParseForm()
	fileName := r.Form["fn"][0]
	if fileName == "" {
		w.Write([]byte("No file name"))
		return
	}
	archivePath := "./upload/" + fileName
	if _, err := os.Stat(archivePath); err != nil {
		w.Write([]byte("No such file"))
		return
	}
	defer os.Remove(archivePath)

	fileData, err := ioutil.ReadFile(archivePath)
	if err != nil {
		log.Println("Read File Err:", err.Error())
	} else {
		log.Println("Send File:", archivePath)
		w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
		w.Write(fileData)
	}
}

/*
 * Output debug information for function to stdout
 */
func debugFuncPrint(funcName string, r *http.Request) {
	fmt.Println(funcName)
	fmt.Println("\tmethod:", r.Method)
	fmt.Println("\tpath:", r.URL.Path)
	fmt.Println("\tscheme:", r.URL.Scheme)
	fmt.Println("\t", r.Form)
	fmt.Println("\t", r.Form["url_long"])
}
