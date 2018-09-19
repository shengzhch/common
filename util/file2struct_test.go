package util

import (
	"testing"
	"time"
	"encoding/json"
	"os"
)

type TS struct {
	Name  string    `json:"name"`
	Age   int       `json:"age"`
	Lengh float64   `json:"len"`
	Time  time.Time `json:"birth"`
	TSC   TSC      `json:"tsc"`
	//*TSC2           `json:"tsc2"`
}
type TSC struct {
	Namec string `json:"namec"`
}

type TSC2 struct {
	Namec2 string `json:"namec2"`
}

func TestSetStruct(t *testing.T) {
	t.Logf("start")
	var ts = TS{
		Name:  "zhangsan",
		Age:   10,
		Lengh: 10,
		Time:  time.Now(),
		TSC: TSC{
			Namec: "zhangsanchild1",
		},
		//TSC2: nil,
	}
	bytes, _ := json.Marshal(ts)
	file, err := os.Create("a.json")
	if err != nil {
		t.Error("err1 ", err)
		return
	}
	file.Write(bytes)
	file.Close()
	rel, err := JsonFileToMap("a.json")
	if err != nil {
		t.Error("err2 ", err)
		return
	}
	t.Logf("rel---- %+v", rel)

	var tsn = new(TS)

	err = SetStruct(tsn, rel, "json")
	if err != nil {
		t.Error("err3 ", err)
		t.Logf("tsn %+v", tsn)
		return
	}
	t.Logf("tsn %+v", tsn)

}
