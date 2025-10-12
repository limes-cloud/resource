package filex

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func IsExistFolder(folderPath string) bool {
	fileInfo, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return fileInfo.IsDir()
}

func IsExistFile(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func ZipDir(dir, output string) error {
	zipfile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	// 递归遍历目录
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// 修正文件路径
		header.Name = filepath.ToSlash(path[len(dir):])
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, er := os.Open(path)
			if er != nil {
				return er
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
		}
		return err
	})
}

func ZipFiles(output string, files map[string]string) error {
	// 创建一个 ZIP 文件
	zipFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// 创建一个新的 ZIP 写入器
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 文件名及其在 ZIP 内的重命名
	filesToZip := files

	// 遍历文件列表，逐个添加到 ZIP 文件
	for originalName, newName := range filesToZip {
		// 打开待压缩的文件
		fileToZip, err := os.Open(originalName)
		if err != nil {
			panic(err)
		}
		defer fileToZip.Close()

		// 获取文件的信息，以便复制文件的元数据
		info, err := fileToZip.Stat()
		if err != nil {
			return err
		}

		// 创建 ZIP 文件中的一个条目，并指定新的文件名
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = newName
		header.Method = zip.Deflate // 设置压缩算法

		// 创建条目的写入器
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// 将文件内容复制到 ZIP 文件中的条目
		if _, err = io.Copy(writer, fileToZip); err != nil {
			return err
		}
	}
	return nil
}
