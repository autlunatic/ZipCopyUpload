package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/fatih/color"

	"github.com/autlunatic/goConfig"
	"github.com/autlunatic/goUtil/Zipping"
	"github.com/autlunatic/goUtil/ftp"
)

const confFile = "ZipCopyUpload.conf"

func main() {
	// load Configfile
	var conf ZipCopyUpload
	file, err := os.OpenFile(confFile, os.O_RDWR, 0666)
	defer file.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	crw := encryptedConfig.ConfigReadWriter{&conf, file, "thisIsTheEllizBatPW"}
	readConfErr := crw.DoRead()
	if readConfErr != nil {
		log.Fatal("config error, script aborted! ", err)
	}
	file.Close()
	// first zip all files ---------------------------------------------------
	if !handleZipping(conf) {
		return
	}
	// then copy all files ----------------------------------------------------
	if !handleCopy(conf) {
		return
	}
	// then upload all files --------------------------------------------------
	handleUploadFiles(conf)
	fmt.Print("Press 'Enter' to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

type FromTo struct {
	FromFileName string
	ToFileName   string
}
type ftpConfig struct {
	Host       string
	Username   string
	Password   string `encrypted:"true"`
	RemotePath string
}
type FromSliceTo struct {
	FromFileNames []string
	ToFileName    string
}

type FileUploadConf struct {
	FileName  string
	FTPConfig ftpConfig
}
type ZipCopyUpload struct {
	ZipFiles    []FromSliceTo
	CopyToDirs  []FromTo
	UploadFiles []FileUploadConf
}

func handleUploadFiles(conf ZipCopyUpload) {
	ec := make(chan error)
	var goroutines int
	for _, c := range conf.UploadFiles {
		goroutines++
		fmt.Println("uploading...", c.FileName)
		uc := ftp.UploadConf{
			c.FTPConfig.Host,
			c.FTPConfig.Username,
			c.FTPConfig.Password,
			c.FTPConfig.RemotePath,
			c.FileName}
		go func() {
			err := ftp.UploadFile(uc)
			ec <- err
		}()
	}
	var successCount int
	for e := range ec {
		goroutines--
		if e != nil {
			color.Red(fmt.Sprint("ERROR! -> ", e))
		} else {
			successCount++
		}
		if goroutines == 0 {
			close(ec)
		}
	}
	if successCount == len(conf.UploadFiles) {
		color.Green("%d files uploaded", successCount)
	} else {
		color.Red("%d files uploaded", successCount)
		color.Red("ERROR! Only %d files uploaded! Check Log!", successCount)
	}
}

func checkMkDir(toFile string) {
	if _, err := os.Stat(filepath.Dir(toFile)); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(toFile), os.ModeDir)
		if err == nil {
			color.Yellow(fmt.Sprint(filepath.Dir(toFile), " didnt exist, it was created!"))
		}
	}
}

func copyFile(fromFile string, toFile string) error {
	from, err := os.Open(fromFile)
	defer from.Close()
	if err != nil {
		return err
	}
	checkMkDir(toFile)
	to, err := os.OpenFile(toFile, os.O_RDWR|os.O_CREATE, 0666)
	defer to.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}
	return nil
}

func canContinue() bool {
	fmt.Print("Continue? (y/n):")
	var input string
	fmt.Scanln(&input)
	if input != "y" {
		return false
	}
	return true
}

func handleZipping(conf ZipCopyUpload) bool {
	var okCnt int
	for _, z := range conf.ZipFiles {
		fmt.Println("zipping...", z.FromFileNames, " > ", z.ToFileName)
		checkMkDir(z.ToFileName)
		err := Zipping.ZipFiles(z.ToFileName, z.FromFileNames)
		if err != nil {
			color.Red(fmt.Sprint("ERROR! > ", err))
		} else {
			okCnt++
		}
	}
	if len(conf.ZipFiles) == okCnt {
		color.Green("%d zipfiles created...", okCnt)
	} else {
		color.Red("%d from %d zipfiles created...", okCnt, len(conf.ZipFiles))
		if !canContinue() {
			return false
		}
	}
	return true
}

func handleCopy(conf ZipCopyUpload) bool {
	var okCnt int
	for _, c := range conf.CopyToDirs {
		fmt.Println("copying...", c.FromFileName, " > ", c.ToFileName)
		err := copyFile(c.FromFileName, c.ToFileName)
		if err != nil {
			color.Red(fmt.Sprint("ERROR! > ", err))
		} else {
			okCnt++
		}
	}
	if len(conf.CopyToDirs) == okCnt {
		color.Green("%d file copied...", okCnt)
	} else {
		color.Red("%d from %d file copied...", okCnt, len(conf.CopyToDirs))
		if !canContinue() {
			return false
		}
	}
	return true
}
