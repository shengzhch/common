//Decode Json File To Struct
package util

import (
	"reflect"
	"fmt"
	"strings"
	"os"
	"errors"
	"encoding/json"
)

func setField(obj interface{}, name string, value interface{}, withtag string) error {
	structData := reflect.ValueOf(obj).Elem()
	var fieldValue reflect.Value
	if withtag != "" {
		fieldValue = structData.FieldByNameFunc(func(field string) bool {
			if fieldInfo, ok := reflect.TypeOf(obj).Elem().FieldByName(field); !ok {
				return false
			} else {
				return fieldInfo.Tag.Get(withtag) == name
			}
		})
	} else {
		fieldValue = structData.FieldByName(strings.ToTitle(name))
	}

	if !fieldValue.IsValid() {
		return fmt.Errorf("No such field: %s in obj %+v", obj)
	}

	if !fieldValue.CanSet() {
		return fmt.Errorf("Cannot set %s field value ", name)
	}
	fieldType := fieldValue.Type()
	val := reflect.ValueOf(value)
	valTypeStr := val.Type().String()
	fieldTypeStr := fieldType.String()
	if valTypeStr == "float64" && fieldTypeStr == "int" {
		val = val.Convert(fieldType)
	} else if fieldType != val.Type() {
		return fmt.Errorf("Provided value type " + valTypeStr + " didn't match obj field type " + fieldTypeStr)
	}
	fieldValue.Set(val)
	return nil
}

func SetStruct(obj interface{}, defs map[string]interface{}, withtag string) error {
	var err error
	for k, v := range defs {
		if err = setField(obj, k, v, withtag); err != nil {
			return err
		}
	}
	return nil
}

func JsonFileToMap(path string) (rel map[string]interface{}, err error) {
	file, err := os.Open("path")
	if err != nil {
		return nil, errors.New("open file failed " + err.Error())
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(rel)
	return
}
