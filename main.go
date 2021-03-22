package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func checkExist(file string) {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		log.Fatal("input file is not exist: ", file)
	}
}

func splitFilePath(file string) (string, string, string) {
	base := path.Base(file)
	ext := filepath.Ext(file)
	dir := path.Dir(file)
	filename := strings.TrimSuffix(base, ext)
	return dir, filename, ext
}

func checkDAR(file string) bool {
	out, _ := exec.Command("ffprobe", file).CombinedOutput()
	output := string(out)
	fmt.Println(output)
	return strings.Contains(output, "DAR")
}

func changeAspect(dir string, filename string, ext string) {
	originalFile := filepath.Join(dir, filename+ext)
	bakFile := filepath.Join(dir, filename+"_bak"+ext)

	fmt.Println(originalFile)
	fmt.Println(bakFile)

	os.Rename(originalFile, bakFile)
	out, _ := exec.Command("ffmpeg", "-i", bakFile, "-c", "copy", "-aspect", "16:9", originalFile).CombinedOutput()
	fmt.Println(string(out))
}

func genMD5(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal("md5生成エラー")
	}

	defer file.Close()

	hash := md5.New()

	if _, err := io.Copy(hash, file); err != nil {
		log.Fatal("md5生成エラー")
	}

	hashInBytes := hash.Sum(nil)[:16]
	return hex.EncodeToString(hashInBytes)
}

func main() {
	// ファイルの存在チェック
	filePath := os.Args[1]
	checkExist(filePath)

	// ディレクトリ、ファイル名、拡張子に分解する
	dir, filename, ext := splitFilePath(filePath)

	// ffprobeコマンドの結果をみて、DARという行があるか判定する
	containsDAR := checkDAR(filePath)
	fmt.Println("contains DAR: ", containsDAR)

	// DARがあったらアスペクト比を16:9へ変更する
	if containsDAR {
		changeAspect(dir, filename, ext)
	}

	// ファイルの内容からmd5を作る
	md5 := genMD5(filePath)
	fmt.Println(md5)

	// md5を使ってリネームする

	// probeファイルを作る

	// probeファイルの文字コードをUTF-8にする

	// シーン情報を取得する

	// シーン情報から静止画を切り出す

	// 静止画からサムネイルGIFを作る
}
