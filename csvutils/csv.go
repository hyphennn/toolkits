// Package csvutils
// Create-time: 2025/2/27
package csvutils

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"sync"
)

const (
	tagSeq          = "csvseq"
	tagReadHandler  = "csvrh"
	tagWriteHandler = "csvwh"
)

var tagHandlerMap = sync.Map{}

type TagHandler func(*string)

func RegisterHandler(name string, f TagHandler) error {
	if name == "" {
		return fmt.Errorf("can not register empty name")
	}
	if _, ok := tagHandlerMap.Load(name); ok {
		return fmt.Errorf("register handler %s failed: name exists", name)
	}
	tagHandlerMap.Store(name, f)

	return nil
}

func ReadCsv[T any](filename string, flag int, mode os.FileMode, ignoreHead bool) ([]T, error) {
	handlers, err := preParseRead[T]()
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filename, flag, mode)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(file)
	defer file.Close()

	if ignoreHead {
		_, _ = r.Read()
	}

	ret := []T{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		t := new(T)
		tv := reflect.ValueOf(t)
		for i := 0; i < tv.Elem().NumField(); i++ {
			if err := handlers[i](tv.Elem(), record); err != nil {
				return nil, err
			}
		}
		ret = append(ret, *t)
	}
	return ret, nil
}

func preParseRead[T any]() ([]func(reflect.Value, []string) error, error) {
	t := reflect.TypeFor[T]()
	ret := []func(reflect.Value, []string) error{}
	for i := 0; i < t.NumField(); i++ {
		preHandle := func(record []string) string {
			idx := i
			if v, err := strconv.Atoi(t.Field(i).Tag.Get(tagSeq)); err == nil {
				idx = v
			}
			item := record[idx]
			if fc, ok := tagHandlerMap.Load(t.Field(i).Tag.Get(tagReadHandler)); ok {
				fc.(TagHandler)(&item)
			}
			return item
		}

		switch t.Field(i).Type.Kind() {
		case reflect.String:
			ret = append(ret, func(rv reflect.Value, s []string) error {
				item := preHandle(s)
				rv.Field(i).Set(reflect.ValueOf(item))
				return nil
			})
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ret = append(ret, func(rv reflect.Value, s []string) error {
				item := preHandle(s)
				if v, err := strconv.Atoi(item); err != nil {
					return fmt.Errorf("field %s is not any kind of int", t.Field(i).Name)
				} else {
					rv.Field(i).Set(reflect.ValueOf(v).Convert(rv.Field(i).Type()))
				}
				return nil
			})
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			ret = append(ret, func(rv reflect.Value, s []string) error {
				item := preHandle(s)
				if v, err := strconv.ParseUint(item, 10, 64); err != nil {
					return fmt.Errorf("field %s is not any kind of uint", t.Field(i).Name)
				} else {
					rv.Field(i).Set(reflect.ValueOf(v).Convert(rv.Field(i).Type()))
				}
				return nil
			})
		case reflect.Float32:
			ret = append(ret, func(rv reflect.Value, s []string) error {
				item := preHandle(s)
				if v, err := strconv.ParseFloat(item, 32); err != nil {
					return fmt.Errorf("field %s is not float32", t.Field(i).Name)
				} else {
					rv.Field(i).Set(reflect.ValueOf(float32(v)))
				}
				return nil
			})
		case reflect.Float64:
			ret = append(ret, func(rv reflect.Value, s []string) error {
				item := preHandle(s)
				if v, err := strconv.ParseFloat(item, 64); err != nil {
					return fmt.Errorf("field %s is not float64", t.Field(i).Name)
				} else {
					rv.Field(i).Set(reflect.ValueOf(v))
				}
				return nil
			})
		case reflect.Bool:
			ret = append(ret, func(rv reflect.Value, s []string) error {
				item := preHandle(s)
				if v, err := strconv.ParseBool(item); err != nil {
					return fmt.Errorf("field %s is not bool", t.Field(i).Name)
				} else {
					rv.Field(i).Set(reflect.ValueOf(v))
				}
				return nil
			})
		default:
			return nil, fmt.Errorf("can not handle type of field %s", t.Field(i).Name)
		}
	}
	return ret, nil
}

func preParseWrite[T any]() ([]func(reflect.Value, *[]string) error, error) {
	t := reflect.TypeFor[T]()
	ret := []func(reflect.Value, *[]string) error{}
	for i := 0; i < t.NumField(); i++ {
		afterHandle := func(item string, record *[]string) {
			if fc, ok := tagHandlerMap.Load(t.Field(i).Tag.Get(tagWriteHandler)); ok {
				fc.(TagHandler)(&item)
			}
			idx := i
			if v, err := strconv.Atoi(t.Field(i).Tag.Get(tagSeq)); err == nil {
				idx = v
			}
			(*record)[idx] = item
		}

		switch t.Field(i).Type.Kind() {
		case reflect.String:
			ret = append(ret, func(rv reflect.Value, s *[]string) error {
				afterHandle(rv.Field(i).Interface().(string), s)
				return nil
			})
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ret = append(ret, func(rv reflect.Value, s *[]string) error {
				afterHandle(strconv.FormatInt(rv.Field(i).Convert(reflect.TypeFor[int64]()).Interface().(int64), 10), s)
				return nil
			})
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			ret = append(ret, func(rv reflect.Value, s *[]string) error {
				afterHandle(strconv.FormatUint(rv.Field(i).Convert(reflect.TypeFor[uint64]()).Interface().(uint64), 10), s)
				return nil
			})
		case reflect.Float32, reflect.Float64:
			ret = append(ret, func(rv reflect.Value, s *[]string) error {
				afterHandle(strconv.FormatFloat(rv.Field(i).Convert(reflect.TypeFor[float64]()).Interface().(float64), 'g', -1, 64), s)
				return nil
			})
		case reflect.Bool:
			ret = append(ret, func(rv reflect.Value, s *[]string) error {
				afterHandle(strconv.FormatBool(rv.Field(i).Interface().(bool)), s)
				return nil
			})
		default:
			return nil, fmt.Errorf("can not handle type of field %s", t.Field(i).Name)
		}
	}
	return ret, nil
}

func WriteCsv[T any](filename string, flag int, mode os.FileMode, values []T, header []string) (*os.File, error) {
	handlers, err := preParseWrite[T]()
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filename, flag, mode)
	if err != nil {
		return nil, err
	}
	w := csv.NewWriter(file)
	defer w.Flush()

	if len(header) != 0 {
		err = w.Write(header)
		if err != nil {
			return nil, err
		}
	}

	for _, value := range values {
		v := reflect.ValueOf(value)
		tmp := make([]string, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			err = handlers[i](v, &tmp)
			if err != nil {
				return nil, err
			}
		}
		err = w.Write(tmp)
		if err != nil {
			return nil, err
		}
	}

	return file, nil
}
