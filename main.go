package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
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

func renameToMD5(dir string, filename string, ext string) string {
	filePath := filepath.Join(dir, filename+ext)
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
	hashString := hex.EncodeToString(hashInBytes)

	newFilePath := filepath.Join(dir, hashString+ext)
	os.Rename(filePath, newFilePath)

	return hashString
}

func createFFPROBE(dir string, filename string, ext string) {
	filePath := filepath.Join(dir, filename+ext)
	out, _ := exec.Command("ffprobe", filePath).CombinedOutput()

	probeFilePath := filepath.Join(dir, filename+".txt")
	err := ioutil.WriteFile(probeFilePath, out, 0644)
	if err != nil {
		fmt.Println(err)
		log.Fatal("probe error")
	}
}

func readCSV(dir string, filename string, ext string) {
	filePath := filepath.Join(dir, filename+ext)
	fmt.Println(filePath)
	reader, _ := os.Open(filePath)
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1

	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for i := 2; i < len(records); i++ {
		a, _ := strconv.ParseFloat(records[i][3], 32)
		b, _ := strconv.ParseFloat(records[i][6], 32)
		mean := (a + b) / 2.0
		fmt.Println(mean)
	}
}

func main() {
	// ファイルの存在チェック
	if len(os.Args) < 2 {
		log.Fatal("one arg is required")
	}
	filePath := os.Args[1]
	checkExist(filePath)

	// ディレクトリ、ファイル名、拡張子に分解する
	dir, filename, ext := splitFilePath(filePath)

	// ffprobeコマンドの結果をみて、DARという行があるか判定する
	// DARがあったらアスペクト比を16:9へ変更する
	if checkDAR(filePath) {
		changeAspect(dir, filename, ext)
	}

	// ファイルの内容からmd5を作る
	md5 := renameToMD5(dir, filename, ext)
	fmt.Println(md5)

	// probeファイルを作る
	createFFPROBE(dir, md5, ext)

	// シーン情報を取得する
	// scenedetect --input cc33da8a36b5a9833e3862ed120de013.mp4 detect-content list-scenes

	// シーン情報から静止画を切り出す
	// cat cc33da8a36b5a9833e3862ed120de013-Scenes.csv
	readCSV(dir, md5+"-Scenes", ".csv")

	// 静止画からサムネイルGIFを作る
	// ffmpeg -y -i ${tempDir}/${file} -vframes 1 -vf scale=320:-1 -ss ${lapTime} -f image2 ${tempDir}/${basename}_${formatted}.jpg

}
