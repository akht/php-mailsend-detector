package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/akht/php-mailsend-detector/detector"
)

func main() {
	// ファイルをOpenする
	f, err := os.Open("test.php")
	if err != nil {
		fmt.Println("error")
	}
	defer f.Close()

	// 一気に全部読み取り
	b, err := ioutil.ReadAll(f)
	src := bytes.NewBufferString(string(b))

	d := detector.NewDetector(src)
	fmt.Println(d.Detect())
}
