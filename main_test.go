package main

import (
	"fmt"
	"testing"
)

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
	f := contentArray{0, 3, 6, 10}
	var actual string
	actual = fmt.Sprintf("%.1f", f.signalBand(0, 2))
	expected := "7.0" // =1+2+4
	if actual != expected {
		t.Fatalf("got: %v want: %v", actual, expected)
	}
	actual = fmt.Sprintf("%.1f", f.signalBand(1, 3))
	expected = "16.0" // =2+4+10
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
	expected1 := 100
	if actual0 != expected0 {
		t.Fatalf("got: %v want: %v", actual0, expected0)
	}
	if actual1 != expected1 {
		t.Fatalf("got: %v want: %v", actual1, expected1)
	}
}

// Include parse parseConfig(b []byte) test
func Test_readTrace(t *testing.T) {
	filename := "data/20200627_180505.txt"
	usecol := 1
	actualConf, actualCont, err := readTrace(filename, usecol)
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
	for k, v := range actualConf {
		if expectedConfig[k] != v {
			t.Fatalf("got: %v want: %v", v, expectedConfig[k])
		}
	}

	// Content test
	expectedCont := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for i, e := range expectedCont {
		if actualCont[i] != e {
			t.Fatalf("got: %v want: %v\ndump all: %v", actualCont[i], e, actualCont)
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
		"data/20200627_180505.txt",
	}
	bench(b, files)
}
*/
