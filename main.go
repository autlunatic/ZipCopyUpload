package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/autlunatic/goUtil/ftp"

	"github.com/autlunatic/goConfig"
	"github.com/autlunatic/goUtil/Zipping"
)

const confFile = "ZipCopyUpload.conf"

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

type FileUploadConf struct {
	FileName  string
	FTPConfig ftpConfig
}
type ZipCopyUpload struct {
	ZipFiles    []FromTo
	CopyToDirs  []FromTo
	UploadFiles []FileUploadConf
}

func copyFile(fromFile string, toFile string) error {
	from, err := os.Open(fromFile)
	defer from.Close()
	if err != nil {
		return err
	}

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
	crw.DoRead()
	file.Close()
	// first zip all files ---------------------------------------------------
	for _, z := range conf.ZipFiles {
		files := []string{z.FromFileName}

		fmt.Println("zipping...", z.FromFileName, " > ", z.ToFileName)
		Zipping.ZipFiles(z.ToFileName, files)

	}

	// then copy all files ----------------------------------------------------
	for _, c := range conf.CopyToDirs {

		fmt.Println("copying...", c.FromFileName, " > ", c.ToFileName)
		err := copyFile(c.FromFileName, c.ToFileName)
		if err != nil {
			fmt.Println()
		}
	}

	// then upload all files --------------------------------------------------
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
			fmt.Println("ERROR!!!!!!! -> ", e)
		} else {
			successCount++
		}

		if goroutines == 0 {
			close(ec)
		}
	}
	if successCount == len(conf.UploadFiles) {
		fmt.Println(successCount, "files uploaded")
	} else {

		fmt.Println("ERROR! Only", successCount, "files uploaded! ", "Check Log")
	}
	fmt.Print("Press 'Enter' to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
