// Package cel implements utilities for working with Asterisk's Channel Event
// Log (CEL) in its CSV form.
package cel

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// An InvalidUnmarshalError describes an invalid argument passed to
// UnmarshalEvent. (The argument to UnmarshalEvent must be a non-nil pointer
// to a struct.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "cel: UnmarshalEvent(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "cel: UnmarshalEvent(non-pointer " + e.Type.String() + ")"
	}
	if e.Type.Elem().Kind() != reflect.Struct {
		return "cel: UnmarshalEvent(pointer to non-struct " + e.Type.String() + ")"
	}
	return "cel: UnmarshalEvent(nil " + e.Type.String() + ")"
}

// UnmarshalEvent takes a record and unmarshals values from that record into
// struct v. Returns an error if v is not a pointer to a struct type.
//
// The struct's exported fields with a struct tag containing a `cel="N"` value
// will be filled with field N from record.
//
// If the struct tag points to an index beyond the length of the given record
// slice, UnmarshalEvent will panic.
//
// Additionally, using a struct tag `cel="N,json"` will take that record
// field, and use encoding/json.Unmarshal to convert its contents to that
// struct field. Adding ",noerror" will allow for json.Unmarshal errors to
// happen silently.
//
// Without ",json" the supported field types are:
//  - string
//  - time.Time (expects Unix time in seconds, or <seconds>.<milliseconds>)
func UnmarshalEvent(record []string, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}
	for i := 0; i < rv.NumField(); i++ {
		err := mapField(record, rv.Field(i), rv.Type().Field(i).Tag.Get("cel"))
		if err != nil {
			return errors.Wrapf(err, "failed to map field %v", rv.Type().Field(i).Name)
		}
	}
	return nil
}

func mapField(record []string, v reflect.Value, tag string) error {
	if tag == "" {
		return nil
	}
	if !v.CanSet() {
		return nil
	}
	tagParts := strings.Split(tag, ",")
	field, err := strconv.ParseInt(tagParts[0], 10, 0)
	if err != nil {
		return errors.Wrapf(err, "bad tag value %q", tag)
	}
	if contains(tagParts, "json") {
		if v.Kind() != reflect.Ptr {
			v = v.Addr()
		}
		err = json.Unmarshal([]byte(record[field]), v.Interface())
		if contains(tagParts, "noerror") {
			return nil
		}
		return err
	}
	if v.Kind() == reflect.String {
		v.SetString(record[int(field)])
	} else if v.Type().PkgPath() == "time" && v.Type().Name() == "Time" {
		t, err := asteriskTime(record[field])
		if err != nil {
			return errors.Wrapf(err, "unable to convert field value %q to time.Time", record[field])
		}
		v.Set(reflect.ValueOf(t))
	} else {
		return fmt.Errorf("type %s not implemented", v.Type())
	}
	return nil
}

func asteriskTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("input is empty string")
	}
	ss := strings.Split(s, ".")
	if len(ss) > 2 {
		return time.Time{}, errors.New("expected at most one period in string")
	}
	sec, err := strconv.ParseInt(ss[0], 10, 0)
	if err != nil {
		return time.Time{}, err
	}
	var nsec int64
	if len(ss) == 2 {
		nsec, err = strconv.ParseInt(ss[1], 10, 0)
		if err != nil {
			return time.Time{}, err
		}
		nsec *= 1000
	}
	return time.Unix(sec, nsec), nil
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
