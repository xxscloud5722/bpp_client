package common

import (
	"fmt"
	"github.com/samber/lo"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)
import "archive/zip"

// Zip 压缩文件; src 源目录, target 输出文件, delete 是否删除源
func Zip(src, target string, delete bool) error {
	// 如果目标文件已存在删除
	if _, err := os.Stat(target); err == nil || !os.IsNotExist(err) {
		err := os.Remove(target)
		if err != nil {
			return err
		}
	}

	// 创建准备写入的文件
	fw, err := os.Create(target)
	defer func(fw *os.File) {
		err := fw.Close()
		if err != nil {
			panic(err)
		}
	}(fw)
	if err != nil {
		return err
	}

	// 通过 fw 来创建 zip.Write
	zw := zip.NewWriter(fw)
	defer func() {
		// 检测一下是否成功关闭
		if err := zw.Close(); err != nil {
			panic(err)
		}
	}()

	// 下面来将文件写入 zw ，因为有可能会有很多个目录及文件，所以递归处理
	var startIndex = len(src) - lo.If(strings.HasPrefix(src, "."), 1).Else(0)
	err = filepath.Walk(src, func(filePath string, fi os.FileInfo, errBack error) (err error) {
		if errBack != nil {
			return errBack
		}

		// 通过文件信息，创建 zip 的文件信息
		fh, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}

		// 替换文件信息中的文件名
		var originalFileName = fh.Name
		fh.Name = strings.TrimPrefix(strings.TrimPrefix(filePath[startIndex:], "/"), "\\")
		if fi.IsDir() && fh.Name == "" {
			return nil
		}

		// 如果根路径是一个文件
		if fh.Name == "" && !fi.IsDir() {
			fh.Name = originalFileName
		}

		// 这步开始没有加，会发现解压的时候说它不是个目录
		if fi.IsDir() {
			fh.Name += "/"
		}

		fh.Name = strings.ReplaceAll(fh.Name, "\\", "/")

		// 写入文件信息，并返回一个 Write 结构
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}

		// 检测，如果不是标准文件就只写入头信息，不写入文件数据到 w
		// 如目录，也没有数据需要写
		if !fh.Mode().IsRegular() {
			return nil
		}

		// 打开要压缩的文件
		fr, err := os.Open(filePath)
		defer func(fr *os.File) {
			err := fr.Close()
			if err != nil {
				panic(err)
			}
		}(fr)
		if err != nil {
			return err
		}

		// 将打开的文件 Copy 到 w
		fileSize, err := io.Copy(w, fr)
		if err != nil {
			return err
		}
		// 输出压缩的内容
		fmt.Printf("[Zip]：%s <-- %s (%s)\n", path.Base(target), filePath, fileSizeFormat(fileSize))
		return nil
	})
	if err != nil {
		return err
	}
	// 是否删除源文件
	if delete {
		err = os.RemoveAll(src)
		if err != nil {
			return err
		}
	}
	return nil
}

// fileSizeFormat 文件大小格式化
func fileSizeFormat(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
