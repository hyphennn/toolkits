// Package csvutils
// Create-time: 2025/3/4
package csvutils_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hyphennn/toolkits/csvutils"
	"github.com/stretchr/testify/assert"
)

type Test struct {
	ReleaseName string  `csvseq:"1"`
	OrgTeam     string  `csvseq:"0"`
	TP90        float32 `csvrh:"test"`
	TP99        float64 `csvrh:"test"`
	Count       int
}

func testHandler(str *string) {
	s := *str
	f, _ := strconv.ParseFloat(s, 64)
	f /= 1000
	*str = strconv.FormatFloat(f, 'g', -1, 64)
}

func TestReadCsv(t *testing.T) {
	err := csvutils.RegisterHandler("test", testHandler)
	if err != nil {
		panic("err: " + err.Error())
	}
	res, err := csvutils.ReadCsv[Test]("test.csv", os.O_RDWR, 0777, true)
	if err != nil {
		panic("err: " + err.Error())
	}
	assert.Equal(t, len(res), 50)
	for _, r := range res {
		fmt.Println(r)
	}

	os.Remove("output.csv")
	f, err := csvutils.WriteCsv("output.csv", os.O_RDWR|os.O_CREATE, 0777, res, nil)
	defer f.Close()
	if err != nil {
		panic(err)
	}
}
