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
	log.Println(output)
	return strings.Contains(output, "DAR")
}

func changeAspect(dir string, filename string, ext string) {
	originalFile := filepath.Join(dir, filename+ext)
	bakFile := filepath.Join(dir, filename+"_bak"+ext)

	log.Println(originalFile)
	log.Println(bakFile)

	os.Rename(originalFile, bakFile)
	out, _ := exec.Command("ffmpeg", "-i", bakFile, "-c", "copy", "-aspect", "16:9", originalFile).CombinedOutput()
	log.Println(string(out))
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
		log.Println(err)
		log.Fatal("probe error")
	}
}

func createSceneCSV(dir string, filename string, ext string) {
	filePath := filepath.Join(dir, filename+ext)
	log.Println("createSceneCSV: " + filePath)
	_, err := exec.Command("scenedetect", "--input", filePath, "-o", dir, "detect-content", "list-scenes").CombinedOutput()
	if err != nil {
		log.Println(err)
		log.Fatal("scenedetect error")
	}
}

func readCSV(dir string, filename string, ext string) []float32 {
	filePath := filepath.Join(dir, filename+ext)
	log.Println(filePath)
	reader, _ := os.Open(filePath)
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1

	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	slice := make([]float32, 1)
	for i := 2; i < len(records); i++ {
		a, _ := strconv.ParseFloat(records[i][3], 32)
		b, _ := strconv.ParseFloat(records[i][6], 32)
		mean := float32((a + b) / 2.0)
		slice = append(slice, mean)
	}

	return slice
}

func createThumbnailGif(dir string, filename string, ext string, scenes []float32) {
	filePath := filepath.Join(dir, filename+ext)
	for i, v := range scenes {
		s := fmt.Sprintf("%08.3f", v)
		log.Printf("#%3d: %s\n", i, s)
		imageFilePath := filepath.Join(dir, filename+"_"+s+".jpg")
		_, err := exec.Command("ffmpeg", "-y", "-i", filePath, "-vframes", "1", "-vf", "scale=320:-1", "-ss", s, "-f", "image2", imageFilePath).CombinedOutput()
		if err != nil {
			log.Println(err)
			log.Fatal("ffmpeg error")
		}
	}

	imageFiles := filepath.Join(dir, filename+"*.jpg")

	gifFile := filepath.Join(dir, filename+".gif")
	_, err := exec.Command("convert", "-delay", "100", imageFiles, gifFile).CombinedOutput()
	if err != nil {
		log.Println(err)
		log.Fatal("covert error")
	}
}

func clean(dir string, filename string, ext string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, filename) && (strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".csv")) {
			log.Println(path)
			os.Remove(path)
		}
		return nil
	})
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
	log.Printf("rename to md5: %s", md5)

	// probeファイルを作る
	createFFPROBE(dir, md5, ext)

	// シーン情報を取得する
	createSceneCSV(dir, md5, ext)

	// シーン情報から静止画を切り出す
	scenes := readCSV(dir, md5+"-Scenes", ".csv")

	// 静止画からサムネイルGIFを作る
	createThumbnailGif(dir, md5, ext, scenes)

	// 中間ファイルを削除する
	clean(dir, md5, ext)

	log.Printf("thumbnail created!: %s/%s%s\n", dir, md5, ".gif")
	fmt.Println(md5)
}
