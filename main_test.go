package main

import (
	"fmt"
	"math"
	"testing"
)

func Test_OutRowString(t *testing.T) {
	o := OutRow{
		Filename: "filename.txt",
		Center:   "Center MHz",
		Datetime: "2016-8-29 17:21:34",
		Fields:   []float64{0, 1, 2, 3},
		Format:   "%f",
	}
	actual := fmt.Sprintf("%s", o)
	expected := "2016-8-29 17:21:34,Center MHz,0.000000,1.000000,2.000000,3.000000"
	if actual != expected {
		t.Fatalf("got: %v want: %v", actual, expected)
	}
}

func Test_db2mw(t *testing.T) {
	var actual []string
	for _, f := range []float64{0, 3, 6, 10} {
		actual = append(actual, fmt.Sprintf("%.3f", db2mw(f)))
	}
	// 10**0.1=1, 10**0.3~=1.99, 10**0.6~=3.98, 10**1=10
	expected := []string{"1.000", "1.995", "3.981", "10.000"}
	for i, e := range expected {
		if actual[i] != e {
			t.Fatalf("got: %v want: %v\ndump all: %v", actual[i], e, actual)
		}
	}
}

func Test_contentArraysignalBand(t *testing.T) {
	f := Trace{Content: []float64{0, 3, 6, 10}} // lambda x: 10^(x/10) => {1, 2, 4, 10}
	var actual string
	actual = fmt.Sprintf("%.1f", f.signalBand(0, 2))
	expected := "8.4" // =10 * log10(1+2+4)
	if actual != expected {
		t.Fatalf("got: %v want: %v", actual, expected)
	}
	actual = fmt.Sprintf("%.1f", f.signalBand(1, 3))
	expected = "12.0" // =10 * log10(2+4+10)
	if actual != expected {
		t.Fatalf("got: %v want: %v", actual, expected)
	}
}

func Test_parseDatetime(t *testing.T) {
	actual := parseDatetime("20200718_190716")
	expected := "2020-07-18 19:07:16"
	if actual != expected {
		t.Fatalf("got: %v want: %v", actual, expected)
	}
}

func Test_parseField(t *testing.T) {
	actual0, actual1, err := parseField("50-100")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	expected0 := 50
	if actual0 != expected0 {
		t.Fatalf("got: %v want: %v", actual0, expected0)
	}
	expected1 := 100
	if actual1 != expected1 {
		t.Fatalf("got: %v want: %v", actual1, expected1)
	}
	// No "-" contain test
	_, _, err = parseField("50")
	if err == nil {
		t.Fatalf("error must be occur %s", err)
	}
	// lower upper test
	_, _, err = parseField("100-50")
	if err == nil {
		t.Fatalf("error must be occur %s", err)
	}
}

func Test_Tracenoisefloor(t *testing.T) {
	c := Trace{Content: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
	actual := c.noisefloor()
	expected := 2.5
	if actual != expected {
		t.Fatalf("got: %v want: %v", actual, expected)
	}
}

// Include parse parseConfig(b []byte) test
func Test_readTrace(t *testing.T) {
	filename := "test/20200627_180505.txt"
	usecol := 1
	actualDf, err := readTrace(filename, usecol)
	if err != nil {
		panic(err)
	}

	// Config test
	expectedConfig := map[string]string{
		":INP:COUP":              "DC",
		":BAND:RES":              "1 Hz",
		":AVER:COUNT":            "10",
		":SWE:POIN":              "11",
		":FREQ:CENT":             "5 MHz",
		":FREQ:SPAN":             "1 MHz",
		":TRAC1:TYPE":            "AVER",
		":INIT:CONT":             "0",
		":FORM":                  "REAL,32",
		":FORM:BORD":             "SWAP",
		":INIT:IMM":              "",
		":POW:ATT":               "0",
		":DISP:WIND:TRAC:Y:RLEV": "-30 dBm",
	}
	for k, v := range actualDf.Config {
		if expectedConfig[k] != v {
			t.Fatalf("got: %v want: %v", v, expectedConfig[k])
		}
	}

	// Content test
	expectedContent := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for i, e := range actualDf.Content {
		if expectedContent[i] != e {
			t.Fatalf("got: %v want: %v\ndump all: %v", actualDf.Content[i], e, actualDf.Content)
		}
	}

	// Index test
	expectedIndex := []float64{4.5, 4.6, 4.7, 4.8, 4.9, 5, 5.1, 5.2, 5.3, 5.4, 5.5}
	for i, a := range actualDf.Index {
		a = math.Round(a*10) / 10 // 123.49999 => 123.4
		if expectedIndex[i] != a {
			t.Fatalf("got: %v want: %v\ndump all: %v", actualDf.Index[i], a, actualDf.Index)
		}
	}
}

/*
func bench(b *testing.B, a []string) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		writeBuffer(a)
	}
}

func Benchmark(b *testing.B) {
	files := []string{
		"test/20200627_180505.txt",
	}
	bench(b, files)
}
*/
