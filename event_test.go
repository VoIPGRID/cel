package cel_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/VoIPGRID/cel"
	"github.com/matryer/is"
)

func TestUnmarshalEventErrors(t *testing.T) {
	var z *struct{}
	cases := []struct {
		in  interface{}
		err string
	}{
		{nil, "cel: UnmarshalEvent(nil)"},
		{z, "cel: UnmarshalEvent(nil *struct {})"},
		{42, "cel: UnmarshalEvent(non-pointer int)"},
		{new(int), "cel: UnmarshalEvent(pointer to non-struct *int)"},
		{struct{}{}, "cel: UnmarshalEvent(non-pointer struct {})"},

		{
			&struct {
				A string `cel:"b"`
			}{},
			`failed to map field A: bad tag value "b": strconv.ParseInt: parsing "b": invalid syntax`,
		},
		{
			&struct {
				B chan string `cel:"2"`
			}{},
			`failed to map field B: type chan string not implemented`,
		},
	}
	is := is.NewRelaxed(t)
	for _, c := range cases {
		err := cel.UnmarshalEvent([]string{"doesn't matter"}, c.in)
		is.Equal(fmt.Sprint(err), c.err)
	}
}

func TestUnmarshalEvent(t *testing.T) {
	is := is.NewRelaxed(t)
	v := struct {
		unexportedIsIgnored string `cel:"0"`
		NoCELTagIsIgnored   string `json:"does_not_matter"`

		Time   time.Time `cel:"2"`
		Type   string    `cel:"3"`
		Number int       `cel:"0,json"`
		JSON   struct {
			Field int `json:"json_field"`
		} `cel:"1,json"`
	}{}
	err := cel.UnmarshalEvent([]string{"1234", `{"json_field": 42}`, "1530794700.987654", "CHAN_START"}, &v)
	is.NoErr(err)
	is.Equal(v.Time.UTC(), time.Date(2018, 7, 5, 12, 45, 0, 987654000, time.UTC))
	is.Equal(v.Type, "CHAN_START")
	is.Equal(v.Number, 1234)
	is.Equal(v.JSON.Field, 42)
}
