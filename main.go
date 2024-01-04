package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/mholt/archiver/v4"
)

// 创建目录
func createDir(dir string) {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
}

// 取文件名，特别是zip压缩包在linux上可能存在乱码
func getFilenameWithoutExt(filename string) string {
	name := filepath.Base(filename)
	ext := filepath.Ext(filename)
	return name[0 : len(name)-len(ext)]
}

// 获取学生名，去除(1)、(2)之类的字符
func getStudentName(name string) string {
	var re = regexp.MustCompile(`([^(]*)\(.*\)(.*)`)
	return re.ReplaceAllString(name, `$1$2`)
}

// 获取临时目录
func getTempFileName() string {
	tempfile, _ := os.CreateTemp("", "ovtVigYrcgSAoZkbI9jg5")
	name := tempfile.Name()
	tempfile.Close()
	defer os.Remove(name)
	return name
}

// 从压缩包解压文件
func copyfile(fsys fs.FS, source string, dest string) error {
	src, err := fsys.Open(source)
	if err != nil {
		return err
	}
	dst, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}
	err = src.Close()
	if err != nil {
		return err
	}
	return dst.Close()
}

// 压缩包遍历
func walk(filename string, paths []string) error {
	fsys, err := archiver.FileSystem(context.Background(), filename)
	if err != nil {
		log.Println(err)
		// 忽略错误
		// panic(err)
		return nil
	}
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		// 忽略所有错误，特别是在linux上少数zip包中文件名会出现乱码导致解码错误
		if err != nil {
			log.Println("Walking: ", path, "Dir?", d.IsDir())
			// return err
		}
		if path == ".git" {
			return fs.SkipDir
		}
		ext := filepath.Ext(path)
		switch ext {
		case ".doc", ".docx":
			student := getStudentName(paths[0])
			dir := "实验报告"
			createDir(dir)
			name := filepath.Join(dir, student+ext)
			e := copyfile(fsys, path, name)
			if e != nil {
				log.Println("解压错误: ", path, " => name: ", name)
			}
		case ".rar", ".zip":
			newPaths := make([]string, len(paths)+1)
			copy(newPaths, paths)
			name := getFilenameWithoutExt(path)
			newPaths = append(paths, name)

			tempfileName := getTempFileName() + ext
			e := copyfile(fsys, path, tempfileName)
			if e == nil {
				walk(tempfileName, newPaths)
				os.Remove(tempfileName)
			} else {
				log.Println("解压错误: ", path, " => name: ", tempfileName)
			}
		case ".py":
			student := getFilenameWithoutExt(paths[0])
			dir := "学生实验作品"
			createDir(dir)
			name := filepath.Join(dir, student+ext)
			e := copyfile(fsys, path, name)
			if e != nil {
				log.Println("解压错误: ", path, " => name: ", name)
			}
		}
		return nil
	})

	return err
}

func main() {
	for _, arg := range os.Args[1:] {
		walk(arg, []string{})
	}
}
