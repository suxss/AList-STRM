package main

import (
	"flag"
	"fmt"
	"github.com/studio-b12/gowebdav"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	host         = flag.String("h", "http://localhost:5244/dav", "AList webdav 地址")
	user         = flag.String("u", "admin", "AList webdav 用户名")
	password     = flag.String("p", "12345", "AList webdav 密码")
	remoteDir    = flag.String("r", "", "AList directory")
	localDir     = flag.String("l", ".", "Local directory")
	downloadExts = flag.String("d", ".jpg,.jpeg,.png,.gif,.nfo,.srt,.ass,.ssa", "直接下载的文件后缀名(以逗号分隔)")
	strmExts     = flag.String("s", ".mp4,.avi,.mkv,.flv", "生成strm的文件后缀名(以逗号分隔)")
)

func main() {
	flag.Parse()
	downloadExt := strings.Split(*downloadExts, ",")
	strmExt := strings.Split(*strmExts, ",")
	err := generate(*host, *user, *password, *remoteDir, *localDir, downloadExt, strmExt)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("生成完毕")
}

func generate(host string, user string, password string, remoteDir string, localDir string, downloadExt []string, strmExt []string, goMaxNum ...int) error {
	client := gowebdav.NewClient(host, user, password)
	if err := client.Connect(); err != nil {
		return err
	}
	goNum := 10
	if len(goMaxNum) > 0 {
		goNum = goMaxNum[0]
	}
	wg := new(sync.WaitGroup)
	once := new(sync.Once)
	tokens := make(chan struct{}, goNum)
	err := walk(client, remoteDir, func(fi os.FileInfo, path string) (err error) {
		tokens <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
				<-tokens
			}()
			e := parse(host, client, fi.Name(), path, localDir, downloadExt, strmExt)
			if e != nil {
				fmt.Println(e)
				once.Do(func() {
					err = e
				})
			}
		}()
		return
	})
	wg.Wait()
	return err
}

func walk(client *gowebdav.Client, path string, callback func(fi os.FileInfo, path string) error) error {
	files, err := client.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			return walk(client, path+"/"+file.Name(), callback)
		} else {
			return callback(file, path)
		}
	}
	return nil
}

func downloadFile(client *gowebdav.Client, remotePath string, localDir string) error {
	reader, err := client.ReadStream(remotePath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		if err := os.MkdirAll(localDir, 0777); err != nil { //os.ModePerm
			return err
		}
	}
	fileName := path.Base(remotePath)
	file, err := os.Create(localDir + "/" + fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func generateSTRM(host string, remotePath string, localDir string, fileName string) error {
	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		if err := os.MkdirAll(localDir, 0777); err != nil { //os.ModePerm
			return err
		}
	}
	file, err := os.Create(localDir + "/" + fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(strings.Replace(host, "/dav", "/d", 1) + "/" + remotePath)
	return err
}

func parse(host string, client *gowebdav.Client, filename string, path string, localDir string, downloadExt []string, strmExt []string) error {
	fileExt := filepath.Ext(filename)
	sort.Strings(downloadExt)
	sort.Strings(strmExt)
	index := sort.SearchStrings(strmExt, fileExt)
	if index < len(strmExt) && strmExt[index] == fileExt {
		return generateSTRM(host, path+"/"+filename, localDir+"/"+path, strings.TrimSuffix(filename, fileExt)+".strm")
	}
	if index = sort.SearchStrings(downloadExt, fileExt); index < len(downloadExt) && downloadExt[index] == fileExt {
		return downloadFile(client, path+"/"+filename, localDir+"/"+path)
	}
	return nil
}
